package txmgr_test

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/WJX2001/contract-caller/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type testHarness struct {
	cfg       txmgr.Config
	mgr       txmgr.TxManager
	backend   *mockBackend
	gasPricer *gasPricer
}

// 创建测试用例所需要的核心组件
func newTestHarnessWithConfig(cfg txmgr.Config) *testHarness {
	backend := newMockBackend()
	mgr := txmgr.NewSimpleTxManager(cfg, backend)

	return &testHarness{
		cfg:       cfg,
		mgr:       mgr,
		backend:   backend,
		gasPricer: newGasPricer(3),
	}
}

func newTestHarness() *testHarness {
	return newTestHarnessWithConfig(configWithNumConfs(1))
}

func configWithNumConfs(numConfirmations uint64) txmgr.Config {
	return txmgr.Config{
		ResubmissionTimeout:       time.Second,
		ReceiptQueryInterval:      50 * time.Millisecond,
		NumConfirmations:          numConfirmations,
		SafeAbortNonceTooLowCount: 3,
	}
}

// 模拟 gas 价格变化
type gasPricer struct {
	epoch         int64      // 当前模拟的 epoch 时间
	mineAtEpoch   int64      // 交易被打包的模拟区块 epoch (控制某笔交易在哪个 epoch 被挖出来)
	baseGasTipFee *big.Int   // 矿工的小费
	baseBaseFee   *big.Int   // 基础手续费
	mu            sync.Mutex // 并发锁 保证在并发调用中字段更新安全
}

func newGasPricer(mineAtEpoch int64) *gasPricer {
	return &gasPricer{
		mineAtEpoch:   mineAtEpoch,
		baseGasTipFee: big.NewInt(5),
		baseBaseFee:   big.NewInt(7),
	}
}

// 期望的 gas 单价
func (g *gasPricer) expGasFeeCap() *big.Int {
	_, gasFeeCap := g.feesForEpoch(g.mineAtEpoch)
	return gasFeeCap
}

func (g *gasPricer) shouldMine(gasFeeCap *big.Int) bool {
	// 返回当前 epoch 期望/要求的 gasFeeCap
	// .Cmp() 是 Go 中 big.Int 提供的比较方法
	/*
		a.Cmp(b) == 0    // a == b
		a.Cmp(b) < 0     // a < b
		a.Cmp(b) > 0     // a > b
	*/
	return g.expGasFeeCap().Cmp(gasFeeCap) == 0
}

func (g *gasPricer) feesForEpoch(epoch int64) (*big.Int, *big.Int) {
	epochBaseFee := new(big.Int).Mul(g.baseBaseFee, big.NewInt(epoch))
	epochGasTipCap := new(big.Int).Mul(g.baseGasTipFee, big.NewInt(epoch))
	epochGasFeeCap := txmgr.CalcGasFeeCap(epochBaseFee, epochGasTipCap)

	return epochGasTipCap, epochGasFeeCap
}

func (g *gasPricer) sample() (*big.Int, *big.Int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.epoch++
	epochGasTipCap, epochGasFeeCap := g.feesForEpoch(g.epoch)
	return epochGasTipCap, epochGasFeeCap
}

type minedTxInfo struct {
	gasFeeCap   *big.Int
	blockNumber uint64
}

type mockBackend struct {
	mu          sync.RWMutex
	blockHeight uint64
	minedTxs    map[common.Hash]minedTxInfo // 存储哪些交易已经上链，以及他们在哪个区块上链
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		minedTxs: make(map[common.Hash]minedTxInfo),
	}
}

func (b *mockBackend) mine(txHash *common.Hash, gasFeeCap *big.Int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.blockHeight++

	if txHash != nil {
		b.minedTxs[*txHash] = minedTxInfo{
			gasFeeCap:   gasFeeCap,
			blockNumber: b.blockHeight,
		}
	}
}

func (b *mockBackend) BlockNumber(ctx context.Context) (uint64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.blockHeight, nil
}

func (b *mockBackend) TransactionReceipt(
	ctx context.Context,
	txHash common.Hash,
) (*types.Receipt, error) {

	b.mu.RLock()
	defer b.mu.RUnlock()

	txInfo, ok := b.minedTxs[txHash]
	if !ok {
		return nil, nil
	}

	return &types.Receipt{
		TxHash:      txHash,
		GasUsed:     txInfo.gasFeeCap.Uint64(),
		BlockNumber: big.NewInt(int64(txInfo.blockNumber)),
	}, nil
}

// 测试模块是否能在最低 gas价格下 成功发送确认交易

func TestTxMgrConfirmAtMinGasPrice(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	gasPricer := newGasPricer(1)

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		if gasPricer.shouldMine(tx.GasFeeCap()) {
			txHash := tx.Hash()
			h.backend.mine(&txHash, tx.GasFeeCap())
		}
		return nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// 测试TxManager 在交易始终无法上链确认的情况下，会自动取消重试行为
func TestTxMgrNeverConfirmCancel(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := h.gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}
	// 这个是模拟发送交易的逻辑，没有调用 h.backend.mine()，所以交易不会被打包,也就不会生成 receipt 模拟永远不会确认的情况
	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		// Don't publish tx to backend, simulating never being mined.
		return nil
	}

	// 设置一个5秒超时的上下文，意味着，mgr.Send() 最多尝试5秒，如果一直没确认，就返回context.DeadlineExceeded错误
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)
}

var errRpcFailure = errors.New("rpc failure")

// 测试 交易每次发送都失败（模拟 RPC节点挂了，网络问题），系统是否会阻塞重试，并在超时后优雅退出
func TestTxMgrBlocksOnFailingRpcCalls(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := h.gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		return errRpcFailure
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)
}

// 测试 之前几次广播失败，直到最后一次gas 达到要求才成功
func TestTxMgrOnlyOnePublicationSucceeds(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := h.gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		if !h.gasPricer.shouldMine(tx.GasFeeCap()) {
			return errRpcFailure
		}

		txHash := tx.Hash()
		h.backend.mine(&txHash, tx.GasFeeCap())
		return nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, h.gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// 模拟 EIP-1559 动态 Gas 交易重试 + 延迟挖矿的测试用例
// 即使需要多次 Gas 提价，TxManager 也能在达到预期 Gas 之后确认成功的交易
func TestTxMgrConfirmsMinGasPriceAfterBumping(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := h.gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		if h.gasPricer.shouldMine(tx.GasFeeCap()) {
			time.AfterFunc(5*time.Second, func() {
				txHash := tx.Hash()
				h.backend.mine(&txHash, tx.GasFeeCap())
			})
		}
		return nil
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, h.gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// 测试一个极其重要的边界条件，即使收到 nonce too low, 只要有交易上链，TxManager 也不应该终止发送流程
func TestTxMgrDoesntAbortNonceTooLowAfterMiningTx(t *testing.T) {

	t.Parallel()

	h := newTestHarness()

	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		gasTipCap, gasFeeCap := h.gasPricer.sample()
		return types.NewTx(&types.DynamicFeeTx{
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
		}), nil
	}

	sendTx := func(ctx context.Context, tx *types.Transaction) error {
		switch {
		case tx.GasFeeCap().Cmp(h.gasPricer.expGasFeeCap()) < 0:
			return nil

		case h.gasPricer.shouldMine(tx.GasFeeCap()):
			txHash := tx.Hash()
			h.backend.mine(&txHash, tx.GasFeeCap())
			time.AfterFunc(5*time.Second, func() {
				h.backend.mine(nil, nil)
			})
			return nil
		default:
			return core.ErrNonceTooLow
		}
	}

	ctx := context.Background()
	receipt, err := h.mgr.Send(ctx, updateGasPrice, sendTx)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, h.gasPricer.expGasFeeCap().Uint64(), receipt.GasUsed)
}

// 测试验证 当交易一开始就被挖出来时， WaitMined 会立刻成功返回交易回执
func TestWaitMinedReturnsReceiptOnFirstSuccess(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	tx := types.NewTx(&types.LegacyTx{}) // 创建一个旧版的交易
	txHash := tx.Hash()
	h.backend.mine(&txHash, new(big.Int))

	ctx := context.Background()
	receipt, err := txmgr.WaitMined(ctx, h.backend, tx, 50*time.Millisecond, 1)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, receipt.TxHash, txHash)
}

// 测试 WaitMined 方法在等待交易上链期间，如果 context 超时取消了，应该正确返回context.DeadlineExceeded 错误，并且不会返回任何交易回执（receipt）。
func TestWaitMinedCanBeCanceled(t *testing.T) {
	t.Parallel()

	h := newTestHarness()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create an unimined tx.
	tx := types.NewTx(&types.LegacyTx{})

	receipt, err := txmgr.WaitMined(ctx, h.backend, tx, 50*time.Millisecond, 1)
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)
}

// 验证 WaitMined 会在交易被挖出后，等待指定数量的确认区块，才返回 receipt，如果确认数未达到，在超时之前会一直等待，否则就返回 context.DeadlineExceeded 错误。
func TestWaitMinedMultipleConfs(t *testing.T) {
	t.Parallel()

	const numConfs = 2

	h := newTestHarnessWithConfig(configWithNumConfs(numConfs))
	// 创建一个 1 秒的上下文，如果超过1秒交易还没有满足条件，就会 context.DeadlineExceeded
	ctxt, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// 构造一个未挖矿的交易（类型为LegacyTx，代表最基本的交易类型）
	tx := types.NewTx(&types.LegacyTx{})
	txHash := tx.Hash()

	// 在mock 的区块链中挖出这个交易 （相当于这个交易进入第一个区块）
	h.backend.mine(&txHash, new(big.Int))

	/**
	由于此时只有1个区块内包含该交易，没有达到 numConfs = 2 的确认数，并且在1秒的上下文时限内没有新块被挖出
	所以 WaitMined 应该返回超时错误
	*/
	receipt, err := txmgr.WaitMined(ctxt, h.backend, tx, 50*time.Millisecond, numConfs)
	require.Equal(t, err, context.DeadlineExceeded)
	require.Nil(t, receipt)

	// 再次设置1秒的等待时间，用于第二次检测
	ctxt, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// 模拟链上又生成一个空区块（不包含交易的block） 此时原交易已经有了2个确认
	h.backend.mine(nil, nil)

	receipt, err = txmgr.WaitMined(ctxt, h.backend, tx, 50*time.Millisecond, numConfs)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, txHash, receipt.TxHash)
}

// 测试 尝试创建了一个配置了 0个确认数 的交易管理器时，程序应该panic（即直接崩溃终止），因为这是一个非法或逻辑错误的配置
func TestManagerPanicOnZeroConfs(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("NewSimpleTxManager should panic when using zero conf")
		}
	}()

	_ = newTestHarnessWithConfig(configWithNumConfs(0))
}

type failingBackend struct {
	returnSuccessBlockNumber bool // 控制是否让 BlockNumber 成功
	returnSuccessReceipt     bool // 控制是否让 TransactionReceipt 成功
}

func (b *failingBackend) BlockNumber(ctx context.Context) (uint64, error) {
	if !b.returnSuccessBlockNumber {
		b.returnSuccessBlockNumber = true
		return 0, errRpcFailure // 第一次调用失败
	}

	return 1, nil // 第二次调用成功
}

func (b *failingBackend) TransactionReceipt(
	ctx context.Context, txHash common.Hash) (*types.Receipt, error) {

	if !b.returnSuccessReceipt {
		b.returnSuccessReceipt = true
		return nil, errRpcFailure
	} // 第一次失败

	return &types.Receipt{
		TxHash:      txHash,
		BlockNumber: big.NewInt(1),
	}, nil // 第二次成功
}

// 这个测试验证的是 当 RPC 后端第一次失败，WaitMined 能够重试并最终返回交易回执，表现出容错能力
func TestWaitMinedReturnsReceiptAfterFailure(t *testing.T) {
	t.Parallel()

	var borkedBackend failingBackend

	tx := types.NewTx(&types.LegacyTx{})
	txHash := tx.Hash()

	ctx := context.Background()
	receipt, err := txmgr.WaitMined(ctx, &borkedBackend, tx, 50*time.Millisecond, 1)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, receipt.TxHash, txHash)
}
