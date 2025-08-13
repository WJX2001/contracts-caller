package cliapp

/*
	生命周期：
		程序启动 -> 程序运行 -> 程序管理
		在程序中我们需要优雅地管理这些阶段，确保程序能够：
			- 正确启动所有服务
			- 正常运行期间处理各种任务
			- 收到停止信号时优雅地关闭所有资源
*/

import (
	"context"
	"errors"
	"fmt"
	"github.com/WJX2001/contract-caller/common/opio"
	"github.com/urfave/cli/v2"
	"os"
)

var interruptErr = errors.New("interrupt signal")

type Lifecycle interface {
	Start(ctx context.Context) error // 启动服务
	Stop(ctx context.Context) error  // 停止服务
	Stopped() bool                   // 检查是否已经停止
}

// 创建一个Lifecycle对象，就像准备开餐厅的过程
type LifecycleAction func(ctx *cli.Context, close context.CancelCauseFunc) (Lifecycle, error)

func LifecycleCmd(fn LifecycleAction) cli.ActionFunc {
	return lifecycleCmd(fn, opio.BlockOnInterruptsContext)
}

type waitSignalFn func(ctx context.Context, signals ...os.Signal)

func lifecycleCmd(fn LifecycleAction, blockOnInterrupt waitSignalFn) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// TODO: 第一步程序启动，创建应用程序上下文
		/*
			  - hostCtx: 房东的合同（系统环境）
				- appCtx: 餐厅的经营许可（应用程序环境）
				- appCancel: 关闭餐厅的权力
		*/
		hostCtx := ctx.Context
		appCtx, appCancel := context.WithCancelCause(hostCtx)
		ctx.Context = appCtx

		// TODO: 第二步：监听中断信号
		/*
			就像安装监控系统：
				- 监听各种 关门信号（Ctrl+C、系统中断等）
				- 收到信号时，通知餐厅准备关门
		*/
		go func() {
			blockOnInterrupt(appCtx)
			appCancel(interruptErr)
		}()

		// TODO: 第三步：创建和启动服务

		/*
			创建应用程序生命周期对象
			fn(ctx, appCancel)：准备餐厅（创建所有服务）
		*/
		appLifecycle, err := fn(ctx, appCancel)
		if err != nil {
			return errors.Join(
				fmt.Errorf("failed to setup: %w", err),
				context.Cause(appCtx),
			)
		}

		// appLifecycle.Start：正式开门营业
		if err := appLifecycle.Start(appCtx); err != nil {
			return errors.Join(
				fmt.Errorf("failed to start: %w", err),
				context.Cause(appCtx),
			)
		}

		// TODO: 第四步：等待关闭信号，等待程序结束信号
		<-appCtx.Done()

		// TODO: 第五步：优雅关闭服务
		stopCtx, stopCancel := context.WithCancelCause(hostCtx)
		go func() {
			blockOnInterrupt(stopCtx)
			stopCancel(interruptErr)
		}()

		stopErr := appLifecycle.Stop(stopCtx)
		stopCancel(nil)
		if stopErr != nil {
			return errors.Join(
				fmt.Errorf("failed to stop: %w", stopErr),
				context.Cause(stopCtx),
			)
		}
		return nil
	}
}
