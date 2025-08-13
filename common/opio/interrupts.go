package opio

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

/*
	封装了对操作系统中断信号的监听与响应逻辑，是一个优雅处理服务终止的工具包
		- 让程序具备可以监听中断信号，并在收到信号时优雅退出或取消 context，避免粗暴终止
		- 常见应用场景
			- Web 服务优雅退出
			- 后台任务监听 中断
*/

/**
为程序提供优雅响应操作系统中断信号的一组工具：
	- 可以简单地阻塞等待信号、把等待信号和 context 组合，在context 中注入一个拦截器
	- 提供一个把收到信号转成 context 取消的便捷函数 CancelOnInterrupt，便于做 graceful shutdown
*/

var DefaultInterruptSignals = []os.Signal{
	os.Interrupt,    // Ctrl+C (SIGINT)
	os.Kill,         // 强制终止 (不能被捕获, 实际上 signal.Notify 对它不起作用)
	syscall.SIGTERM, // kill 命令的默认信号
	syscall.SIGQUIT, // Ctrl+\ 或 kill -3
}

// 简单阻塞等待信号
/**
- 功能：简单阻塞直到收到指定 OS 信号（或默认集合中的一个）
- 实现：用 signal.Notify 把信号推到一个 channel，然后读这个 channel 阻塞
- 注意：os.Kill 不可捕获
*/
func BlockOnInterrupts(signals ...os.Signal) {
	// 没传参数 -> 默认用 DefaultInterruptSignals
	if len(signals) == 0 {
		signals = DefaultInterruptSignals
	}
	// 创建一个带缓冲的通道
	interruptChannel := make(chan os.Signal, 1)
	// 会让 Go 在收到指定信号的时候， 把它放进 channel
	// <- interruptChannel 会卡住 直到接受到一个信号
	signal.Notify(interruptChannel, signals...)
	<-interruptChannel
}

// 支持 context 的阻塞
/**
  - 功能：阻塞直到收到信号或上游 ctx 结束（更灵活）
*/
func BlockOnInterruptsContext(ctx context.Context, signals ...os.Signal) {
	if len(signals) == 0 {
		signals = DefaultInterruptSignals
	}
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, signals...)
	select {
	case <-interruptChannel: // 收到信号
	case <-ctx.Done(): // context 主动结束
		signal.Stop(interruptChannel) // 停止监听
	}
}

// 在 context 中注入拦截器
type interruptContextKeyType struct{}

var blockerContextKey = interruptContextKeyType{}

// interruptCatcher 持有一个 incoming 信号通道，缓冲为10
type interruptCatcher struct {
	incoming chan os.Signal
}

// 会阻塞直到收到信号或 ctx.Done()，是一个可复用的阻塞函数
func (c *interruptCatcher) Block(ctx context.Context) {
	select {
	case <-c.incoming: // 信号来了
	case <-ctx.Done(): // context 结束
	}
}

/*
*
  - 功能：在给定ctx 中注入一个 BlockFn (封装了 catcher.Block) 这样下游可以从 context 取到 BlockFn 来阻塞等待信号
*/
func WithInterruptBlocker(ctx context.Context) context.Context {
	if ctx.Value(blockerContextKey) != nil { // already has an interrupt handler
		return ctx
	}
	catcher := &interruptCatcher{
		incoming: make(chan os.Signal, 10),
	}
	signal.Notify(catcher.incoming, DefaultInterruptSignals...)

	return context.WithValue(ctx, blockerContextKey, BlockFn(catcher.Block))
}

func WithBlocker(ctx context.Context, fn BlockFn) context.Context {
	return context.WithValue(ctx, blockerContextKey, fn)
}

type BlockFn func(ctx context.Context)

func BlockerFromContext(ctx context.Context) BlockFn {
	v := ctx.Value(blockerContextKey)
	if v == nil {
		return nil
	}
	return v.(BlockFn)
}

// 把信号转成 context 取消
func CancelOnInterrupt(ctx context.Context) context.Context {
	inner, cancel := context.WithCancel(ctx)
	blockOnInterrupt := BlockerFromContext(ctx)
	if blockOnInterrupt == nil {
		blockOnInterrupt = func(ctx context.Context) {
			BlockOnInterruptsContext(ctx)
		}
	}
	go func() {
		blockOnInterrupt(ctx)
		cancel()
	}()

	return inner
}
