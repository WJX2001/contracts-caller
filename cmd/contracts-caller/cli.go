package main

import (
	"context"

	dapplink_vrf "github.com/WJX2001/contract-caller"
	"github.com/WJX2001/contract-caller/common/cliapp"
	"github.com/WJX2001/contract-caller/config"
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
	// return dappli
	// return nil, nil
	return dapplink_vrf.NewDappLinkVrf(ctx.Context, &cfg, shutdown)
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
		},
	}
}
