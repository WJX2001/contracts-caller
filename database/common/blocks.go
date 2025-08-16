package common

import (
	"math/big"

	"github.com/WJX2001/contract-caller/database/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BlockHeader struct {
	GUID       uuid.UUID   `gorm:"primaryKey;DEFAULT replace(uuid_generate_v4()::text,'-','')"`
	Hash       common.Hash `gorm:"serializer:bytes"`
	ParentHash common.Hash `gorm:"serializer:bytes"`
	Number     *big.Int    `gorm:"serializer:u256"`
	Timestamp  uint64
	RLPHeader  *utils.RLPHeader `gorm:"serializer:rlp;column:rlp_bytes"`
}

type BlocksView interface {
	BlockHeader(common.Hash) (*BlockHeader, error)
	BlockHeaderByNumber(*big.Int) (*BlockHeader, error)
	BlockHeaderWithFilter(BlockHeader) (*BlockHeader, error)
	BlockHeaderWithScope(func(db *gorm.DB) *gorm.DB) (*BlockHeader, error)
	LatestBlockHeader() (*BlockHeader, error)
}

type BlocksDB interface {
	BlocksView
	StoreBlockHeaders([]BlockHeader) error
}

type blocksDB struct {
	gorm *gorm.DB
}

func (b blocksDB) StoreBlockHeaders(headers []BlockHeader) error {
	// 将 headers中每一条数据插入数据库
	// 这里数据不是大批量，否则使用CreateInBatches，小批量 使用 Create 更简洁
	result := b.gorm.Table("block_headers").Omit("guid").Create(&headers)
	return result.Error
}

// func NewBlocksDB(db *gorm.DB) BlocksDB {
// 	return &blocksDB{gorm: db}
// }
