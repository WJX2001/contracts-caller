package worker

import (
	"errors"
	"fmt"
	"math/big"

	_ "github.com/WJX2001/contract-caller/database/utils/serializers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RequestSend struct {
	GUID       uuid.UUID      `gorm:"primaryKey" json:"guid"`
	RequestId  *big.Int       `json:"request_id" gorm:"serializer:u256"`
	VrfAddress common.Address `json:"vrf_address" gorm:"serializer:bytes"`
	NumWords   *big.Int       `json:"num_words" gorm:"serializer:u256"`
	Status     uint8          `json:"status"` // 0:扫到合约事件,1:已经上传随机数
	Timestamp  uint64
}

type RequestSendView interface {
	QueryUnHandleRequestSendList() ([]RequestSend, error)
}

type RequestSendDB interface {
	RequestSendView

	MarkRequestSendFinish(RequestSend) error
	StoreRequestSend([]RequestSend) error
}

type requestSendDB struct {
	gorm *gorm.DB
}

func NewRequestSendDB(db *gorm.DB) RequestSendDB {
	return &requestSendDB{gorm: db}
}

// 查询未处理的请求
func (db requestSendDB) QueryUnHandleRequestSendList() ([]RequestSend, error) {
	var requestSendList []RequestSend
	// status = 0 表示未处理的事件
	err := db.gorm.Table("request_sent").Where("status = ?", 0).Find(&requestSendList).Error

	if err != nil {
		return nil, fmt.Errorf("query unhandle request sent list failed: %w", err)
	}
	return requestSendList, nil
}

func (db requestSendDB) MarkRequestSendFinish(requestSent RequestSend) error {
	var requestSendSingle = RequestSend{}
	result := db.gorm.Table("request_sent").Where(&RequestSend{GUID: requestSent.GUID}).Take(&requestSendSingle)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		return result.Error
	}
	requestSendSingle.Status = 1
	err := db.gorm.Table("request_sent").Save(&requestSendSingle).Error
	if err != nil {
		return err
	}
	return nil
}

func (db requestSendDB) StoreRequestSend(RequestSendList []RequestSend) error {
	result := db.gorm.Table("request_sent").CreateInBatches(&RequestSendList, len(RequestSendList))
	return result.Error
}
