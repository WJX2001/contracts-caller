package txmgr

import (
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

type SendState struct {
	minedTxs                  map[common.Hash]struct{} // 保存已上链交易的hash
	nonceTooLowCount          uint64                   // nonce太低次数
	mu                        sync.RWMutex
	safeAbortNonceTooLowCount uint64 // 安全终止阈值 当nonceTooLowCount >= 这个值 可以安全停止重发
}

// 创建并初始化一个SendState实例
func NewSendState(safeAbortNonceTooLowCount uint64) *SendState {
	if safeAbortNonceTooLowCount == 0 {
		panic("txmgr: safeAbortNonceTooLowCount cannot be zero")
	}

	return &SendState{
		minedTxs:                  make(map[common.Hash]struct{}),
		nonceTooLowCount:          0,
		safeAbortNonceTooLowCount: safeAbortNonceTooLowCount,
	}

}

/*
检查传入错误是否是 nonce too low
  - 如果是则增加nonceTooLowCount
  - 如果交易已经被矿工打包，重新发送同样 nonce 的交易会触发 nonce too low
  - 多次遇到这个错误可推测原交易已经被成功打包
*/
func (s *SendState) ProcessSendError(err error) {
	if err == nil {
		return
	}

	if !strings.Contains(err.Error(), core.ErrNonceTooLow.Error()) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonceTooLowCount++
}

// 标记交易已经上链
func (s *SendState) TxMined(txHash common.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.minedTxs[txHash] = struct{}{}
}

// 取消已上链标记 TxNotmined
func (s *SendState) TxNotMined(txHash common.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, wasMined := s.minedTxs[txHash]
	delete(s.minedTxs, txHash)
	// 如果删除后minedTxs 为空，且之前确实有已经上链的交易，则重置nonceTooLowCount
	// 如果我们发现交易“消失”（链上没找到），则之前的 nonce too low 判断不再成立。
	if len(s.minedTxs) == 0 && wasMined {
		s.nonceTooLowCount = 0
	}
}

/*
是否应该立即终止
- 如果有交易已上链：不应该终止
- 如果 nonceTooLowCount >= 阈值：应该立即终止
- 多次遇到 nonce too low 且无交易确认 -> 极可能交易已上链
*/
func (s *SendState) ShouldAbortImmediately() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.minedTxs) > 0 {
		return false
	}
	return s.nonceTooLowCount >= s.safeAbortNonceTooLowCount
}

// 判断是否还有交易在等待链上确认
func (s *SendState) IsWaitingForConfirmation() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.minedTxs) > 0
}
