package bigint

import "math/big"

// 文件作用：处理任意精度整数和浮点数

// 定义了两个常用的大整数常量：0和1
var (
	Zero = big.NewInt(0)
	One  = big.NewInt(1)
)

// Clamp 限制范围函数
func Clamp(start, end *big.Int, size uint64) *big.Int {
	temp := new(big.Int)
	count := temp.Sub(end, start).Uint64() + 1
	if count <= size {
		return end
	}
	// (end - start + 1) ≤ size，控制一次最多拉多少区块
	temp.Add(start, big.NewInt(int64(size-1)))
	return temp
}

// 返回一个闭包函数，用来判断某个 big.Int 是否等于指定值。 适用于过滤器或条件判断
func Matcher(num int64) func(*big.Int) bool {
	return func(bi *big.Int) bool {
		return bi.Int64() == num
	}
}

// 把以太坊单位从 Wei -> ETH
// 1 ETH = 1e18 Wei
// 1 ETH = 10¹⁸ Wei
func WeiToETH(wei *big.Int) *big.Float {
	f := new(big.Float) // 任意精度浮点数
	/*
		wei 是一个 *big.Int，只能存整数；
		不能直接除以浮点型数；

		所以先把它转换成字符串（例如 "1000000000000000000"），
		再用 f.SetString() 读入为高精度浮点数。
	*/
	f.SetString(wei.String())
	return f.Quo(f, big.NewFloat(1e18))
}

func StringToInt(value string) int {
	if value == "" {
		return 0
	}
	return int(StringToBigInt(value).Int64())
}

func StringToBigInt(value string) *big.Int {
	intValue, success := big.NewInt(0).SetString(value, 0)
	if !success {
		return nil
	}
	return intValue
}
