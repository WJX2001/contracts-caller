package database

import (
	"context"
	"fmt"
	"github.com/WJX2001/contract-caller/config"
	"github.com/WJX2001/contract-caller/database/common"
	"github.com/WJX2001/contract-caller/database/event"
	"github.com/WJX2001/contract-caller/database/worker"
	"github.com/WJX2001/contract-caller/synchronizer/retry"
	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"path/filepath"
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

// 实现一个数据库访问层的封装实现
// 把GORM连接对象封装成DB，并在其中组合多个子数据模块，同时提供连接重试、事务支持、SQL迁移执行等实用功能

type DB struct {
	gorm            *gorm.DB
	Blocks          common.BlocksDB       // 区块头表的读写层
	ContractEvent   event.ContractEventDB // 合约事件的日志存储
	EventBlocks     worker.EventBlocksDB  // 事件同步进度管理
	FillRandomWords worker.FillRandomWordsDB
	RequestSend     worker.RequestSendDB
	PoxyCreated     worker.PoxyCreatedDB
}

func NewDB(ctx context.Context, dbConfig config.DBConfig) (*DB, error) {
	dsn := fmt.Sprintf("host=%s dbname=%s sslmode=disable", dbConfig.Host, dbConfig.Name)
	if dbConfig.Port != 0 {
		dsn += fmt.Sprintf(" port=%d", dbConfig.Port)
	}
	if dbConfig.User != "" {
		dsn += fmt.Sprintf(" user=%s", dbConfig.User)
	}

	if dbConfig.Password != "" {
		dsn += fmt.Sprintf(" password=%s", dbConfig.Password)
	}

	gormConfig := gorm.Config{
		SkipDefaultTransaction: true,
		CreateBatchSize:        3_000,
	}
	// 创建一个指数退避重试策略，用来控制程序在失败后等待时间策略
	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
	gorm, err := retry.Do[*gorm.DB](context.Background(), 10, retryStrategy, func() (*gorm.DB, error) {
		gorm, err := gorm.Open(postgres.Open(dsn), &gormConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		return gorm, nil
	})

	if err != nil {
		return nil, err
	}

	db := &DB{
		gorm:            gorm,
		Blocks:          common.NewBlocksDB(gorm),
		ContractEvent:   event.NewContractEventsDB(gorm),
		EventBlocks:     worker.NewEventBlocksDB(gorm),
		FillRandomWords: worker.NewFillRandomWordsDB(gorm),
		RequestSend:     worker.NewRequestSendDB(gorm),
		PoxyCreated:     worker.NewPoxyCreatedDB(gorm),
	}

	return db, nil
}

// 让传入的函数 fn 在同一个数据库事务中执行
// 这些操作都通过新的子数据库对象 txDB 来完成
// 事务成功就自动提交，失败就自动回滚
func (db *DB) Transaction(fn func(db *DB) error) error {
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		txDB := &DB{
			gorm:            tx,
			Blocks:          common.NewBlocksDB(tx),
			ContractEvent:   event.NewContractEventsDB(tx),
			EventBlocks:     worker.NewEventBlocksDB(tx),
			FillRandomWords: worker.NewFillRandomWordsDB(tx),
			RequestSend:     worker.NewRequestSendDB(tx),
			PoxyCreated:     worker.NewPoxyCreatedDB(tx),
		}
		return fn(txDB)
	})
}

func (db *DB) Close() error {
	sql, err := db.gorm.DB()
	if err != nil {
		return err
	}
	return sql.Close()
}

// 递归扫描一个文件夹，找出里面所有的 SQL文件，依次读取并执行其中的SQL语句
// 用于数据库的初始化或迁移
func (db *DB) ExecuteSQLMigration(migrationsFolder string) error {
	// 会递归遍历指定文件夹以及其子目录
	err := filepath.Walk(migrationsFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to process migration file: %s", path))
		}

		if info.IsDir() {
			return nil
		}
		// 读取 SQL 文件内容
		fileContent, readErr := os.ReadFile(path)
		if readErr != nil {
			return errors.Wrap(readErr, fmt.Sprintf("Error reading SQL file: %s", path))
		}

		execErr := db.gorm.Exec(string(fileContent)).Error
		if execErr != nil {
			return errors.Wrap(execErr, fmt.Sprintf("Error executing SQL script: %s", path))
		}
		return nil
	})
	return err
}
