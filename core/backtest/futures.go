package backtest

import (
	"math"

	"high-freq-quant-go/core/exch"
)

const (
	LiqRate = 0.9 //爆仓率
)

type TradePosBlc struct {
	Price    float64 //仓位均价
	Size     float64 //仓位数量
	Margin   float64 //仓位保证金 Price*Size/ pp.Lv
	Asset    float64 //剩余资产
	Pnl      float64 //已实现盈亏
	LastPnl  float64 //最后已实现盈亏
	UnPnl    float64 //当前仓位未实现盈亏
	LiqPrice float64 //当前仓位爆仓价
	Lv       float64 //仓位杠杆
	Time     int64
}

//asset 总资产
func FuturesBackTrade(ti int64, ask, bid, asset, mfee, tfee float64, makers, takers []*exch.Order, pos *exch.Position) ([]*exch.Order, *TradePosBlc) {
	tk := NewTicker(ti, ask, bid)
	pp := NewPTradePosBlc(ti, pos)
	LiqSellPos(tk, pp)
	LiqBuyPos(tk, pp)
	orders := NewMakerOrders()
	price, size, pnl := pp.Price, pp.Size, 0.0
	//已挂单处理
	for _, o := range makers {
		as, ap := MakerOrderTrade(tk, o)
		if as == nil || ap == nil {
			continue
		}
		if *as == 0 {
			orders = append(orders, o)
			continue
		}
		p, s, n, e := exch.SumPosAvgPrice(price, size, *ap, o.Size)
		ma := e - ((*as) * mfee)
		if asset+ma < 0 {
			break
		}
		price = p
		size = s
		pnl += n
		asset += ma
	}
	lasset := asset
	for _, o := range takers {
		as, ap := TakerOrderTrade(tk, o)
		if as == nil || ap == nil {
			continue
		}
		p, s, n, e := exch.SumPosAvgPrice(price, size, *ap, o.Size)
		ma := e - (*as)*tfee
		if lasset+ma < 0 {
			break
		}
		//主动挂单
		lasset -= ma
		if *as == 0 {
			orders = append(orders, o)
			continue
		}
		price = p
		size = s
		pnl += n
		asset += ma
	}
	pp.Price = price
	pp.Size = size
	pp.Margin = SetMargin(price, size, pp.Lv)
	pp.LiqPrice = SetLiqPrice(price, size, pp.Margin)
	pp.Pnl += pnl
	pp.LastPnl = pnl
	pp.UnPnl = ((ask+bid)/2 - price) * size
	pp.Asset = asset
	pp.Time = ti
	return orders, pp
}

func NewPTradePosBlc(ti int64, pos *exch.Position) *TradePosBlc {
	pp := &TradePosBlc{
		Price:    pos.Price,
		Size:     pos.Size,
		Margin:   pos.Margin,
		LiqPrice: pos.LiqPrice,
		Pnl:      pos.Pnl,
		Lv:       pos.Lv,
		Time:     ti,
	}
	if pp.Margin == 0 && pp.Lv > 0 {
		pp.Margin = SetMargin(pp.Price, pp.Size, pp.Lv)
	}
	if pp.LiqPrice == 0 && pp.Size != 0 {
		pp.LiqPrice = SetLiqPrice(pp.Price, pp.Size, pp.Margin)
	}
	return pp
}

func SetMargin(price, size, lv float64) float64 {
	if lv <= 0 {
		return 0
	}
	margin := math.Abs(price*size) / lv
	return margin
}

func SetLiqPrice(price, size, margin float64) float64 {
	if size == 0 {
		return 0
	}
	liqPrice := price - (margin * LiqRate / size)
	return liqPrice
}

func LiqBuyPos(tk *Ticker, pos *TradePosBlc) {
	if pos.Size <= 0 {
		return
	}
	if tk.Bid > pos.LiqPrice {
		return
	}
	pos.Pnl = -pos.Price * pos.Size
	pos.Margin = 0
	pos.Price = 0
	pos.Size = 0
	pos.LiqPrice = 0
	pos.UnPnl = 0
}

func LiqSellPos(tk *Ticker, pos *TradePosBlc) {
	if pos.Size >= 0 {
		return
	}
	if tk.Ask < pos.LiqPrice {
		return
	}
	pos.Pnl = pos.Price * pos.Size
	pos.Margin = 0
	pos.Price = 0
	pos.Size = 0
	pos.LiqPrice = 0
	pos.UnPnl = 0
}

func MakerOrderTrade(tk *Ticker, or *exch.Order) (*float64, *float64) {
	if or.Size == 0 || or.Price < 0 {
		return nil, nil
	}
	asset, price := 0.0, 0.0
	if or.Price == 0 {
		if or.Size > 0 {
			price = tk.Ask
			asset = or.Size * tk.Ask
		}
		if or.Size < 0 {
			price = tk.Bid
			asset = math.Abs(or.Size * tk.Bid)
		}
	} else {
		if or.Size > 0 && or.Price >= tk.Ask {
			price = or.Price
			asset = or.Price * or.Size
		}
		if or.Size < 0 && or.Price <= tk.Bid {
			price = or.Price
			asset = math.Abs(or.Price * or.Size)
		}
	}
	return &asset, &price
}

func TakerOrderTrade(tk *Ticker, or *exch.Order) (*float64, *float64) {
	if or.Size == 0 || or.Price < 0 {
		return nil, nil
	}
	asset, price := 0.0, 0.0
	if or.Price == 0 {
		if or.Size > 0 {
			price = tk.Ask
			asset = or.Size * tk.Ask
		}
		if or.Size < 0 {
			price = tk.Bid
			asset = math.Abs(or.Size * tk.Bid)
		}
	} else {
		if or.Size > 0 && or.Price >= tk.Ask {
			price = tk.Ask
			asset = tk.Ask * or.Size
		}
		if or.Size < 0 && or.Price <= tk.Bid {
			price = tk.Bid
			asset = math.Abs(tk.Bid * or.Size)
		}
	}
	return &asset, &price
}
