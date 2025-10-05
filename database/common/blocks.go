package common

import (
	"errors"
	"math/big"

	"github.com/WJX2001/contract-caller/database/utils"
	_ "github.com/WJX2001/contract-caller/database/utils/serializers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BlockHeader struct {
	GUID       uuid.UUID   `gorm:"primaryKey;DEFAULT replace(uuid_generate_v4()::text,'-','')"`
	Hash       common.Hash `gorm:"serializer:bytes"` // 区块哈希
	ParentHash common.Hash `gorm:"serializer:bytes"` // 父区块哈希
	Number     *big.Int    `gorm:"serializer:u256"`
	Timestamp  uint64
	RLPHeader  *utils.RLPHeader `gorm:"serializer:rlp;column:rlp_bytes"` // RLP 编码后的区块头，存储在数据库字段 rlp_bytes
}

func (BlockHeader) TableName() string {
	return "block_headers"
}

// 只读查询接口
type BlocksView interface {
	BlockHeader(common.Hash) (*BlockHeader, error)
	BlockHeaderByNumber(*big.Int) (*BlockHeader, error)
	BlockHeaderWithFilter(BlockHeader) (*BlockHeader, error)
	BlockHeaderWithScope(func(db *gorm.DB) *gorm.DB) (*BlockHeader, error)
	LatestBlockHeader() (*BlockHeader, error)
}

// 在原先基础上，增加了写操作，方便区分 只读数据库和读写数据库
type BlocksDB interface {
	BlocksView
	StoreBlockHeaders([]BlockHeader) error
}

type blocksDB struct {
	gorm *gorm.DB
}

func (b blocksDB) BlockHeader(hash common.Hash) (*BlockHeader, error) {
	return b.BlockHeaderWithFilter(BlockHeader{Hash: hash})
}

func (b blocksDB) BlockHeaderByNumber(number *big.Int) (*BlockHeader, error) {
	return b.BlockHeaderWithFilter(BlockHeader{Number: number})
}

// 通用过滤查询
func (b blocksDB) BlockHeaderWithFilter(header BlockHeader) (*BlockHeader, error) {
	return b.BlockHeaderWithScope(func(gorm *gorm.DB) *gorm.DB {
		return gorm.Where(&header)
	})
}

// 通过 scopes 查找
func (b blocksDB) BlockHeaderWithScope(f func(db *gorm.DB) *gorm.DB) (*BlockHeader, error) {
	var header BlockHeader
	result := b.gorm.Table("block_headers").Scopes(f).Take(&header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &header, nil
}

// 查最新的区块头
func (b blocksDB) LatestBlockHeader() (*BlockHeader, error) {
	var header BlockHeader
	result := b.gorm.Table("block_headers").Order("number DESC").Take(&header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, nil
	}
	return &header, nil
}

func (b blocksDB) StoreBlockHeaders(headers []BlockHeader) error {
	// 将 headers中每一条数据插入数据库
	// 这里数据不是大批量，否则使用CreateInBatches，小批量 使用 Create 更简洁
	result := b.gorm.Table("block_headers").Omit("guid").Create(&headers)
	return result.Error
}

func NewBlocksDB(db *gorm.DB) BlocksDB {
	return &blocksDB{gorm: db}
}
