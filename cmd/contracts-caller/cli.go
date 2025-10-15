package main

import (
	"context"

	dapplink_vrf "github.com/WJX2001/contract-caller"
	"github.com/WJX2001/contract-caller/common/cliapp"
	"github.com/WJX2001/contract-caller/common/opio"
	"github.com/WJX2001/contract-caller/config"
	"github.com/WJX2001/contract-caller/database"
	flag2 "github.com/WJX2001/contract-caller/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

func runDappLinkVrf(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log.Info("run dapplink vrf")
	// 1. 加载配置
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Error("failed to load config", "err", err)
		return nil, err
	}
	// 2. 创建 DappLinkVrf 对象
	return dapplink_vrf.NewDappLinkVrf(ctx.Context, &cfg, shutdown)
}

// 执行数据库迁移 （Schema 升级/初始化）
// 使用场景：首次部署或数据库结构更新时运行

func runMigrations(ctx *cli.Context) error {
	log.Info("Running migrations...")
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Error("failed to load config", "err", err)
		return err
	}

	ctx.Context = opio.CancelOnInterrupt(ctx.Context)
	db, err := database.NewDB(ctx.Context, cfg.MasterDB)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return err
	}
	defer func(db *database.DB) {
		err := db.Close()
		if err != nil {
			return
		}
	}(db)
	return db.ExecuteSQLMigration(cfg.Migrations)
}

func NewCli(GitCommit string, GitData string) *cli.App {
	flags := flag2.Flags
	return &cli.App{
		Version:              "v0.0.1",
		Description:          "An indexer of all optimism events with a serving api layer",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "index",
				Flags:       flags,
				Description: "Runs the indexing service",
				Action:      cliapp.LifecycleCmd(runDappLinkVrf),
			},
			{
				Name:        "migrate",
				Flags:       flags,
				Description: "Runs the database migrations",
				Action:      runMigrations,
			},
			{
				Name:        "version",
				Description: "print version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
		},
	}
}
