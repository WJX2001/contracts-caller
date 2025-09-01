package dapplink_vrf

import (
	"context"
	"sync/atomic"

	"github.com/WJX2001/contract-caller/config"
	"github.com/WJX2001/contract-caller/database"
	"github.com/WJX2001/contract-caller/worker"
)

type DappLinkVrf struct {
	db *database.DB
	// 补充一个同步器
	// synchronizer  *synchronizer.Synchronizer
	// 补充一个事件处理器
	// eventHandler *event
	worker   *worker.Worker
	shutdown context.CancelCauseFunc
	stopped  atomic.Bool
}

func NewDappLinkVrf(ctx context.Context, cfg *config.Config, shutdown context.CancelCauseFunc) (*DappLinkVrf, error) {
	return nil, nil
}

// 启动所有服务
func (dvrf *DappLinkVrf) Start(ctx context.Context) error {
	// 1. 创建以太坊客户端

	// 2. 创建数据库连接

	// 3. 创建同步器

	// 4. 创建事件处理器

	// 5. 创建驱动引擎

	// 6. 创建工作器

	// 7. 返回完整的 DappLinkVrf 对象
}
