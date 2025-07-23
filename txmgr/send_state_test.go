package txmgr_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	txmgr "github.com/the-web3/contracts-caller/txmgr"
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


func 
