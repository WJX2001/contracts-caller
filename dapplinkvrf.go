package dapplink_vrf

import (
	"context"
	"math/big"
	"sync/atomic"

	common2 "github.com/WJX2001/contract-caller/common"
	"github.com/WJX2001/contract-caller/config"
	"github.com/WJX2001/contract-caller/database"
	"github.com/WJX2001/contract-caller/driver"
	"github.com/WJX2001/contract-caller/event"
	"github.com/WJX2001/contract-caller/synchronizer"
	"github.com/WJX2001/contract-caller/synchronizer/node"
	"github.com/WJX2001/contract-caller/worker"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type DappLinkVrf struct {
	db            *database.DB
	synchronizer  *synchronizer.Synchronizer
	eventsHandler *event.EventsHandler
	worker        *worker.Worker
	shutdown      context.CancelCauseFunc
	stopped       atomic.Bool
}

func NewDappLinkVrf(ctx context.Context, cfg *config.Config, shutdown context.CancelCauseFunc) (*DappLinkVrf, error) {
	// 创建以太坊客户端
	ethClient, err := node.DialEthClient(ctx, cfg.Chain.ChainRpcUrl)
	if err != nil {
		log.Error("new eth client fail", "err", err)
		return nil, err
	}

	// 创建数据库连接
	db, err := database.NewDB(ctx, cfg.MasterDB)
	if err != nil {
		log.Error("new database fail", "err", err)
		return nil, err
	}

	// 3. 创建同步器
	synchronizerS, err := synchronizer.NewSynchronizer(cfg, db, ethClient, shutdown)
	if err != nil {
		log.Error("new synchronizer fail", "err", err)
		return nil, err
	}

	eventConfigm := &event.EventsHandlerConfig{
		DappLinkVrfAddress:        cfg.Chain.DappLinkVrfContractAddress,
		DappLinkVrfFactoryAddress: cfg.Chain.DappLinkVrfFactoryContractAddress,
		LoopInterval:              cfg.Chain.EventInterval,
		StartHeight:               big.NewInt(int64(cfg.Chain.StartingHeight)),
		Epoch:                     500,
	}

	// 4. 创建事件处理器
	eventHandler, err := event.NewEventsHandler(db, eventConfigm, shutdown)
	if err != nil {
		return nil, err
	}

	// 5. 创建驱动引擎
	ethcli, err := driver.EthClientWithTimeout(ctx, cfg.Chain.ChainRpcUrl)
	if err != nil {
		log.Error("new eth client fail", "err", err)
		return nil, err
	}

	callerPrivateKey, _, err := common2.ParseWalletPrivKeyAndContractAddr(
		"ContractCaller",
		cfg.Chain.Mnemonic,
		cfg.Chain.CallerHDPath,
		cfg.Chain.PrivateKey,
		cfg.Chain.DappLinkVrfContractAddress,
		cfg.Chain.Passphrase,
	)

	decg := &driver.DriverEngineConfig{
		ChainClient:               ethcli,
		ChainId:                   big.NewInt(int64(cfg.Chain.ChainId)),
		DappLinkVrfAddress:        common.HexToAddress(cfg.Chain.DappLinkVrfContractAddress),
		CallerAddress:             common.HexToAddress(cfg.Chain.CallerAddress),
		PrivateKey:                callerPrivateKey,
		NumConfirmations:          cfg.Chain.Confirmations,
		SafeAbortNonceTooLowCount: cfg.Chain.SafeAbortNonceTooLowCount,
	}

	eingine, err := driver.NewDriverEngine(ctx, decg)
	if err != nil {
		log.Error("new driver eingine fail", "err", err)
		return nil, err
	}

	workerConfig := &worker.WorkerConfig{
		LoopInterval: cfg.Chain.CallInterval,
	}

	// 6. 创建工作器
	workerProcessor, err := worker.NewWorker(db, eingine, workerConfig, shutdown)
	if err != nil {
		log.Error("new event processor fail", "err", err)
		return nil, err
	}
	// 7. 返回完整的 DappLinkVrf 对象
	return &DappLinkVrf{
		db:            db,
		synchronizer:  synchronizerS,
		eventsHandler: eventHandler,
		worker:        workerProcessor,
		shutdown:      shutdown,
	}, nil
}

// 启动所有服务
func (dvrf *DappLinkVrf) Start(ctx context.Context) error {
	// 1. 启动同步器
	err := dvrf.synchronizer.Start()
	if err != nil {
		return err
	}

	// 2. 启动事件处理器
	err = dvrf.eventsHandler.Start()
	if err != nil {
		return err
	}
	// 3. 启动工作器
	err = dvrf.worker.Start()
	if err != nil {
		return err
	}
	return nil
}

// 当收到关闭信号时，调用 DappLinkVrf.Stop()
func (dvrf *DappLinkVrf) Stop(ctx context.Context) error {
	// 1. 关闭同步器
	err := dvrf.synchronizer.Close()
	if err != nil {
		return err
	}

	// 2. 关闭事件处理器
	err = dvrf.eventsHandler.Close()
	if err != nil {
		return err
	}

	// 3. 关闭工作器
	err = dvrf.worker.Close()
	if err != nil {
		return err
	}
	return nil
}

func (dvrf *DappLinkVrf) Stopped() bool {
	return dvrf.stopped.Load()
}
