package txmgr_test

import (
	"context"
	"math/big"
	"sync"

	"github.com/WJX2001/contract-caller/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type testHarness struct {
	cfg       txmgr.Config
	mgr       txmgr.TxManager
	backend   *mockBackend
	gasPricer *gasPricer
}

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
