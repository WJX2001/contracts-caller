package node

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/WJX2001/contract-caller/common/bigint"
	"github.com/ethereum/go-ethereum/core/types"
)

// 区块头遍历器

var (
	ErrHeaderTraversalAheadOfProvider            = errors.New("the HeaderTraversal's internal state is ahead of the provider")
	ErrHeaderTraversalAndProviderMismatchedState = errors.New("the HeaderTraversal and provider have diverged in state")
)

type HeaderTraversal struct {
	ethClient EthClient
	chainId   uint

	latestHeader        *types.Header // 最近一次从链上获取的最新区块头
	lastTraversedHeader *types.Header // 上次遍历到的区块头 （当前状态停在这里）

	blockConfirmationDepth *big.Int // 区块确认深度，确保我们只处理已经确认的区块
}

// 构造函数，初始化一个构造器实例
func NewHeaderTraversal(ethClient EthClient, fromHeader *types.Header, confDepth *big.Int, chainId uint) *HeaderTraversal {
	return &HeaderTraversal{
		ethClient:              ethClient,
		lastTraversedHeader:    fromHeader,
		blockConfirmationDepth: confDepth,
		chainId:                chainId,
	}
}

// 辅助 getter 方法
func (f *HeaderTraversal) LatestHeader() *types.Header {
	return f.latestHeader
}

func (f *HeaderTraversal) LastTraversedHeader() *types.Header {
	return f.lastTraversedHeader
}

// 从上次遍历的区块头继续，获取下一批新区块头
func (f *HeaderTraversal) NextHeaders(maxSize uint64) ([]types.Header, error) {
	latestHeader, err := f.ethClient.BlockHeaderByNumber(nil)
	if err != nil {
		return nil, fmt.Errorf("unable to query latest block: %w", err)
	} else if latestHeader == nil {
		return nil, fmt.Errorf("latest header unreported")
	} else {
		f.latestHeader = latestHeader
	}

	// 能安全处理的最新区块号
	endHeight := new(big.Int).Sub(latestHeader.Number, f.blockConfirmationDepth)
	if endHeight.Sign() < 0 {
		// No blocks with the provided confirmation depth available
		return nil, nil
	}

	// 检查是否已经遍历到最新确认区块
	if f.lastTraversedHeader != nil {
		cmp := f.lastTraversedHeader.Number.Cmp(endHeight)
		if cmp == 0 {
			return nil, nil // 已经是最新的,没有新区块
		} else if cmp > 0 {
			// 当前区块号比 endHeight 大，说明内部状态超前链上状态（异常状态）
			return nil, ErrHeaderTraversalAheadOfProvider
		}
	}
	// 计算下一个要获取的区块号范围,下一次要获取的区块号 = 上次区块号 + 1
	nextHeight := bigint.Zero
	if f.lastTraversedHeader != nil {
		nextHeight = new(big.Int).Add(f.lastTraversedHeader.Number, bigint.One)
	}

	// 限制批量大小
	endHeight = bigint.Clamp(nextHeight, endHeight, maxSize)
	// 批量查询区块头
	headers, err := f.ethClient.BlockHeadersByRange(nextHeight, endHeight, f.chainId)
	if err != nil {
		return nil, fmt.Errorf("error querying blocks by range: %w", err)
	}
	numHeaders := len(headers)
	if numHeaders == 0 {
		return nil, nil
	} else if f.lastTraversedHeader != nil && headers[0].ParentHash != f.lastTraversedHeader.Hash() {
		// 校验链连续性（防止分叉/状态不一致）
		// 如果第一个新区块头的 ParentHash 不等于上一个区块的 Hash
		// 说明链发生了分叉或者 provider 的数据和本地状态不一致
		fmt.Println(f.lastTraversedHeader.Number)
		fmt.Println(headers[0].Number)
		fmt.Println(len(headers))
		return nil, ErrHeaderTraversalAndProviderMismatchedState
	}

	// 更新最后遍历到的区块头，并返回本次取到的所有 headers
	f.lastTraversedHeader = &headers[numHeaders-1]
	return headers, nil
}
