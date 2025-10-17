package synchronizer

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/WJX2001/contract-caller/common/tasks"
	"github.com/WJX2001/contract-caller/config"
	"github.com/WJX2001/contract-caller/database"
	common2 "github.com/WJX2001/contract-caller/database/common"
	"github.com/WJX2001/contract-caller/database/event"
	"github.com/WJX2001/contract-caller/database/utils"
	"github.com/WJX2001/contract-caller/synchronizer/node"
	"github.com/WJX2001/contract-caller/synchronizer/retry"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

/*

 */

type Synchronizer struct {
	ethClient node.EthClient // 以太坊客户端
	db        *database.DB   // 数据库连接

	loopInterval     time.Duration         // 同步循环间隔
	headerBufferSize uint64                // 批量处理大小
	headerTraversal  *node.HeaderTraversal // 区块头遍历器

	headers      []types.Header // 待处理的区块头缓存
	latestHeader *types.Header  // 最新区块头

	startHeight       *big.Int            // 起始高度
	confirmationDepth *big.Int            // 确认深度
	chainCfg          *config.ChainConfig // 链配置

	resourceCtx    context.Context    // 资源上下文
	resourceCancel context.CancelFunc // 取消函数
	tasks          tasks.Group        // 任务组
}

// 创建区块同步器，从链上拉区块头与事件写库
func NewSynchronizer(cfg *config.Config, db *database.DB, client node.EthClient, shutdown context.CancelCauseFunc) (*Synchronizer, error) {

	// 从数据库获取最后同步的区块头
	// 如果存在，从该区块继续同步，如果不存在且配置了起始高度，从配置的起始高度开始，否则从头开始同步
	latestHeader, err := db.Blocks.LatestBlockHeader()
	if err != nil {
		return nil, err
	}

	var fromHeader *types.Header
	if latestHeader != nil {
		// 指定高度同步
		// 当数据库为空的时候，从配置的起始高度开始，适用于首次部署或数据重置场景
		log.Info("sync detected last indexed block", "number", latestHeader.Number, "hash", latestHeader.Hash)
		fromHeader = latestHeader.RLPHeader.Header()
	} else if cfg.Chain.BlockStep > 0 {
		// 从头开始同步
		log.Info("no sync indexed state starting from supplied ethereum height", "height", cfg.Chain.StartingHeight)
		header, err := client.BlockHeaderByNumber(big.NewInt(int64(cfg.Chain.StartingHeight)))
		if err != nil {
			return nil, fmt.Errorf("could not fetch starting block header: %w", err)
		}
		fromHeader = header
	} else {
		log.Info("no eth wallet indexed state")
	}

	headerTraversal := node.NewHeaderTraversal(client, fromHeader, big.NewInt(0), cfg.Chain.ChainId)

	resCtx, resCancel := context.WithCancel(context.Background())
	return &Synchronizer{
		loopInterval:     time.Duration(cfg.Chain.MainLoopInterval) * time.Second,
		headerBufferSize: uint64(cfg.Chain.BlockStep),
		headerTraversal:  headerTraversal,
		ethClient:        client,
		latestHeader:     fromHeader,
		db:               db,
		chainCfg:         &cfg.Chain,
		resourceCtx:      resCtx,
		resourceCancel:   resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in Synchronizer: %w", err))
		}},
	}, nil
}

// 启动逻辑
func (syncer *Synchronizer) Start() error {
	tickerSyncer := time.NewTicker(time.Second * 3)
	syncer.tasks.Go(func() error {
		for range tickerSyncer.C {
			/*
				每3秒执行一次
				1. 获取区块头
				2. 处理区块数据
				3. 存储到数据库
			*/
			if len(syncer.headers) > 0 {
				// 判断是否有上一次未处理完的 headers
				// syncer.headers 是一个缓存区块头数组，如果上一次同步失败、没有清空，他会在下一轮重试（避免丢数据）
				// 否则就去链上拉新的区块头
				log.Info("retrying previous batch")
			} else {
				newHeaders, err := syncer.headerTraversal.NextHeaders(uint64(syncer.chainCfg.BlockStep))
				if err != nil {
					// 如果 RPC 调用出错，就跳过
					log.Error("error querying for headers", "err", err)
					continue
				} else if len(newHeaders) == 0 {
					// 如果没有新块，说明同步器已经到 链头
					log.Warn("no new headers. syncer at head?")
				} else {
					// 将新 headers 存入 syncer.headers 以便后续处理
					syncer.headers = newHeaders
				}
				// 获取最新的区块头
				latestHeader := syncer.headerTraversal.LatestHeader()
				if latestHeader != nil {
					log.Info("Latest header", "latestHeader Number", latestHeader.Number)
				}
			}

			err := syncer.processBatch(syncer.headers, syncer.chainCfg)
			if err == nil {
				syncer.headers = nil
			}
		}
		return nil
	})
	return nil
}

/*
批量处理区块数据
对一批区块头做一次：抽取日志 -> 构建区块头结构 -> 构造合约事件 -> 持久化到数据库
*/
func (syncer *Synchronizer) processBatch(headers []types.Header, chainCfg *config.ChainConfig) error {
	if len(headers) == 0 {
		return nil
	}

	firstHeader, lastHeader := headers[0], headers[len(headers)-1]
	log.Info("extracting batch", "size", len(headers), "startBlock", firstHeader.Number.String(), "endBlock", lastHeader.Number.String())

	headerMap := make(map[common.Hash]*types.Header, len(headers))
	for i := range headers {
		header := headers[i]
		headerMap[header.Hash()] = &header
	}

	// 获取监听地址列表
	// 动态地址列表：从数据库获取需要监听的合约地址
	// VRF：这些地址是 VRF 代理合约的地址
	// 过滤优化： 只监听相关合约的事件，减少数据量
	addressList, err := syncer.db.PoxyCreated.QueryPoxyCreatedAddressList()
	if err != nil {
		log.Error("QueryPoxyCreatedAddressList fail", "err", err)
		return err
	}

	filterQuery := ethereum.FilterQuery{
		FromBlock: firstHeader.Number,
		ToBlock:   lastHeader.Number,
		Addresses: addressList,
	}

	// 过滤事件日志
	logs, err := syncer.ethClient.FilterLogs(filterQuery)
	if err != nil {
		log.Info("failed to extract logs", "err", err)
		return err
	}

	// 数据一致性验证
	if logs.ToBlockHeader.Number.Cmp(lastHeader.Number) != 0 {
		return fmt.Errorf("mismatch in FilterLog#ToBlock number")
	} else if logs.ToBlockHeader.Hash() != lastHeader.Hash() {
		return fmt.Errorf("mismatch in FitlerLog#ToBlock block hash")
	}

	if len(logs.Logs) > 0 {
		log.Info("detected logs", "size", len(logs.Logs))
	}

	// 区块头数据转换
	// 把 types.Header 转换成项目内部 common2.BlockHeader 结构，准备写入 DB
	blockHeaders := make([]common2.BlockHeader, len(headers))
	for i := range headers {
		if headers[i].Number == nil {
			continue
		}
		bHeader := common2.BlockHeader{
			Hash:       headers[i].Hash(),
			ParentHash: headers[i].ParentHash,
			Number:     headers[i].Number,
			Timestamp:  headers[i].Time,
			RLPHeader:  (*utils.RLPHeader)(&headers[i]),
		}
		blockHeaders = append(blockHeaders, bHeader)
	}

	// 把 RPC 返回的 每个 Log 变成 event.ContractEvent 并把区块时间戳从 headerMap 中取出赋值给事件
	chainContractEvent := make([]event.ContractEvent, len(logs.Logs))
	for i := range logs.Logs {
		logEvent := logs.Logs[i]
		if _, ok := headerMap[logEvent.BlockHash]; !ok {
			continue
		}
		timestamp := headerMap[logEvent.BlockHash].Time
		chainContractEvent[i] = event.ContractEventFromLog(&logs.Logs[i], timestamp)
	}

	// 使用指数退避重试策略尝试做一次事务性的持久化
	// StoreBlockHeaders 和 StoreContractEvents 都在同一事物内
	/*
		最小等待 1s，最大等待20s 抖动 250ms
	*/
	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
	if _, err := retry.Do[interface{}](syncer.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
		// 每次重试内调用 Transaction 执行 DB操作 成功则提交 失败则返回 error
		if err := syncer.db.Transaction(func(tx *database.DB) error {
			if err := tx.Blocks.StoreBlockHeaders(blockHeaders); err != nil {
				return err
			}

			if err := tx.ContractEvent.StoreContractEvents(chainContractEvent); err != nil {
				return err
			}
			return nil
		}); err != nil {
			log.Info("unable to persist batch", err)
			return nil, fmt.Errorf("unable to persist batch: %w", err)
		}
		return nil, nil
	}); err != nil {
		return err
	}
	return nil
}

func (syncer *Synchronizer) Close() error {
	return nil
}
