package config

import (
	"time"

	"github.com/WJX2001/contract-caller/flags"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

const (
	defaultConfirmations = 64
	defaultLoopInterval  = 5000
)

type Config struct {
	Migrations     string      // 数据库迁移文件路径
	Chain          ChainConfig // 区块链配置
	MasterDB       DBConfig    // 主数据库配置
	SlaveDB        DBConfig    // 从数据库配置
	SlaveDbEnable  bool        // 是否启用从数据库
	ApiCacheEnable bool        // 是否启用 API 缓存
}

type ChainConfig struct {
	ChainRpcUrl                       string           // 区块链节点 RPC 地址
	ChainId                           uint             // 链ID
	StartingHeight                    uint64           // 起始区块高度
	Confirmations                     uint64           // 确认数（需要多少个确认区块才认为交易或事件是安全的）
	BlockStep                         uint64           // 区块步长（扫块时每次跨多少个区块）
	Contracts                         []common.Address // 合约地址列表
	MainLoopInterval                  time.Duration    // 主循环执行间隔
	EventInterval                     time.Duration    // 事件处理间隔
	CallInterval                      time.Duration    // 普通合约调用间隔
	PrivateKey                        string           // 钱包私钥
	DappLinkVrfContractAddress        string           // VRF合约地址
	DappLinkVrfFactoryContractAddress string           // VRF工厂合约地址（用于创建VRF实例）
	CallerAddress                     string           // 调用者地址
	NumConfirmations                  uint64           // 确认数量
	SafeAbortNonceTooLowCount         uint64           // 交易 nonce 太低时，安全终止的计数阈值
	Mnemonic                          string           // 助记词
	CallerHDPath                      string           // HD钱包的派生路径
	Passphrase                        string           // 助记词的额外密码（如果有）
}

type DBConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
}

// 配置加载函数
func LoadConfig(cliCtx *cli.Context) (Config, error) {
	var cfg Config
	cfg = NewConfig(cliCtx)

	if cfg.Chain.Confirmations == 0 {
		cfg.Chain.Confirmations = defaultConfirmations
	}

	if cfg.Chain.MainLoopInterval == 0 {
		cfg.Chain.MainLoopInterval = defaultLoopInterval
	}

	log.Info("loaded chain config", "config", cfg.Chain)
	return cfg, nil
}

func LoadContracts() []common.Address {
	var Contracts []common.Address
	Contracts = append(Contracts, DappLinkVrfAddr)
	return Contracts
}

// 配置创建函数
func NewConfig(ctx *cli.Context) Config {
	return Config{
		// 这里会去取命令行中对应的参数值，没传的话返回空字符串"",例如go run main.go --migrations ./db/migrations
		Migrations: ctx.String(flags.MigrationsFlag.Name),
		Chain: ChainConfig{
			ChainId:                           ctx.Uint(flags.ChainIdFlag.Name),
			ChainRpcUrl:                       ctx.String(flags.ChainRpcFlag.Name),
			StartingHeight:                    ctx.Uint64(flags.StartingHeightFlag.Name),
			Confirmations:                     ctx.Uint64(flags.ConfirmationsFlag.Name),
			BlockStep:                         ctx.Uint64(flags.BlocksStepFlag.Name),
			Contracts:                         LoadContracts(),
			MainLoopInterval:                  ctx.Duration(flags.MainIntervalFlag.Name),
			EventInterval:                     ctx.Duration(flags.EventIntervalFlag.Name),
			CallInterval:                      ctx.Duration(flags.CallIntervalFlag.Name),
			PrivateKey:                        ctx.String(flags.PrivateKeyFlag.Name),
			DappLinkVrfContractAddress:        ctx.String(flags.DappLinkVrfContractAddressFlag.Name),
			DappLinkVrfFactoryContractAddress: ctx.String(flags.DappLinkVrfFactoryContractAddressFlag.Name),
			CallerAddress:                     ctx.String(flags.CallerAddressFlag.Name),
			NumConfirmations:                  ctx.Uint64(flags.NumConfirmationsFlag.Name),
			SafeAbortNonceTooLowCount:         ctx.Uint64(flags.SafeAbortNonceTooLowCountFlag.Name),
			Mnemonic:                          ctx.String(flags.MnemonicFlag.Name),
			CallerHDPath:                      ctx.String(flags.CallerHDPathFlag.Name),
			Passphrase:                        ctx.String(flags.PassphraseFlag.Name),
		},
		MasterDB: DBConfig{
			Host:     ctx.String(flags.MasterDbHostFlag.Name),
			Port:     ctx.Int(flags.MasterDbPortFlag.Name),
			Name:     ctx.String(flags.MasterDbNameFlag.Name),
			User:     ctx.String(flags.MasterDbUserFlag.Name),
			Password: ctx.String(flags.MasterDbPasswordFlag.Name),
		},
		SlaveDB: DBConfig{
			Host:     ctx.String(flags.SlaveDbHostFlag.Name),
			Port:     ctx.Int(flags.SlaveDbPortFlag.Name),
			Name:     ctx.String(flags.SlaveDbNameFlag.Name),
			User:     ctx.String(flags.SlaveDbUserFlag.Name),
			Password: ctx.String(flags.SlaveDbPasswordFlag.Name),
		},
		SlaveDbEnable: ctx.Bool(flags.SlaveDbEnableFlag.Name),
	}
}
