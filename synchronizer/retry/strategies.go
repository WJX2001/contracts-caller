package retry

import (
	"math"
	"math/rand"
	"time"
)

/*
	用于控制重试操作之间的等待时间：
		- 指数退避策略
		- 固定间隔策略
*/

type Strategy interface {
	Duration(attempt int) time.Duration
}

type ExponentialStrategy struct {
	Min       time.Duration // 最小等待时间
	Max       time.Duration // 最大等待时间
	MaxJitter time.Duration // 最大抖动时间
}

/*
	指数退避策略
	e.Min       = 1 * time.Second
	e.Max       = 30 * time.Second
	e.MaxJitter = 2 * time.Second

	- 第一次重试：attempt = 0
		基础等待：1s + 2^0 * 1s = 2s
		加抖动：2s + [0, 2s) = [2s, 4s)
	- 第二次重试：attempt = 1
		基础等待：1s + 2^1 * 1s = 3s
		加抖动：3s + [0, 2s) = [3s, 5s)
	- 第三次重试：attempt = 2
		基础等待：1s + 2^2 * 1s = 5s
		加抖动：5s + [0, 2s) = [5s, 7s)
*/

func (e *ExponentialStrategy) Duration(attempt int) time.Duration {
	/*
		time.Duration: 时间间隔
			- 本质是一个 int64 单位是 纳秒 (ns)
			- 5 * time.second // 5 秒
	*/
	var jitter time.Duration

	if e.MaxJitter > 0 {
		// 在 [0, e.MaxJitter) 范围内生成随机时间
		jitter = time.Duration(rand.Int63n(e.MaxJitter.Nanoseconds()))
	}

	if attempt < 0 {
		// 如果 attempt < 0 直接返回最小值 + 抖动
		return e.Min + jitter
	}
	// 基础时长从 Min 开始
	durFloat := float64(e.Min)

	// 指数退避：每次失败，等待时间翻倍(2^attempt 秒)
	durFloat += math.Pow(2, float64(attempt)) * float64(time.Second)

	// 转换成 time.Duration
	dur := time.Duration(durFloat)

	// 如果超过了最大值 Max 就取 Max
	if durFloat > float64(e.Max) {
		dur = e.Max
	}
	dur += jitter

	return dur
}

// 默认配置
func Exponential() Strategy {
	return &ExponentialStrategy{
		Min:       0,
		Max:       10 * time.Second,
		MaxJitter: 250 * time.Millisecond,
	}
}

// 固定间隔策略
type FixedStrategy struct {
	Dur time.Duration
}

func (f *FixedStrategy) Duration(attempt int) time.Duration {
	return f.Dur
}

func Fixed(dur time.Duration) Strategy {
	return &FixedStrategy{
		Dur: dur,
	}
}
