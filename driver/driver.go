package driver

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/WJX2001/contract-caller/bindings"
	"github.com/WJX2001/contract-caller/txmgr"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// TODO: 此文件封装与 VRF 合约的底层交互逻辑：合约调用、构造交易、动态 gas 设置、重试发送等链上交互能力
// 这是链下服务调用 VRF 合约的核心组件

/**
一个名为DriverEingine的结构体，主要作用是与部署在链上的 DappLink VRF 合约进行交互，并通过 txmgr.TxManager 管理交易的生命周期
	- 如：构建、发送、重发、确认等，是链下服务与合约之间通信的桥梁
	- 封装并管理链上 DappLink VRF 合约的调用逻辑。
	- 统一构建、重发、确认交易的流程，防止交易失败或者 gas 设置错误
	- 兼容旧链上不支持 EIP-1559 的情况
*/

var (
	errMaxPriorityFeePerGasNotFound = errors.New(
		"Method eth_maxPriorityFeePerGas not found",
	)

	FallbackGasTipCap = big.NewInt(1500000000)
)

type DriverEngineConfig struct {
	ChainClient               *ethclient.Client // 链客户端
	ChainId                   *big.Int          // 链ID
	DappLinkVrfAddress        common.Address    // DappLinkVRF 合约地址
	CallerAddress             common.Address    // 发交易的地址
	PrivateKey                *ecdsa.PrivateKey // CallerAddress 和 PrivateKey 是一一对应的
	NumConfirmations          uint64            // 交易确认区块数
	SafeAbortNonceTooLowCount uint64            // nonce 错误重试上限
}

type DriverEngine struct {
	Ctx                    context.Context
	Cfg                    *DriverEngineConfig
	DappLinkVrfContract    *bindings.DappLinkVRF
	RawDappLinkVrfContract *bind.BoundContract
	DappLinkVrfContractAbi *abi.ABI
	TxMgr                  txmgr.TxManager // 交易管理器
	cancel                 func()
	wg                     sync.WaitGroup
}

func NewDriverEngine(ctx context.Context, cfg *DriverEngineConfig) (*DriverEngine, error) {
	_, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	// 解析 ABI JSON
	dappLinkVrfContract, err := bindings.NewDappLinkVRF(cfg.DappLinkVrfAddress, cfg.ChainClient)
	if err != nil {
		log.Error("new dapplink vrf fail", "err", err)
		return nil, err
	}

	// 解析 ABI JSON
	parsed, err := abi.JSON(strings.NewReader(bindings.DappLinkVRFMetaData.ABI))
	if err != nil {
		log.Error("parsed abi fail", "err", err)
		return nil, err
	}

	dappLinkVrfContractAbi, err := bindings.DappLinkVRFFactoryMetaData.GetAbi()
	if err != nil {
		log.Error("get dapplink vrf meta data fail", "err", err)
		return nil, err
	}

	// 构建 RAW 合约绑定器
	rawDappLinkVrfContract := bind.NewBoundContract(cfg.DappLinkVrfAddress, parsed, cfg.ChainClient, cfg.ChainClient, cfg.ChainClient)

	txManagerConfig := txmgr.Config{
		ResubmissionTimeout:       time.Second * 5,
		ReceiptQueryInterval:      time.Second,
		NumConfirmations:          cfg.NumConfirmations,
		SafeAbortNonceTooLowCount: cfg.SafeAbortNonceTooLowCount,
	}

	// 初始化交易管理器
	txManager := txmgr.NewSimpleTxManager(txManagerConfig, cfg.ChainClient)

	return &DriverEngine{
		Ctx:                    ctx,
		Cfg:                    cfg,
		DappLinkVrfContract:    dappLinkVrfContract,
		RawDappLinkVrfContract: rawDappLinkVrfContract,
		DappLinkVrfContractAbi: dappLinkVrfContractAbi,
		TxMgr:                  txManager,
		cancel:                 cancel,
	}, nil
}

// 动态更新 Gas Price 方法
// 构建一个新的交易，复用旧交易的数据（如 nonce 和 data） 用于重新估算 gas

func (de *DriverEngine) UpdateGasPrice(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	var opts *bind.TransactOpts
	var err error
	// 创建交易配置对象
	opts, err = bind.NewKeyedTransactorWithChainID(de.Cfg.PrivateKey, de.Cfg.ChainId)
	// 失败处理
	if err != nil {
		log.Error("new keyed transactor with chain id fail", "err", err)
		return nil, err
	}

	// 设置交易上下文、nonce、标记为不发送
	opts.Context = ctx
	// 使用旧交易的 nonce，确保它是同一笔交易的替代
	/**
	Nonce 是一个指针类型 *big.Int nonce 通常是 uint64。但是ABI通用处理大数，所以统一使用 *big.Int
	tx.Nonce() 是从交易中获取的 nonce 的方法，nonce 通常是 uint64
	*/
	opts.Nonce = new(big.Int).SetUint64(tx.Nonce())
	// 表示只构造交易，不发送到链上
	opts.NoSend = true
	// 使用RawTransact构造一个新的裸交易（原始交易数据 tx.Data()）
	// 这一步会根据链上情况自动设置 GasFeeCap 和 GasTipCap
	findalTx, err := de.RawDappLinkVrfContract.RawTransact(opts, tx.Data())

	switch {
	case err == nil:
		return findalTx, nil
	case de.isMaxPriorityFeePerGasNotFoundError(err):
		// 如果链上节点 不支持 EIP-1559，老节点不支持eth_maxPriorityFeePerGas，就使用预设的 FallbackGasTipCap 再试一次
		log.Info("Don't support priority fee")
		opts.GasTipCap = FallbackGasTipCap
		return de.RawDappLinkVrfContract.RawTransact(opts, tx.Data())
	default:
		return nil, err
	}
}

func (de *DriverEngine) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return de.Cfg.ChainClient.SendTransaction(ctx, tx)
}

func (de *DriverEngine) isMaxPriorityFeePerGasNotFoundError(err error) bool {
	return strings.Contains(err.Error(), errMaxPriorityFeePerGasNotFound.Error())
}

func (de *DriverEngine) fulfillRandomWords(ctx context.Context, requestId *big.Int, randomList []*big.Int) (*types.Transaction, error) {
	// 通过链上的 RPC 获取当前调用者地址的 nonce
	nonce, err := de.Cfg.ChainClient.NonceAt(ctx, de.Cfg.CallerAddress, nil)
	if err != nil {
		log.Error("get nonce error", "err", err)
		return nil, err
	}
	// 创建交易配置对象
	opts, err := bind.NewKeyedTransactorWithChainID(de.Cfg.PrivateKey, de.Cfg.ChainId)
	if err != nil {
		log.Error("new keyed transactor with chain id fail", "err", err)
		return nil, err
	}

	// 设置上下文，用于取消/超时控制
	opts.Context = ctx
	// 明确指定这笔交易的 nonce
	opts.Nonce = new(big.Int).SetUint64(nonce)
	// 不直接发送交易，只构造交易（用于手动估算 gas, 设置 fee cap 等）
	opts.NoSend = true

	tx, err := de.DappLinkVrfContract.FulfillRandomWords(opts, requestId, randomList)
	switch {
	case err == nil:
		return tx, nil

	case de.isMaxPriorityFeePerGasNotFoundError(err):
		log.Info("Don't support priority fee")
		opts.GasTipCap = FallbackGasTipCap
		return de.DappLinkVrfContract.FulfillRandomWords(opts, requestId, randomList)

	default:
		return nil, err
	}
}

func (de *DriverEngine) FulfillRandomWords(requestId *big.Int, randomList []*big.Int) (*types.Receipt, error) {
	tx, err := de.fulfillRandomWords(de.Ctx, requestId, randomList)
	if err != nil {
		log.Error("build request random words tx fail", "err", err)
		return nil, err
	}

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		return de.UpdateGasPrice(ctx, tx)
	}

	// 使用状态管理器：自动构造+动态提价+重试发送+等待确认
	receipt, err := de.TxMgr.Send(de.Ctx, updateGasPrice, de.SendTransaction)
	if err != nil {
		log.Error("send tx fail", "err", err)
		return nil, err
	}
	return receipt, nil
}
