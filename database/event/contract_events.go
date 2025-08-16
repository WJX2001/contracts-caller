package event

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContractEvent struct {
	GUID            uuid.UUID      `gorm:"primaryKey"`
	BlockHash       common.Hash    `gorm:"serializer:bytes"`
	ContractAddress common.Address `gorm:"serializer:bytes"`
	TransactionHash common.Hash    `gorm:"serializer:bytes"`
	LogIndex        uint64
	EventSignature  common.Hash `gorm:"serializer:bytes"`
	Timestamp       uint64
	RLPLog          *types.Log `gorm:"serializer:rlp;column:rlp_bytes"`
}

type ContractEventsView interface {
	ContractEvent(uuid.UUID) (*ContractEvent, error)
	ContractEventWithFilter(ContractEvent) (*ContractEvent, error)
	ContractEventsWithFilter(ContractEvent, *big.Int, *big.Int) ([]ContractEvent, error)
	LatestContractEventWithFilter(ContractEvent) (*ContractEvent, error)
}

type ContractEventDB interface {
	ContractEventsView
	StoreContractEvents([]ContractEvent) error
}

type contractEventDB struct {
	gorm *gorm.DB
}

func (db *contractEventDB) StoreContractEvents(events []ContractEvent) error {
	// 一次性插入所有事件
	result := db.gorm.CreateInBatches(&events, len(events))
	return result.Error
}
