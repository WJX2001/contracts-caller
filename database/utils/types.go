package utils

import (
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// 对以太坊区块头 types.Header 进行RLP编码/解码封装，并提供一些辅助方法
/*
	RLP 是以太坊中最基础的序列化编码格式，用来表示区块、交易、状态数据等
	便于数据库存储和后续验证使用
*/

type RLPHeader types.Header

// RLP 编码
// 用这个头部创建一个Block 调用 Block.EncodeRLP(w) 来进行编码
func (h *RLPHeader) EncodeRLP(w io.Writer) error {
	return types.NewBlockWithHeader((*types.Header)(h)).EncodeRLP(w)
}

// RLP解码
func (h *RLPHeader) DecodeRLP(s *rlp.Stream) error {
	// 创建一个新的空区块
	block := new(types.Block)
	// 从 RLP 流中解码
	err := block.DecodeRLP(s)
	if err != nil {
		return err
	}
	// 拿出解码后的区块头
	header := block.Header()
	*h = (RLPHeader)(*header)
	return nil
}

func (h *RLPHeader) Header() *types.Header {
	return (*types.Header)(h)
}

// 返回区块头的哈希
func (h *RLPHeader) Hash() common.Hash {
	return h.Header().Hash()
}

type Bytes []byte

func (b Bytes) Bytes() []byte {
	return b[:]
}

func (b *Bytes) SetBytes(bytes []byte) {
	*b = bytes
}
