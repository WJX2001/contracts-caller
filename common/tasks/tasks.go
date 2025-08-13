package tasks

import (
	"fmt"
	"runtime/debug"

	"golang.org/x/sync/errgroup"
)

/*
	TODO: 定义了一个并发任务组 Group
		支持并发执行多个任务(goroutine) 并统一等待结果，同时捕获并处理 panic
		- 支持并发任务执行
		- 如果某个任务发生 panic 能捕获并通过自定义处理函数 HandleCrit 处理
		- 所有任务完成后，通过 Wait() 等待并返回可能的错误
*/

type Group struct {
	errGroup   errgroup.Group
	HandleCrit func(err error)
}

// 添加任务
func (t *Group) Go(fn func() error) {
	/*
		使用 errgroup.Go() 开一个新的 goroutine 执行传入的任务 fn()
		用 defer 包住：确保：
			- 如果任务内部触发 panic，不会导致整个程序崩溃
			- 而是打印栈信息（debug.PrintStack()）
			- 并调用用户定义的 HandleCrit() 处理逻辑
	*/
	t.errGroup.Go(func() error {
		defer func() {
			if err := recover(); err != nil {
				debug.PrintStack()
				t.HandleCrit(fmt.Errorf("panic: %v", err))
			}
		}()
		return fn()
	})
}
