package synchronizer

import (
	"time"

	"github.com/WJX2001/contract-caller/database"
	"github.com/WJX2001/contract-caller/synchronizer/node"
)

type Synchronizer struct {
	ethClient node.EthClient // 以太坊客户端
	db        *database.DB   // 数据库连接

	loopInterval     time.Duration // 同步循环间隔
	headerBufferSize uint64        // 批量处理大小
	// headerTraversal *node.
}
