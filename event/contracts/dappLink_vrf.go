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

type DappLinkVrf struct {
	DlVrfAbi    *abi.ABI
	DlVrfFilter *bindings.DappLinkVRFFilterer
}

func NewDappLinkVrf() (*DappLinkVrf, error) {
	// 从生成的 绑定代码获取 ABI
	dappLinkVrfAbi, err := bindings.DappLinkVRFMetaData.GetAbi()
	if err != nil {
		log.Error("get dapplink vrf abi fail", "err", err)
		return nil, err
	}
	// 创建事件过滤器 用于解析事件日志
	dappLinkVRFFilterer, err := bindings.NewDappLinkVRFFilterer(common.Address{}, nil)
	if err != nil {
		log.Error("new dapplink vrf filter fail", "err", err)
		return nil, err
	}
	return &DappLinkVrf{
		DlVrfAbi:    dappLinkVrfAbi,
		DlVrfFilter: dappLinkVRFFilterer,
	}, nil
}

func (dvf *DappLinkVrf) ProcessDappLinkVrfEvent(db *database.DB, dappLinkVrfAddres string, startHeight, endHeight *big.Int) ([]worker.RequestSend, []worker.FillRandomWords, error) {
	var RequestSentList []worker.RequestSend
	var FillRandomWordList []worker.FillRandomWords

	// 查询原始事件
	contactFilter := event.ContractEvent{ContractAddress: common.HexToAddress(dappLinkVrfAddres)}
	contractEventList, err := db.ContractEvent.ContractEventsWithFilter(contactFilter, startHeight, endHeight)
	if err != nil {
		log.Error("query contacts event fail", "err", err)
		return RequestSentList, FillRandomWordList, err
	}

	for _, contractEvent := range contractEventList {
		// 解析 RequestSent 事件
		if contractEvent.EventSignature.String() == dvf.DlVrfAbi.Events["RequestSent"].ID.String() {
			rquestSentEvent, err := dvf.DlVrfFilter.ParseRequestSent(*contractEvent.RLPLog)
			if err != nil {
				log.Error("parse request sent fail", "err", err)
				return RequestSentList, FillRandomWordList, err
			}
			log.Info("Request sent event", "RequestId", rquestSentEvent.RequestId, "NumWords", rquestSentEvent.NumWords, "Current", rquestSentEvent.Current)
			// 转为业务数据
			rs := worker.RequestSend{
				GUID:       uuid.New(),
				RequestId:  rquestSentEvent.RequestId,
				VrfAddress: rquestSentEvent.Current,
				NumWords:   rquestSentEvent.NumWords,
				Status:     0, // 未处理状态
				Timestamp:  uint64(time.Now().Unix()),
			}
			RequestSentList = append(RequestSentList, rs)
		}
		// 解析 FillRandomWords 事件
		if contractEvent.EventSignature.String() == dvf.DlVrfAbi.Events["FillRandomWords"].ID.String() {
			fillRandomWords, err := dvf.DlVrfFilter.ParseFillRandomWords(*contractEvent.RLPLog)
			if err != nil {
				log.Error("parse fill random fail", "err", err)
				return RequestSentList, FillRandomWordList, err
			}
			log.Info("Fill random words event", "RequestId", fillRandomWords.RequestId, "RandomWords", fillRandomWords.RandomWords)
			var randomWords string
			for _, rword := range fillRandomWords.RandomWords {
				randomWords = rword.String()
			}
			frw := worker.FillRandomWords{
				GUID:        uuid.New(),
				RequestId:   fillRandomWords.RequestId,
				RandomWords: randomWords,
				Timestamp:   uint64(time.Now().Unix()),
			}
			FillRandomWordList = append(FillRandomWordList, frw)
		}
	}
	return RequestSentList, FillRandomWordList, nil
}
