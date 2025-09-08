package node

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/WJX2001/contract-caller/common/global_const"
	"github.com/WJX2001/contract-caller/synchronizer/retry"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

/*
	- 封装以太坊 RPC 客户端
	- 提供区块头、交易、日志等数据查询接口
	- 实现连接重试和超时处理
	- 与以太坊节点进行 RPC 通信，提供统一的接口来获取区块链数据
*/

const (
	defaultDialTimeout    = 5 * time.Second
	defaultDialAttempts   = 5
	defaultRequestTimeout = 100 * time.Second
)

type EthClient interface {
	// 区块头相关
	BlockHeaderByNumber(*big.Int) (*types.Header, error)  // 根据区块号获取区块头
	LatestSafeBlockHeader() (*types.Header, error)        // 获取最新的安全区块头
	LatestFinalizedBlockHeader() (*types.Header, error)   // 获取最新的最终确认区块头
	BlockHeaderByHash(common.Hash) (*types.Header, error) // 根据区块哈希获取区块头
	// 批量区块头查询，支持批量获取指定范围内的区块头，对 Polygon 链使用并发请求优化，对其他链使用标准的批量 RPC 调用
	BlockHeadersByRange(*big.Int, *big.Int, uint) ([]types.Header, error)

	// 交易查询（根据交易哈希获取交易详情）
	TxByHash(common.Hash) (*types.Transaction, error)

	// 获取指定地址在指定区块的存储哈希
	StorageHash(common.Address, *big.Int) (common.Hash, error)
	// 事件日志过滤
	// 支持按区块范围、地址、主题过滤事件日志
	// 使用批量 RPC 调用同时获取日志和对应的区块头
	// 返回自定义的 Logs 结构，包含日志和对应的区块头
	FilterLogs(ethereum.FilterQuery) (Logs, error)

	Close()
}

type clnt struct {
	rpc RPC
}

// 客户端连接
// 支持 URL 可用性检查
// 封装底层 RPC 客户端
func DialEthClient(ctx context.Context, rpcUrl string) (EthClient, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()
	bOff := retry.Exponential()
	rpcClient, err := retry.Do(ctx, defaultDialAttempts, bOff, func() (*rpc.Client, error) {
		if !IsURLAvailable(rpcUrl) {
			return nil, fmt.Errorf("address unavailable (%s)", rpcUrl)
		}

		client, err := rpc.DialContext(ctx, rpcUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to dial address (%s): %w", rpcUrl, err)
		}

		return client, nil
	})

	if err != nil {
		return nil, err
	}

	return &clnt{rpc: NewRPC(rpcClient)}, nil
}

// 根据区块哈希获取区块头
func (c *clnt) BlockHeaderByHash(hash common.Hash) (*types.Header, error) {
	// 创建一个带超时的 context, 超时时间是 defaultRequestTimeout
	// 确保函数返回时取消 context, 释放资源，避免 RPC 调用卡死
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()
	// 区块头变量
	var header *types.Header

	err := c.rpc.CallContext(ctxwt, &header, "eth_getBlockByHash", hash, false)
	if err != nil {
		return nil, err
	} else if header == nil {
		return nil, ethereum.NotFound
	}

	if header.Hash() != hash {
		return nil, errors.New("header mismatch")
	}

	return header, nil
}

// 根据区块号获取区块头
func (c *clnt) BlockHeaderByNumber(number *big.Int) (*types.Header, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	var header *types.Header
	err := c.rpc.CallContext(ctxwt, &header, "eth_getBlockByNumber", toBlockNumArg(number), false)
	if err != nil {
		log.Fatalln("Call eth_getBlockByNumber method fail", "err", err)
		return nil, err
	} else if header == nil {
		log.Println("header not found")
		return nil, ethereum.NotFound
	}

	return header, nil
}

/*
根据区块高度范围，批量获取这一段的区块头信息
如果只要一个区块 -> 直接调用 BlockHeaderByNumber
如果是普通链，以太坊、BSC等，用 BatchCallContext 一次性批量请求，效率高
如果是 Polygon链，每组最多100个区块，每个区块单独 RPC 请求，避免节点拒绝大批量请求
最后整理结果，返回结果
*/
func (c *clnt) BlockHeadersByRange(startHeight, endHeight *big.Int, chainId uint) ([]types.Header, error) {
	if startHeight.Cmp(endHeight) == 0 {
		header, err := c.BlockHeaderByNumber(startHeight)
		if err != nil {
			return nil, err
		}
		return []types.Header{*header}, nil
	}

	count := new(big.Int).Sub(endHeight, startHeight).Uint64() + 1
	headers := make([]types.Header, count)
	batchElems := make([]rpc.BatchElem, count)

	// 普通链，非 Polygon
	if chainId != uint(global_const.PolygonChainId) {
		for i := uint64(0); i < count; i++ {
			height := new(big.Int).Add(startHeight, new(big.Int).SetUint64(i))
			batchElems[i] = rpc.BatchElem{
				Method: "eth_getBlockByNumber",
				Args:   []interface{}{toBlockNumArg(height), false},
				Result: &headers[i],
			}
		}

		ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
		defer cancel()

		err := c.rpc.BatchCallContext(ctxwt, batchElems)
		if err != nil {
			return nil, err
		}
	} else {
		groupSize := 100
		// 等待一组 goroutine 全部执行完成
		var wg sync.WaitGroup
		numGroups := (int(count)-1)/groupSize + 1
		wg.Add(numGroups)

		// 对 polygon 链做了特殊处理，不能一次性批量请求太多区块，所以分组处理，每组做多100个
		for i := 0; i < int(count); i += groupSize {
			start := i
			end := i + groupSize - 1
			if end > int(count) {
				end = int(count) - 1
			}

			go func(start, end int) {
				defer wg.Done()
				for j := start; j <= end; j++ {
					ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
					defer cancel()
					height := new(big.Int).Add(startHeight, new(big.Int).SetUint64(uint64(j)))
					batchElems[j] = rpc.BatchElem{
						Method: "eth_getBlockByNumber",
						Result: new(types.Header),
						Error:  nil,
					}
					header := new(types.Header)
					batchElems[j].Error = c.rpc.CallContext(ctxwt, header, "eth_getBlockByNumber", toBlockNumArg(height), false)
					batchElems[j].Result = header
				}
			}(start, end)
		}
		// 等待所有的 goroutine 完成
		wg.Wait()
	}

	size := 0
	for i, batchElem := range batchElems {
		header, ok := batchElem.Result.(*types.Header)
		if !ok {
			return nil, fmt.Errorf("unable to transform rpc response %v into utils.Header", batchElem.Result)
		}

		headers[i] = *header

		size = size + 1
	}

	headers = headers[:size]
	return headers, nil
}

type Logs struct {
	Logs          []types.Log
	ToBlockHeader *types.Header
}

func (c *clnt) FilterLogs(query ethereum.FilterQuery) (Logs, error) {
	arg, err := toFilterArg(query)
	if err != nil {
		return Logs{}, err
	}

	var logs []types.Log
	var header types.Header
	batchElems := make([]rpc.BatchElem, 2)
	batchElems[0] = rpc.BatchElem{Method: "eth_getBlockByNumber", Args: []interface{}{toBlockNumArg(query.ToBlock), false}, Result: &header}
	batchElems[1] = rpc.BatchElem{Method: "eth_getLogs", Args: []interface{}{arg}, Result: &logs}

	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()
	err = c.rpc.BatchCallContext(ctxwt, batchElems)

	if err != nil {
		return Logs{}, err
	}

	if batchElems[0].Error != nil {
		return Logs{}, fmt.Errorf("unable to query for the `FilterQuery#ToBlock` header: %w", batchElems[0].Error)
	}

	if batchElems[1].Error != nil {
		return Logs{}, fmt.Errorf("unable to query logs: %w", batchElems[1].Error)
	}

	return Logs{Logs: logs, ToBlockHeader: &header}, nil

}

// 获取最新的安全区块头
func (c *clnt) LatestSafeBlockHeader() (*types.Header, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	var header *types.Header

	err := c.rpc.CallContext(ctxwt, &header, "eth_getBlockByNumber", "safe", false)
	if err != nil {
		return nil, err
	} else if header == nil {
		return nil, ethereum.NotFound
	}
	return header, nil
}

// 获取最新的最终确认区块头
func (c *clnt) LatestFinalizedBlockHeader() (*types.Header, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	var header *types.Header
	err := c.rpc.CallContext(ctxwt, &header, "eth_getBlockByNumber", "finalized", false)
	if err != nil {
		return nil, err
	} else if header == nil {
		return nil, ethereum.NotFound
	}

	return header, nil
}

// 存储证明，获取指定地址在指定区块的存储哈希
func (c *clnt) StorageHash(address common.Address, blockNumber *big.Int) (common.Hash, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	proof := struct{ StorageHash common.Hash }{}
	err := c.rpc.CallContext(ctxwt, &proof, "eth_getProof", address, nil, toBlockNumArg(blockNumber))
	if err != nil {
		return common.Hash{}, err
	}

	return proof.StorageHash, nil
}

func (c *clnt) TxByHash(hash common.Hash) (*types.Transaction, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	var tx *types.Transaction
	err := c.rpc.CallContext(ctxwt, &tx, "eth_getTransactionByHash", hash)
	if err != nil {
		return nil, err
	} else if tx == nil {
		return nil, ethereum.NotFound
	}

	return tx, nil
}

func (c *clnt) Close() {
	c.rpc.Close()
}

type RPC interface {
	// 关闭连接
	Close()
	// 发起一次 RPC 调用
	CallContext(ctx context.Context, result any, method string, args ...any) error
	// 一次性批量发器多个 RPC 请求（提高效率）
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
}

type rpcClient struct {
	rpc *rpc.Client
}

func NewRPC(client *rpc.Client) RPC {
	return &rpcClient{client}
}

func (c *rpcClient) Close() {
	c.rpc.Close()
}

func (c *rpcClient) CallContext(ctx context.Context, result any, method string, args ...any) error {
	err := c.rpc.CallContext(ctx, result, method, args...)
	return err
}

func (c *rpcClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	err := c.rpc.BatchCallContext(ctx, b)
	return err
}

// 将区块号转换为 RPC 参数格式
func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}

	if number.Sign() >= 0 {
		return hexutil.EncodeBig(number)
	}

	return rpc.BlockNumber(number.Int64()).String()
}

// 将过滤查询转换为 RPC 参数格式，把 Go 里定义的以太坊日志查询条件转换成 JSON-RPC 调用
func toFilterArg(q ethereum.FilterQuery) (interface{}, error) {
	// 创建一个 map，先把 address 合约地址过滤条件 和 topics 事件主题过滤条件放进去
	arg := map[string]interface{}{"address": q.Addresses, "topics": q.Topics}
	// 如果指定了区块哈希,说明要查询 某个具体区块的日志，这时把 blockHash 加到参数里
	if q.BlockHash != nil {
		arg["blockHash"] = *q.BlockHash
		// 不能同时指定 blockHash 和 fromBlock 这时eth 的 JSON-RPC 就是这样规定的
		if q.FromBlock != nil || q.ToBlock != nil {
			return nil, errors.New("cannot specify both BlockHash and From/ToBlock")
		}
	} else { // 如果没有指定区块哈希
		if q.FromBlock == nil {
			// 默认为创世区块
			arg["fromBlock"] = "0x0"
		} else {
			arg["fromBlock"] = toBlockNumArg(q.FromBlock)
		}
		arg["toBlock"] = toBlockNumArg(q.ToBlock)
	}
	return arg, nil
}

func IsURLAvailable(address string) bool {
	u, err := url.Parse(address)
	if err != nil {
		return false
	}

	addr := u.Host
	if u.Port() == "" {
		switch u.Scheme {
		case "http", "ws":
			addr += ":80"
		case "https", "wss":
			addr += ":443"
		default:
			return true
		}
	}

	// 尝试使用 TCP连接 addr 域名+端口，超时时间5秒
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return false
	}
	err = conn.Close()
	if err != nil {
		return false
	}
	return true
}
