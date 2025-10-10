package contracts

import (
	"math/big"
	"time"

	"github.com/WJX2001/contract-caller/bindings"
	"github.com/WJX2001/contract-caller/database"
	"github.com/WJX2001/contract-caller/database/event"
	"github.com/WJX2001/contract-caller/database/worker"
	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type DappLinkVrfFactory struct {
	DlVrfFactoryAbi    *abi.ABI                             // 工厂合约的 ABI 定义
	DlVrfFactoryFilter *bindings.DappLinkVRFFactoryFilterer // 工厂合约事件过滤器
}

func NewDappLinkVrfFactory() (*DappLinkVrfFactory, error) {
	// 从绑定代码中获取工厂合约的 ABI
	dappLinkVrfFactoryAbi, err := bindings.DappLinkVRFFactoryMetaData.GetAbi()
	if err != nil {
		log.Error("get dapplink vrf factory abi fail", "err", err)
		return nil, err
	}
	// 创建工厂合约事件过滤器
	dappLinkVrfFactoryFilterer, err := bindings.NewDappLinkVRFFactoryFilterer(common.Address{}, nil)
	if err != nil {
		log.Error("new dapplink vrf factory filter fail", "err", err)
		return nil, err
	}
	// 将 ABI 和 过滤器组合成完整的解析器
	return &DappLinkVrfFactory{
		DlVrfFactoryAbi:    dappLinkVrfFactoryAbi,
		DlVrfFactoryFilter: dappLinkVrfFactoryFilterer,
	}, nil
}

func (dvff *DappLinkVrfFactory) ProcessDappLinkVrfFactoryEvent(db *database.DB, dappLinkVrfFactoryAddres string, startHeight, endHeight *big.Int) ([]worker.PoxyCreated, error) {
	var proxyCreatedList []worker.PoxyCreated
	// 创建合约事件过滤器
	contactFilter := event.ContractEvent{ContractAddress: common.HexToAddress(dappLinkVrfFactoryAddres)}
	contractEventList, err := db.ContractEvent.ContractEventsWithFilter(contactFilter, startHeight, endHeight)
	if err != nil {
		log.Error("query contacts event fail", "err", err)
		return proxyCreatedList, err
	}
	for _, contractEvent := range contractEventList {
		// 记录日志
		if contractEvent.EventSignature.String() == dvff.DlVrfFactoryAbi.Events["ProxyCreated"].ID.String() {
			// 转为业务模型
			proxyCreated, err := dvff.DlVrfFactoryFilter.ParseProxyCreated(*contractEvent.RLPLog)
			if err != nil {
				log.Error("proxy created fail", "err", err)
				return proxyCreatedList, err
			}
			log.Info("proxy created event", "MintProxyAddress", proxyCreated.MintProxyAddress)
			pc := worker.PoxyCreated{
				GUID:         uuid.New(),
				ProxyAddress: proxyCreated.MintProxyAddress,
				Timestamp:    uint64(time.Now().Unix()),
			}
			proxyCreatedList = append(proxyCreatedList, pc)
		}
	}
	return proxyCreatedList, nil
}
