package database

import (
	"github.com/WJX2001/contract-caller/database/common"
	"github.com/WJX2001/contract-caller/database/event"
	"github.com/WJX2001/contract-caller/database/worker"
	"gorm.io/gorm"
)

type DB struct {
	gorm          *gorm.DB
	Blocks        common.BlocksDB
	ContractEvent event.ContractEventDB
	EventBlocks   worker.EventBlocksDB
}
