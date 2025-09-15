package database

import (
	"github.com/WJX2001/contract-caller/database/common"
	"github.com/WJX2001/contract-caller/database/event"
	"github.com/WJX2001/contract-caller/database/worker"
	"gorm.io/gorm"
)

/*
  - Blocks (database/common.BlocksDB): 区块头表的读写层。存/查 block_headers（Hash、ParentHash、Number、Timestamp、RLPHeader）。用于记录同步过的区块高度与去重校验；被同步器用来获取最新已索引区块等。
  - ContractEvent (database/event.ContractEventDB): 合约事件表的读写层。把链上 types.Log 以 RLP 完整落库，同时平铺 BlockHash/TxHash/Address/Topic0 等索引字段，支持按区块范围和过滤条件查询；被同步器/事件处理器用于存取事件。
  - EventBlocks (database/worker.EventBlocksDB): 事件处理进度用的“事件区块头”表。提供查询最新事件区块高度和批量写入，用于事件轮询的位点管理，避免重复或漏扫。
  - FillRandomWords (database/worker.FillRandomWordsDB): 业务结果表，记录已回填的随机数结果（RequestId、RandomWords、时间戳），支持批量写入；由工作器在完成 VRF 回填后落库。
  - RequestSend (database/worker.RequestSendDB): 请求任务表，记录合约请求的待处理任务（RequestId、VrfAddress、NumWords、Status）。提供：
    查询未处理列表（status=0）
    标记处理完成（status=1）
    批量写入请求
    工作器据此拉取任务并驱动链上回填。
  - PoxyCreated (database/worker.PoxyCreatedDB): 代理/子合约地址表。提供查询全部代理地址列表、批量写入。同步器会先查这张表拿到需要监听的合约地址集合，再用 FilterLogs 拉取这些地址的事件。
*/
type DB struct {
	gorm   *gorm.DB
	Blocks common.BlocksDB

	ContractEvent   event.ContractEventDB
	EventBlocks     worker.EventBlocksDB
	FillRandomWords worker.FillRandomWordsDB
	RequestSend     worker.RequestSendDB
	PoxyCreated     worker.PoxyCreatedDB
}
