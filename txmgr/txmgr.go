package txmgr

import (
	"context"
	"math/big"

	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

/*
合约整体是一个交易发送管理器，用于以太坊或兼容网络上自动重试和确认交易
	- 自动发送交易
	- 动态更新 GAS 价格
	- 处理发送错误
	- 等待交易上链并确认
*/

type UpdateGasPriceFunc = func(ctx context.Context) (*types.Transaction, error)

type SendTransactionFunc = func(ctx context.Context, tx *types.Transaction) error

type Config struct {
	ResubmissionTimeout       time.Duration // 重发交易的时间间隔
	ReceiptQueryInterval      time.Duration // 轮询 receipt 的时间间隔
	NumConfirmations          uint64        // 交易所需确认数
	SafeAbortNonceTooLowCount uint64        // 遇到 nonce too low 错误的容忍次数
}

type TxManager interface {
	// 负责发送交易并等待其确认
	Send(ctx context.Context, updateGasPrice UpdateGasPriceFunc, sendTxn SendTransactionFunc) (*types.Receipt, error)
}

// 提供必要的 RPC 接口，包括获取区块号和获取交易数据
type ReceiptSource interface {
	BlockNumber(ctx context.Context) (uint64, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type SimpleTxManager struct {
	cfg     Config        // 配置
	backend ReceiptSource // 区块链客户端
	l       log.Logger
}

func NewSimpleTxManager(cfg Config, backend ReceiptSource) *SimpleTxManager {
	if cfg.NumConfirmations == 0 {
		panic("txmgr: NumConfirmations cannot be zero")
	}
	return &SimpleTxManager{
		cfg:     cfg,
		backend: backend,
	}
}

func (m *SimpleTxManager) Send(ctx context.Context, updateGasPrice UpdateGasPriceFunc, sendTx SendTransactionFunc) (*types.Receipt, error) {
	// 使用 sync.WaitGroup 来等待所有 goroutine 执行完成，确保函数退出时所有异步操作结束
	var wg sync.WaitGroup
	defer wg.Wait()

	// 创建一个可取消的上下文 ctx, 便于在某些情况下直接终止 goroutine，比如错误发生时
	ctxc, cancel := context.WithCancel(ctx)
	defer cancel()
	// 初始化 sendState 用于追踪 nonceTooLow 错误等状态
	sendState := NewSendState(m.cfg.SafeAbortNonceTooLowCount)
	// 缓冲为1的 channel 用于传回成功上链的回执
	receiptChan := make(chan *types.Receipt, 1)

	// 定义异步发送交易逻辑
	sendTxAsync := func() {
		// 开头注册 Done 保证退出时通知 WaitGroup
		defer wg.Done()

		// 更新 gas 并生成交易
		tx, err := updateGasPrice(ctxc)
		if err != nil {
			if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
				return
			}

			log.Error("ContractsCaller update txn gas price fail", "err", err)
			cancel()
			return
		}

		// 成功生成交易后
		// 提取一些交易参数用于日志
		txHash := tx.Hash()
		nonce := tx.Nonce()
		gasTipCap := tx.GasTipCap()
		gasFeeCap := tx.GasFeeCap()

		log.Debug("ContractsCaller publishing transaction", "txHash", txHash, "nonce", nonce, "gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap)

		// 发送交易 记录错误状态
		err = sendTx(ctxc, tx)
		sendState.ProcessSendError(err)

		if err != nil {
			if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
				return
			}

			log.Error("ContractsCaller unable to publish transaction", "err", err)

			if sendState.ShouldAbortImmediately() {
				cancel()
			}

			return
		}

		log.Debug("ContractsCaller transaction published successfully", "hash", txHash, "nonce", nonce, "gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap)

		// 等待上链确认
		// 调用 waitMined 等待交易上链 并满足指定确认数
		receipt, err := waitMined(
			ctxc, m.backend, tx, m.cfg.ReceiptQueryInterval,
			m.cfg.NumConfirmations, sendState,
		)

		if err != nil {
			log.Debug("ContractsCaller send tx failed", "hash", txHash, "nonce", nonce, "gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap, "err", err)
		}

		if receipt != nil {
			select {
			// 如果收到回执，尝试发送到 receiptChan. 使用 select-default 避免阻塞
			case receiptChan <- receipt:
				log.Trace("ContractsCaller send tx succeeded", "hash", txHash,
					"nonce", nonce, "gasTipCap", gasTipCap,
					"gasFeeCap", gasFeeCap)
			default:
			}
		}
	}

	// 即将启动一个 goroutine, 要计入等待列表
	wg.Add(1)
	// 每次调用 sendTxAsync()前都会加 wg.Add(1) 表示将要启动一个新的发送交易任务
	go sendTxAsync()

	// 启动定时器重试机制
	// 每隔一段时间尝试重新发送交易
	ticker := time.NewTicker(m.cfg.ResubmissionTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 如果不是在等上链 就触发新一轮重发（gas 价格可能已经变化）
			if sendState.IsWaitingForConfirmation() {
				continue
			}
			wg.Add(1)

			go sendTxAsync()

		case <-ctxc.Done():
			return nil, ctxc.Err()
		// 一旦收到回执，说明交易成功，直接返回
		case receipt := <-receiptChan:
			return receipt, nil
		}
	}
}

func WaitMined(
	ctx context.Context,
	backend ReceiptSource,
	tx *types.Transaction,
	queryInterval time.Duration,
	numConfirmations uint64,
) (*types.Receipt, error) {
	return waitMined(ctx, backend, tx, queryInterval, numConfirmations, nil)
}

func waitMined(
	ctx context.Context, // 用于取消/超时控制
	backend ReceiptSource, // 提供链上数据的接口
	tx *types.Transaction, // 要等待上链的交易对象
	queryInterval time.Duration, // 每隔多久轮训一次链上交易回执
	numConfirmations uint64, // 要求的确认区块数
	sendState *SendState, // 状态记录器，用于控制是否继续重发
) (*types.Receipt, error) {
	// 创建轮询定时器

	queryTicker := time.NewTicker(queryInterval)
	defer queryTicker.Stop()

	txHash := tx.Hash()

	for {
		// 查询交易是否已经上链（mined）
		// 如果没有 receipt 说明还没有被打包
		receipt, err := backend.TransactionReceipt(ctx, txHash)
		switch {
		case receipt != nil:
			if sendState != nil {
				sendState.TxMined(txHash)
			}

			// 拿到交易所在的区块高度
			txHeight := receipt.BlockNumber.Uint64()
			// 拿到当前链上最新区块高度
			tipHeight, err := backend.BlockNumber(ctx)

			if err != nil {
				log.Error("ContractsCaller Unable to fetch block number", "err", err)
				break
			}

			log.Trace("ContractsCaller Transaction mined, checking confirmations",
				"txHash", txHash, "txHeight", txHeight,
				"tipHeight", tipHeight,
				"numConfirmations", numConfirmations)

			// 判断是否已经获取足够确认数
			if txHeight+numConfirmations <= tipHeight+1 {
				log.Debug("ContractsCaller Transaction confirmed", "txHash", txHash)
				return receipt, nil
			}

			// 计算还差几个确认才满足条件，打印日志
			confsRemaining := (txHeight + numConfirmations) - (tipHeight + 1)
			log.Info("ContractsCaller Transaction not yet confirmed", "txHash", txHash,
				"confsRemaining", confsRemaining)

		case err != nil:
			log.Trace("ContractsCaller Receipt retrieve failed", "hash", txHash,
				"err", err)

		default:
			// 交易还没有被打包
			if sendState != nil {
				// 通知 SendState 这笔交易还未上链
				sendState.TxNotMined(txHash)
			}
			log.Trace("ContractsCaller Transaction not yet mined", "hash", txHash)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-queryTicker.C:
		}

	}

}

func CalcGasFeeCap(baseFee, gasTipCap *big.Int) *big.Int {
	return new(big.Int).Add(
		gasTipCap,
		new(big.Int).Mul(baseFee, big.NewInt(2)),
	)
}
