package main

import (
	"context"
	"os"

	"github.com/WJX2001/contract-caller/common/opio"
	"github.com/ethereum/go-ethereum/log"
)

// 用于追踪生产环境运行的是哪个版本的代码
var (
	GitCommit = "" // Git 提交的哈希值
	GitData   = "" // Git 提交的日期/事件
)

func main() {
	// 创建一个终端日志处理器
	/*
		日志级别设置为：LevelInfo: 会记录 Info、Warn、Error 等级别的日志
		日志输出到标准错误输出：os.Stderr¸
		最后一个参数 true 表示启用彩色输出
	*/
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))
	app := NewCli(GitCommit, GitData)
	// 创建一个带有中断阻塞器的上下文，这允许程序优雅处理系统中断信号
	ctx := opio.WithInterruptBlocker(context.Background())
	// 运行应用并处理错误 解析命令行参数
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Error("Application failed", err)
		os.Exit(1)
	}
}
