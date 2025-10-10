package event

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/WJX2001/contract-caller/common/bigint"
	"github.com/WJX2001/contract-caller/common/tasks"
	"github.com/WJX2001/contract-caller/database"
	"github.com/WJX2001/contract-caller/database/common"
	"github.com/WJX2001/contract-caller/database/worker"
	"github.com/WJX2001/contract-caller/event/contracts"
	"github.com/WJX2001/contract-caller/synchronizer/retry"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var blocksLimit = 10_000

/*
	此文件是 VRF 系统的事件处理器，负责：
		1. 从数据库中读取同步器存储的原始事件日志
		2. 解析 VRF 相关的智能合约事件
		3. 将解析后的事件转换为业务数据
		4. 存储处理结果到数据库
*/

type EventsHandlerConfig struct {
	DappLinkVrfAddress        string        // VRF 主合约地址
	DappLinkVrfFactoryAddress string        // VRF 工厂合约地址
	LoopInterval              time.Duration // 处理循环间隔
	StartHeight               *big.Int      // 起始处理高度
	Epoch                     uint64        // 处理批次大小
}

type EventsHandler struct {
	dappLinkVrf        *contracts.DappLinkVrf        // VRF 合约解析器
	dappLinkVrfFactory *contracts.DappLinkVrfFactory // VRF 工厂合约解析器

	db                  *database.DB         // 数据库连接
	eventsHandlerConfig *EventsHandlerConfig // 配置参数

	latestBlockHeader *common.BlockHeader // 最新处理的区块头

	resourceCtx    context.Context    // 资源上下文
	resourceCancel context.CancelFunc // 资源取消函数
	tasks          tasks.Group        // 任务组管理器
}

func NewEventsHandler(db *database.DB, eventsHandlerConfig *EventsHandlerConfig, shutdown context.CancelCauseFunc) (*EventsHandler, error) {
	// 创建合约解析器
	dappLinkVrf, err := contracts.NewDappLinkVrf()
	if err != nil {
		log.Error("new dapplink vrf fail", "err", err)
		return nil, err
	}

	dappLinkVrfFactory, err := contracts.NewDappLinkVrfFactory()
	if err != nil {
		log.Error("new dapplink vrf factory fail", "err", err)
		return nil, err
	}
	// 初始化事件处理器
	ltBlockHeader, err := db.EventBlocks.LatestEventBlockHeader()
	if err != nil {
		log.Error("fetch latest block header fail", "err", err)
		return nil, err
	}

	resCtx, resCancel := context.WithCancel(context.Background())

	return &EventsHandler{
		dappLinkVrf:         dappLinkVrf,
		dappLinkVrfFactory:  dappLinkVrfFactory,
		db:                  db,
		eventsHandlerConfig: eventsHandlerConfig,
		latestBlockHeader:   ltBlockHeader,
		resourceCtx:         resCtx,
		resourceCancel:      resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in bridge processor: %w", err))
		}},
	}, nil
}

// 启动方法
func (eh *EventsHandler) Start() error {
	log.Info("starting event processor...")
	tickerEventWorker := time.NewTicker(eh.eventsHandlerConfig.LoopInterval)
	eh.tasks.Go(func() error {
		for range tickerEventWorker.C {
			/*
				定期执行：
					1. 处理区块链事件
					2. 解析 VRF 相关事件
					3. 存储事件数据
			*/
			log.Info("start parse event logs")
			err := eh.processEvent()
			if err != nil {
				log.Info("process event error", "err", err)
				return err
			}
		}
		return nil
	})
	return nil
}

func (eh *EventsHandler) Close() error {
	eh.resourceCancel()    // 取消上下文
	return eh.tasks.Wait() // 等待所有任务完成
}

/*
1. 从数据库中读取同步器存储的原始事件
2. 解析 VRF 相关的智能合约事件
3. 将解析后的事件转换为业务数据
4. 批量存储处理结果到数据库
*/
func (eh *EventsHandler) processEvent() error {
	lastBlockNumber := eh.eventsHandlerConfig.StartHeight
	if eh.latestBlockHeader != nil {
		lastBlockNumber = eh.latestBlockHeader.Number
	}
	log.Info("process event latest block number", "lastBlockNumber", lastBlockNumber)
	latestHeaderScope := func(db *gorm.DB) *gorm.DB {
		// 开启一个新的查询，不被之前的查询条件干扰
		newQuery := db.Session(&gorm.Session{NewDB: true})
		// 指定模型表为 BlockHeader，添加条件 number > lastBlockNumber
		// 表示一个子查询构造器，选择 number 大于 lastBlockNumber 的记录
		headers := newQuery.Model(common.BlockHeader{}).Where("number >= ?", lastBlockNumber)
		/*
			SELECT * FROM block_headers
			WHERE number = (
			  SELECT MAX(number)
			  FROM (
			    SELECT number
			    FROM block_headers
			    WHERE number > 100
			    ORDER BY number ASC
			    LIMIT 50
			  ) AS block_numbers
			);
		*/
		return db.Where("number = (?)", newQuery.Table("(?) as block_numbers", headers.Order("number ASC").Limit(blocksLimit)).Select("MAX(number)"))
	}

	if latestHeaderScope == nil {
		return nil
	}

	latestBlockHeader, err := eh.db.Blocks.BlockHeaderWithScope(latestHeaderScope)
	if err != nil {
		log.Error("get latest block header with scope fail", "err", err)
		return err
	} else if latestBlockHeader == nil {
		log.Debug("no new block for process event")
		return nil
	}

	// 生成事件区块记录的逻辑
	fromHeight, toHeight := new(big.Int).Add(lastBlockNumber, bigint.One), latestBlockHeader.Number
	// 第二个参数 预分配容量
	eventBlocks := make([]worker.EventBlocks, 0, toHeight.Uint64()-fromHeight.Uint64())
	// 逐个查询区块头
	for index := fromHeight.Uint64(); index < toHeight.Uint64(); index++ {
		blockHeader, err := eh.db.Blocks.BlockHeaderByNumber(big.NewInt(int64(index)))
		if err != nil {
			return err
		}
		// 将区块头信息转换为 事件区块记录
		/*
			记录作用：
				1. 进度跟踪：记录已处理的区块
				2. 去重机制：避免重复处理相同区块
				3. 状态恢复：支持从任意点恢复处理
		*/
		evBlock := worker.EventBlocks{
			GUID:       uuid.New(),
			Hash:       blockHeader.Hash,
			ParentHash: blockHeader.ParentHash,
			Number:     blockHeader.Number,
			Timestamp:  blockHeader.Timestamp,
		}
		eventBlocks = append(eventBlocks, evBlock)
	}

	// 合约事件处理
	/*
				数据库原始事件 → 合约解析器 → 业务数据
		     ↓              ↓           ↓
		ContractEvent → DappLinkVrf → RequestSend
		              → DappLinkVrf → FillRandomWords
		              → DappLinkVrfFactory → PoxyCreated
	*/

	// 主合约事件处理
	requestSentList, fillRandomWordList, err := eh.dappLinkVrf.ProcessDappLinkVrfEvent( // 随机数请求，随机数回填
		eh.db,
		eh.eventsHandlerConfig.DappLinkVrfAddress,
		fromHeight,
		toHeight,
	)

	if err != nil {
		log.Error("process dapplink vrf event fail", "err", err)
		return err
	}

	// 工厂合约事件处理
	proxyCreatedList, err := eh.dappLinkVrfFactory.ProcessDappLinkVrfFactoryEvent(
		eh.db,
		eh.eventsHandlerConfig.DappLinkVrfFactoryAddress,
		fromHeight,
		toHeight,
	)

	if err != nil {
		return err
	}

	// 重试策略配置
	/*
		处理临时性数据库连接问题
		避免因网络抖动导致的数据丢失
		通过指数退避减少对数据库压力
	*/
	retryStrategy := &retry.ExponentialStrategy{
		Min:       1000,
		Max:       20_000,
		MaxJitter: 250,
	}

	if _, err := retry.Do[interface{}](eh.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
		// 数据库事务处理
		if err := eh.db.Transaction(func(tx *database.DB) error {
			// 存储随机数请求
			if len(requestSentList) > 0 {
				err := eh.db.RequestSend.StoreRequestSend(requestSentList)
				if err != nil {
					log.Error("store request send fail", "err", err)
					return err
				}
			}

			// 存储随机数回填
			if len(fillRandomWordList) > 0 {
				err := eh.db.FillRandomWords.StoreFillRandomWords(fillRandomWordList)
				if err != nil {
					log.Error("store fill random words fail", "err", err)
					return err
				}
			}

			// 存储代理创建记录
			if len(proxyCreatedList) > 0 {
				err := eh.db.PoxyCreated.StorePoxyCreated(proxyCreatedList)
				if err != nil {
					log.Error("store proxy created fail", "err", err)
					return err
				}
			}

			// 存储事件区块记录
			if len(eventBlocks) > 0 {
				err := eh.db.EventBlocks.StoreEventBlocks(eventBlocks)
				if err != nil {
					log.Error("store event blocks fail", "err", err)
					return err
				}
			}
			return nil
		}); err != nil {
			log.Debug("unable to persist batch", err)
			return nil, fmt.Errorf("unable to persist batch: %w", err)
		}
		return nil, nil
	}); err != nil {
		return err
	}
	// 状态更新
	eh.latestBlockHeader = latestBlockHeader
	return nil
}
