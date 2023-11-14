package backtest

import "math"

type Profit struct {
	T float64 //收益时间
	P float64 //收益金额 盈利正  亏损负
}

type Analyze struct {
	TotalAssets          float64 `json:"totalAssets"`          //总资产
	YearDays             float64 `json:"yearDays"`             //年天数
	TotalReturns         float64 `json:"totalReturns"`         //总收入
	AnnualizedReturns    float64 `json:"annualizedReturns"`    //年化收益
	SharpeRatio          float64 `json:"sharpeRatio"`          //夏普比率
	Volatility           float64 `json:"volatility"`           //投资组合年化报酬率的标准差
	MaxDrawdown          float64 `json:"maxDrawdown"`          //最大回撤率
	MaxDrawdownTime      float64 `json:"maxDrawdownTime"`      //最大回撤时间
	MaxAssetsTime        float64 `json:"maxAssetsTime"`        //最大资产时间
	MaxDrawdownStartTime float64 `json:"maxDrawdownStartTime"` //最大回撤开始时间
	WinningRate          float64 `json:"winningRate"`          //胜率
}

// ReturnSharpe 回测夏普
//totalAssets 总资产
//ts te 开始时间  结束时间
//period 周期时间:天 86400000 小时:3600000
//yearDays 周期数，假如周期时间86400000 则年化：365 月化: 30 日化: 1
//profits 收益列表
func ReturnSharpe(totalAssets, ts, te, period, yearDays float64, profits []*Profit) *Analyze {
	// 年化 则 force by days
	//period = 86400000
	//yearDays = 365

	if len(profits) == 0 {
		return nil
	}
	freeProfit := 0.03 //年化无风险利率 比如银行利息 或 大盘收益 0.04
	yearRange := yearDays * period
	totalReturns := profits[len(profits)-1].P / totalAssets
	annualizedReturns := (totalReturns * yearRange) / (te - ts)

	// MaxDrawDown
	maxDrawdown := 0.0
	maxAssets := totalAssets
	maxAssetsTime := 0.0
	maxDrawdownTime := 0.0
	maxDrawdownStartTime := 0.0
	winningRate := 0.0
	winningResult := 0.0
	for i, profit := range profits {
		if i == 0 {
			if profit.P > 0 {
				winningResult++
			}
		} else {
			if profit.P > profits[i-1].P {
				winningResult++
			}
		}
		if (profits[i].P + totalAssets) > maxAssets {
			maxAssets = profits[i].P + totalAssets
			maxAssetsTime = profits[i].T
		}
		if maxAssets > 0 {
			var drawDown = 1 - (profits[i].P+totalAssets)/maxAssets
			if drawDown > maxDrawdown {
				maxDrawdown = drawDown
				maxDrawdownTime = profits[i].T
				maxDrawdownStartTime = maxAssetsTime
			}
		}
	}
	if len(profits) > 0 {
		winningRate = winningResult / float64(len(profits))
	}
	// trim profits
	i := 0
	var datas []float64
	sum := 0.0
	preProfit := 0.0
	perRatio := 0.0
	rangeEnd := te
	es := int64(te-ts) % int64(period)
	if es > 0 {
		rangeEnd = (te/period + 1) * period
	}
	for n := ts; n < rangeEnd; n += period {
		var dayProfit = 0.0
		var cut = n + period
		for i < len(profits) && profits[i].T < cut {
			dayProfit += profits[i].P - preProfit
			preProfit = profits[i].P
			i++
		}
		perRatio = ((dayProfit / totalAssets) * yearRange) / period
		sum += perRatio
		datas = append(datas, perRatio)
	}

	var sharpeRatio = 0.0
	var volatility = 0.0
	if len(datas) > 0 {
		var avg = sum / float64(len(datas))
		std := 0.0
		for _, d := range datas {
			std += math.Pow(d-avg, 2.0)
		}
		volatility = math.Sqrt(std / float64(len(datas)))
		if volatility != 0 {
			sharpeRatio = (annualizedReturns - freeProfit) / volatility
		}
	}
	analyze := &Analyze{
		TotalAssets:          totalAssets,          //总资产
		YearDays:             yearDays,             //年天数
		TotalReturns:         totalReturns,         //总收入
		AnnualizedReturns:    annualizedReturns,    //年化收益
		SharpeRatio:          sharpeRatio,          //夏普比率
		Volatility:           volatility,           //投资组合年化报酬率的标准差
		MaxDrawdown:          maxDrawdown,          //最大回撤率
		MaxDrawdownTime:      maxDrawdownTime,      //最大回撤时间
		MaxAssetsTime:        maxAssetsTime,        //最大资产时间
		MaxDrawdownStartTime: maxDrawdownStartTime, //最大回撤开始时间
		WinningRate:          winningRate,          //胜率
	}
	return analyze
}
