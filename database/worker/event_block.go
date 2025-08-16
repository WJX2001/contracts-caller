package worker

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"

	common2 "github.com/WJX2001/contract-caller/database/common"
)

type EventBlocks struct {
	GUID       uuid.UUID   `gorm:"primaryKey"`
	Hash       common.Hash `gorm:"serializer:bytes"`
	ParentHash common.Hash `gorm:"serializer:bytes"`
	Number     *big.Int    `gorm:"serializer:u256"`
	Timestamp  uint64
}

type BlocksView interface {
	// LatestEventBlockHeader() (*common2.BlockHeader, error)
}

type EventBlocksDB interface {
	BlocksView
	StoreEventBlocks([]EventBlocks) error
}

type eventBlocksDB struct {
	gorm *gorm.DB
}

func (e eventBlocksDB) StoreEventBlocks(eventBlocks []EventBlocks) error {
	result := e.gorm.CreateInBatches(&eventBlocks, len(eventBlocks))
	return result.Error
}

// 从 block_headers表里，查出 event_blocks 表中记录最新区块号对应的那条区块头数据
func (e eventBlocksDB) LatestEventBlockHeader() (*common2.BlockHeader, error) {
	/*
		这个语句是一个GORM子查询
		找出 number 等于event_blocks表中最大 number 的那一行
	*/
	eventQuery := e.gorm.Where("number = (?)", e.gorm.Table("event_blocks").Select("MAX(number)"))
	var header common2.BlockHeader
	// 执行查询操作后将结果填充到 header 变量中
	result := eventQuery.Take(&header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &header, nil
}

// func NewEventBlocksDB(db *gorm.DB) EventBlocksDB {
// 	return &eventBlocksDB{gorm: db}
// }
