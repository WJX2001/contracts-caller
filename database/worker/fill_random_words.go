package worker

import (
	"math/big"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FillRandomWords struct {
	GUID        uuid.UUID `gorm:"primaryKey" json:"guid"`
	RequestId   *big.Int  `json:"request_id" gorm:"serializer:u256"`
	RandomWords string    `json:"random_words"`
	Timestamp   uint64
}

type FillRandomWordsView interface {
}

type FillRandomWordsDB interface {
	FillRandomWordsView

	StoreFillRandomWords([]FillRandomWords) error
}

type fillRandomWordsDB struct {
	gorm *gorm.DB
}

func NewFillRandomWordsDB(db *gorm.DB) FillRandomWordsDB {
	return &fillRandomWordsDB{gorm: db}
}

func (db fillRandomWordsDB) StoreFillRandomWords(FillRandomWordsList []FillRandomWords) error {
	result := db.gorm.Table("fill_random_words").CreateInBatches(&FillRandomWordsList, len(FillRandomWordsList))
	return result.Error
}
