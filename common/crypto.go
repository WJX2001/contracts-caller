package common

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"

	"github.com/decred/dcrd/hdkeychain/v3"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/tyler-smith/go-bip39"
)

/*
	1. 私钥管理：支持多种私钥获取方式
	2. HD钱包：支持 BIP39/BIP44 标准的分层确定性钱包
	3. 交易签名：提供以太坊交易签名功能
	4. HSM集成：支持硬件安全模块进行密钥管理
*/

var (
	ErrCannotGetPrivateKey = errors.New("invalid combination of private key or mnemonic + hdpath")
)

// 地址解析
// 验证以太坊地址格式
func ParseAddress(address string) (common.Address, error) {
	if common.IsHexAddress(address) {
		return common.HexToAddress(address), nil
	}
	return common.Address{}, fmt.Errorf("invalid address: %v", address)
}

// 统一的私钥获取接口
/*
	mnemonic: 助记词 (BIP-39 标准的 12/24 个单词)
	hdPath - HD 分层确定性 派生路径，例如 "m/44'/60'/0'/0/0"
	privKeyStr - 直接提供的私钥字符串（十六进制格式）
	password - 可选的密码，用于从助记词派生种子
*/
func GetConfiguredPrivateKey(mnemonic, hdPath, privKeyStr, password string) (*ecdsa.PrivateKey, error) {
	// 使用互斥验证逻辑，确保只使用一种方式获取私钥

	useMnemonic := mnemonic != "" && hdPath != ""
	usePrivKeyStr := privKeyStr != ""

	switch {
	case useMnemonic && !usePrivKeyStr: // 使用助记词 + HD 路径
		// 当提供了 mnemonic 和 hdPath，且没有提供 privKeyStr
		return DerivePrivateKey(mnemonic, hdPath, password)

	case usePrivKeyStr && !useMnemonic:
		// 当提供了 privKeyStr 且没有提供助记词和HD路径时
		// 直接解析十六进制私钥字符串
		return ParsePrivateKeyStr(privKeyStr)

	default:
		return nil, ErrCannotGetPrivateKey
	}

}

/*
fakeNetworkParams 作用：
  - 这是一个占位实现：
    HD 密钥派生本身是网络无关的
    版本前缀仅用于序列化扩展密钥（xprv/xpub）
    以太坊只需要原始私钥，不需要序列化扩展密钥
    因此返回空字节数组即可
*/
type fakeNetworkParams struct{}

func (f fakeNetworkParams) HDPrivKeyVersion() [4]byte {
	return [4]byte{}
}

func (f fakeNetworkParams) HDPubKeyVersion() [4]byte {
	return [4]byte{}
}

// 从助记词派生私钥
/*
	- 使用 BIP-39 生成种子
	- 通过 HD 派生路径生成特定账户的私钥
	这个函数实现了 分层确定钱包 （HD Wallet）遵循 BIP-39 和 BIP-44 标准

*/
func DerivePrivateKey(mnemonic, hdPath, password string) (*ecdsa.PrivateKey, error) {
	/*
		第一步：BIP-39 助记词转种子
		助记词：12或24个英文单词，人类可读的私钥表示
		PBKDF2 算法：
		- 输入：助记词 + 可选密码（salt）
		- 迭代：2048 次 HMAC-SHA512
		- 输出：512位（64字节）的种子
		- 种子 = PBKDF2(助记词, "mnemonic" + password, 2048轮, HMAC-SHA512)
	*/
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, password)
	if err != nil {
		return nil, err
	}

	/*
				第二步：创建主密钥：
				原理：使用种子 通过 BIP-32 算法生成主扩展密钥
				主密钥 = HMAC-SHA512(key="Bitcoin seed", data=seed)
				返回：
				前32字节：主私钥
				后32字节：链码（Chain Code），用于后续派生
				扩展密钥结构：
		Extended Key = {
		    私钥: 32字节
		    链码: 32字节  // 用于派生子密钥的"盐"
		    深度: 1字节
		    父指纹: 4字节
		    索引: 4字节
		}
	*/
	privKey, err := hdkeychain.NewMaster(seed, fakeNetworkParams{})
	if err != nil {
		return nil, err
	}

	/*
						第三步：解析派生路径
						派生路径格式：m / purpose' / coin_type' / account' / change / address_index

						以太坊标准路径示例：
				m/44'/60'/0'/0/0
				│  │   │   │  │ └─ 地址索引 (0, 1, 2, ...)
				│  │   │   │  └─── 找零链 (0=外部, 1=内部)
				│  │   │   └────── 账户索引 (0, 1, 2, ...)
				│  │   └────────── 币种类型 (60=以太坊, 0=比特币)
				│  └────────────── 用途 (44=BIP-44标准)
				└───────────────── 主密钥

				撇号（'）的含义：
		44' = 强化派生（Hardened Derivation）
		使用 索引 + 0x80000000 作为实际索引
		增强安全性，防止链码泄露导致父密钥暴露
	*/

	derivationPath, err := accounts.ParseDerivationPath(hdPath)
	if err != nil {
		return nil, err
	}

	// 第四步：逐级派生子密钥
	for _, child := range derivationPath {
		privKey, err = privKey.Child(child)
		if err != nil {
			return nil, err
		}
	}

	// 序列化私钥
	// 提取出 32 字节的原始私钥数据（去除链码、深度等元数据）
	rawPrivKey, err := privKey.SerializedPrivKey()
	if err != nil {
		return nil, err
	}

	// 将原始字节转换为 Go 的 ecdsa.PrivateKey 类型
	/*
		用于：
		 1. 签名交易
		 2. 生成公钥
		 3. 派生以太坊地址
	*/
	return crypto.ToECDSA(rawPrivKey)
}

/*
解析十六进制私钥
  - 去除 0x 前缀
  - 转换为 ECDSA 私钥对象
*/
func ParsePrivateKeyStr(privKeyStr string) (*ecdsa.PrivateKey, error) {
	hex := strings.TrimPrefix(privKeyStr, "0x")
	return crypto.HexToECDSA(hex)
}

// 交易签名功能
/*
	1. 一站式解析钱包私钥和合约地址
	2. 从私钥派生公钥和钱包地址
	3. 记录详细的配置信息

	使用场景：
		1. 在系统初始化时配置钱包
		2. 用于 VRF 系统中的交易签名
*/

func ParseWalletPrivKeyAndContractAddr(name string,
	mnemonic string,
	hdPath string,
	privKeyStr string,
	contractAddrStr string,
	password string) (*ecdsa.PrivateKey, common.Address, error) {
	// 1. 获取私钥
	privKey, err := GetConfiguredPrivateKey(mnemonic, hdPath, privKeyStr, password)
	if err != nil {
		return nil, common.Address{}, err
	}
	// 2. 解析合约地址
	contractAddress, err := ParseAddress(contractAddrStr)
	if err != nil {
		return nil, common.Address{}, err
	}
	// 3. 计算钱包地址
	walletAddress := crypto.PubkeyToAddress(privKey.PublicKey)

	// 4. 记录日志
	log.Info(name+" wallet params parsed successfully", "wallet_address",
		walletAddress, "contract_address", contractAddress)

	return privKey, contractAddress, nil
}
