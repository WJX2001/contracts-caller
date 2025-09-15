package event

import (
	"errors"
	"fmt"
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

// 从链上日志构造事件
// 取 topics[0] 作为事件签名
// 把原始 log 作为 RLPLog 用于完整还原
func ContractEventFromLog(log *types.Log, timestamp uint64) ContractEvent {
	eventSig := common.Hash{}
	if len(log.Topics) > 0 {
		eventSig = log.Topics[0]
	}

	return ContractEvent{
		GUID:            uuid.New(),
		BlockHash:       log.BlockHash,
		TransactionHash: log.TxHash,
		ContractAddress: log.Address,
		EventSignature:  eventSig,
		Timestamp:       timestamp,
		RLPLog:          log,
	}
}

// 只读视图接口
type ContractEventsView interface {
	ContractEvent(uuid.UUID) (*ContractEvent, error)
	ContractEventWithFilter(ContractEvent) (*ContractEvent, error)
	ContractEventsWithFilter(ContractEvent, *big.Int, *big.Int) ([]ContractEvent, error)
	LatestContractEventWithFilter(ContractEvent) (*ContractEvent, error)
}

// 读写接口
type ContractEventDB interface {
	ContractEventsView
	StoreContractEvents([]ContractEvent) error
}

type contractEventDB struct {
	gorm *gorm.DB
}

func NewContractEventsDB(db *gorm.DB) ContractEventDB {
	return &contractEventDB{gorm: db}
}

// 最新事件（按时间排序）
func (db *contractEventDB) LatestContractEventWithFilter(filter ContractEvent) (*ContractEvent, error) {
	var l1ContractEvent ContractEvent
	result := db.gorm.Where(&filter).Order("timestamp DESC").Take(&l1ContractEvent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &l1ContractEvent, nil
}

func (db *contractEventDB) StoreContractEvents(events []ContractEvent) error {
	// 一次性插入所有事件
	result := db.gorm.CreateInBatches(&events, len(events))
	return result.Error
}

func (db *contractEventDB) ContractEvent(uuid uuid.UUID) (*ContractEvent, error) {
	return db.ContractEventWithFilter(ContractEvent{GUID: uuid})
}

// 单条查询
func (db *contractEventDB) ContractEventWithFilter(filter ContractEvent) (*ContractEvent, error) {
	var l2ContractEvent ContractEvent
	result := db.gorm.Where(&filter).Take(&l2ContractEvent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &l2ContractEvent, nil
}

// 按条件 + 区块高度范围查询多条事件
func (db *contractEventDB) ContractEventsWithFilter(filter ContractEvent, fromHeight, toHeight *big.Int) ([]ContractEvent, error) {
	if fromHeight == nil {
		fromHeight = big.NewInt(0)
	}

	if toHeight == nil {
		return nil, errors.New("end height unspecified")
	}

	if fromHeight.Cmp(toHeight) > 0 {
		return nil, fmt.Errorf("fromHeight %d is greater than toHeight %d", fromHeight, toHeight)
	}

	query := db.gorm.Table("contract_events").Where(&filter)
	query = query.Joins("INNER JOIN block_headers ON contract_events.block_hash = block_headers.hash")
	query = query.Where("block_headers.number >= ? AND block_headers.number <= ?", fromHeight, toHeight)
	// 按照高度升序排序，指定只选回 contract_events 的列，便于后续处理
	query = query.Order("block_headers.number ASC").Select("contract_events.*")
	var events []ContractEvent
	// 执行查询并把结果映射到 events 切片
	result := query.Find(&events)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return events, nil
}
