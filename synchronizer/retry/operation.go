package retry

import (
	"context"
	"fmt"
	"time"
)

type ErrFailedPermanently struct {
	attempts int
	LastErr  error
}

func (e *ErrFailedPermanently) Error() string {
	return fmt.Sprintf("operation failed permanently after %d attempts: %v", e.attempts, e.LastErr)
}

func (e *ErrFailedPermanently) Unwrap() error {
	return e.LastErr
}

type pair[T, U any] struct {
	a T
	b U
}

func Do2[T any, U any](ctx context.Context, maxAttempts int, strategy Strategy, op func() (T, U, error)) (T, U, error) {
	f := func() (pair[T, U], error) {
		a, b, err := op()
		return pair[T, U]{a, b}, err
	}
	res, err := Do(ctx, maxAttempts, strategy, f)
	return res.a, res.b, err
}

// 在可配置的最大重试次数内，按给定的重试策略（如指数退避）重复执行一个操作函数，直到成功或最终失败
// ctx: 支持取消，一旦 ctx 结束，立刻返回 ctx.Err()
// maxAttempts: 最大重试次数，至少为1
// strategy: 决定每次失败后的等待时长（如指数退避）
// op: 实际要执行的操作，返回泛型结果和错误
func Do[T any](ctx context.Context, maxAttempts int, strategy Strategy, op func() (T, error)) (T, error) {
	var empty, ret T
	var err error

	if maxAttempts < 1 {
		return empty, fmt.Errorf("need at least 1 attempt to run op, but have %d max attempts", maxAttempts)
	}

	for i := 0; i < maxAttempts; i++ {
		if ctx.Err() != nil {
			return empty, ctx.Err()
		}

		ret, err = op()
		if err == nil {
			return ret, nil
		}
		if i != maxAttempts-1 {
			time.Sleep(strategy.Duration(i))
		}
	}
	return empty, &ErrFailedPermanently{
		attempts: maxAttempts,
		LastErr:  err,
	}
}
