package node

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

/*
	- 封装以太坊 RPC 客户端
	- 提供区块头、交易、日志等数据查询接口
	- 实现连接重试和超时处理
	- 与以太坊节点进行 RPC 通信，提供统一的接口来获取区块链数据
*/

const defaultDialTimeout = 5 * time.Second

type EthClient interface {
	// 区块头相关
	BlockHeaderByNumber(*big.Int) (*types.Header, error)
	LatestSafeBlockHeader() (*types.Header, error)
	LatestFinalizedBlockHeader() (*types.Header, error) // 最终确定的区块头
	BlockHeaderByHash(common.Hash) (*types.Header, error)
	BlockHeadersByRange(*big.Int, *big.Int, uint) ([]types.Header, error)

	// 交易相关
	TxByHash(common.Hash) (*types.Transaction, error)

	// 存储和日志相关
	StorageHash(common.Address, *big.Int) (common.Hash, error)
	FilterLogs(ethereum.FilterQuery) (Logs, error)
}

// 客户端连接
func DialEthClient(ctx context.Context, rpcUrl string) (EthClient, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

}

type Logs struct {
	Logs          []types.Log
	ToBlockHeader *types.Header
}
