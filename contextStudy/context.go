package contextStudy

import (
	"context"
	"fmt"
	"time"
)

func ContextMain() {
	/**
	创建一个1秒超时的上下文 ctx。
		- 超过1秒后 ctx.Done() 会被关闭，ctx.Err 会变成 context deadline exceeded。
	*/
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 开启一个 go routine
	go handle(ctx, 1500*time.Millisecond)
	select {
	case <-ctx.Done():
		fmt.Println("main", ctx.Err())
		// context.WithCancel()
	}
}

func handle(ctx context.Context, duration time.Duration) {
	select {
	case <-ctx.Done():
		fmt.Println("handle", ctx.Err())
	case <-time.After(duration):
		// 超时之前完成了处理
		fmt.Println("process request with", duration)
	}
}
