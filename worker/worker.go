package worker

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/WJX2001/contract-caller/common/tasks"
	"github.com/WJX2001/contract-caller/database"
	"github.com/WJX2001/contract-caller/driver"
	"github.com/ethereum/go-ethereum/log"
)

type WorkerConfig struct {
	LoopInterval time.Duration
}

type Worker struct {
	workerConfig   *WorkerConfig
	db             *database.DB
	deg            *driver.DriverEngine
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewWorker(db *database.DB, deg *driver.DriverEngine, workerConfig *WorkerConfig, shutdown context.CancelCauseFunc) (*Worker, error) {
	resCtx, resCancel := context.WithCancel(context.Background())

	return &Worker{
		db:             db,
		deg:            deg,
		workerConfig:   workerConfig,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in bridge processor: %w", err))
		}},
	}, nil
}

func (wk *Worker) Start() error {
	log.Info("starting worker processor...")
	tickerEventWorker := time.NewTicker(wk.workerConfig.LoopInterval) // 每隔5s 执行一次 ticker
	wk.tasks.Go(func() error {
		for range tickerEventWorker.C {
			log.Info("start handler random for vrf")
			// 每隔一段时间 会发一笔交易更新一下ProcessCallerVrf
			err := wk.ProcessCallerVrf()
			if err != nil {
				log.Error("process caller vrf fail", "err", err)
				return err
			}
		}
		return nil
	})
	return nil
}

// 组织数据通过 FulfillRandomWords 调用合约的方法，将数据写入合约

func (wk *Worker) ProcessCallerVrf() error {
	// 获取 RequestSent 合约事件
	var randomList []*big.Int

	randomList = append(randomList, big.NewInt(1000))
	randomList = append(randomList, big.NewInt(1001))
	randomList = append(randomList, big.NewInt(1002))

	txReceipt, err := wk.deg.FulfillRandomWords(big.NewInt(22222222), randomList)
	if err != nil {
		log.Error("fulfill random words fail", "err", err)
		return err
	}
	if txReceipt.Status == 1 {
		log.Info("call contract success ......")
	}
	return nil

}

func (wk *Worker) Close() error {
	wk.resourceCancel()
	return wk.tasks.Wait()
}
