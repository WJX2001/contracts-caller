package txmgr_test

import (
	"errors"
	"testing"

	txmgr "github.com/WJX2001/contract-caller/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/require"
)

var (
	testHash = common.HexToHash("0x01")
)

const testSafeAbortNonceTooLowCount = 3

func newSendState() *txmgr.SendState {
	return txmgr.NewSendState(testSafeAbortNonceTooLowCount)
}

func processNSendErrors(sendState *txmgr.SendState, err error, n int) {
	for i := 0; i < n; i++ {
		sendState.ProcessSendError(err)
	}
}

/*
		验证 刚创建时：
	  	- 不应该要求终止
	  	- 没有交易在等待确认
*/
func TestSendStateNoAbortAfterInit(t *testing.T) {
	sendState := newSendState()
	require.False(t, sendState.ShouldAbortImmediately())
	require.False(t, sendState.IsWaitingForConfirmation())
}

// 处理 nil 错误
func TestSendStateNoAbortAfterProcessNilError(t *testing.T) {
	sendState := newSendState()

	processNSendErrors(sendState, nil, testSafeAbortNonceTooLowCount)
	// 理应返回 false
	require.False(t, sendState.ShouldAbortImmediately())
}

// 处理无关(其他)错误
func TestSendStateNoAbortAfterProcessOtherError(t *testing.T) {
	sendState := newSendState()

	otherError := errors.New("other error")
	processNSendErrors(sendState, otherError, testSafeAbortNonceTooLowCount)
	require.False(t, sendState.ShouldAbortImmediately())
}

// 连续 nonce too low 错误后安全终止
func TestSendStateAbortSafelyAfterNonceTooLowButNoTxMined(t *testing.T) {
	sendState := newSendState()
	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.False(t, sendState.ShouldAbortImmediately())
	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.False(t, sendState.ShouldAbortImmediately())
	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.True(t, sendState.ShouldAbortImmediately())
}

// 交易上链后取消终止行为
// 由于已经上链了 所以不会终止
func TestSendStateMiningTxCancelsAbort(t *testing.T) {
	sendState := newSendState()

	sendState.ProcessSendError(core.ErrNonceTooLow)
	sendState.ProcessSendError(core.ErrNonceTooLow)
	sendState.TxMined(testHash)

	require.False(t, sendState.ShouldAbortImmediately())

	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.False(t, sendState.ShouldAbortImmediately())
}

/*
*

	系统刚经历链重组，不会立即abort，需要宽容处理
*/
func TestSendStateReorgingTxResetsAbort(t *testing.T) {
	sendState := newSendState()

	sendState.ProcessSendError(core.ErrNonceTooLow)
	sendState.ProcessSendError(core.ErrNonceTooLow)
	sendState.TxMined(testHash)
	sendState.TxNotMined(testHash)

	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.False(t, sendState.ShouldAbortImmediately())
}

// 只要有交易已成功上链，nonceTooLow 不再触发 abort。
func TestSendStateNoAbortEvenIfNonceTooLowAfterTxMined(t *testing.T) {
	sendState := newSendState()
	sendState.TxMined(testHash)

	processNSendErrors(
		sendState, core.ErrNonceTooLow, testSafeAbortNonceTooLowCount,
	)
	require.False(t, sendState.ShouldAbortImmediately())
}

func TestSendStateSafeAbortIfNonceTooLowPersistsAfterUnmine(t *testing.T) {
	sendState := newSendState()

	sendState.TxMined(testHash)
	sendState.TxNotMined(testHash)

	sendState.ProcessSendError(core.ErrNonceTooLow)
	sendState.ProcessSendError(core.ErrNonceTooLow)

	require.False(t, sendState.ShouldAbortImmediately())
	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.True(t, sendState.ShouldAbortImmediately())
}

func TestSendStateSafeAbortWhileCallingNotMinedOnUnminedTx(t *testing.T) {
	sendState := newSendState()

	processNSendErrors(
		sendState, core.ErrNonceTooLow, testSafeAbortNonceTooLowCount,
	)
	sendState.TxNotMined(testHash)
	require.True(t, sendState.ShouldAbortImmediately())
}

func TestSendStateIsWaitingForConfirmationAfterTxMined(t *testing.T) {
	sendState := newSendState()

	testHash2 := common.HexToHash("0x02")

	sendState.TxMined(testHash)
	require.True(t, sendState.IsWaitingForConfirmation())
	sendState.TxMined(testHash2)
	require.True(t, sendState.IsWaitingForConfirmation())
}

func TestSendStateIsNotWaitingForConfirmationAfterTxUnmined(t *testing.T) {
	sendState := newSendState()

	sendState.TxMined(testHash)
	sendState.TxNotMined(testHash)
	require.False(t, sendState.IsWaitingForConfirmation())
}
