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
	LatestEventBlockHeader() (*common2.BlockHeader, error)
}

type EventBlocksDB interface {
	BlocksView
	StoreEventBlocks([]EventBlocks) error
}

type eventBlocksDB struct {
	gorm *gorm.DB
}

func (e eventBlocksDB) LatestEventBlockHeader() (*common2.BlockHeader, error) {
	eventQuery := e.gorm.Where("number = (?)", e.gorm.Table("event_blocks").Select("MAX(number)"))
	var header common2.BlockHeader
	result := eventQuery.Take(&header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &header, nil
}

func (e eventBlocksDB) StoreEventBlocks(eventBlocks []EventBlocks) error {
	result := e.gorm.CreateInBatches(&eventBlocks, len(eventBlocks))
	return result.Error
}

func NewEventBlocksDB(db *gorm.DB) EventBlocksDB {
	return &eventBlocksDB{gorm: db}
}
