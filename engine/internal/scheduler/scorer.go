package scheduler

import "math"

// Score 计算账号的综合调度评分（0–100）
// 算法：urgency×0.35 + price×0.25 + health×0.30 + capacity×0.10
//
// 各维度说明：
//   - urgency  （紧急度）：余额越充足分越高，促进优先消耗高余额账号
//   - price     （价格）  ：单价越低分越高（用余额对数近似）
//   - health    （健康度）：直接使用 health 字段（0–100）
//   - capacity  （容量）  ：RPM 余量越多分越高
func Score(info *AccountInfo) float64 {
	urgency := scoreUrgency(info)
	price := scorePrice(info)
	health := info.Health
	capacity := scoreCapacity(info)

	return urgency*0.35 + price*0.25 + health*0.30 + capacity*0.10
}

// scoreUrgency 余额紧急度（余额越充足，urgency 越高）
// 分段映射：>=1000→100, >=100→80~100, >=10→60~80, >=1→40~60, <1→0
func scoreUrgency(info *AccountInfo) float64 {
	b := info.BalanceUSD
	switch {
	case b >= 1000:
		return 100
	case b >= 100:
		return 80 + (b-100)/900*20
	case b >= 10:
		return 60 + (b-10)/90*20
	case b >= 1:
		return 40 + (b-1)/9*20
	default:
		return 0
	}
}

// scorePrice 价格得分（余额越高意味着授权额度越大，单价潜在越低）
// 使用 log10 映射：log10(1)=0→0分，log10(10000)=4→100分
func scorePrice(info *AccountInfo) float64 {
	b := info.BalanceUSD
	if b <= 1 {
		return 0
	}
	v := math.Log10(b) / 4 * 100
	if v > 100 {
		return 100
	}
	return v
}

// scoreCapacity RPM 容量得分（使用率越低，得分越高）
func scoreCapacity(info *AccountInfo) float64 {
	if info.RPMLimit <= 0 {
		return 100 // 无限制，满分
	}
	used := float64(info.RPMUsed)
	limit := float64(info.RPMLimit)
	ratio := used / limit
	if ratio >= 1 {
		return 0
	}
	return (1 - ratio) * 100
}
