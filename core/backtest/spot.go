package backtest

import (
	"high-freq-quant-go/core/exch"
)

type Amount struct {
	Base  float64 //交易币数量
	Quote float64 //计价币数量

	AvgPrice  float64 //交易币均价
	Pnl       float64 //已实现盈亏
	LastPnl   float64 //最后一次已实现盈亏
	LastPrice float64 //最后价格
	UnPnl     float64 //当前仓位未实现盈亏

	Time int64
}

func SpotBackTrade(ti int64, ask, bid, base, quote, price, mfee, tfee float64, makers, takers []*exch.Order) ([]*exch.Order, *Amount) {
	tk := NewTicker(ti, ask, bid)
	orders := NewMakerOrders()
	at := NewAmount(base, quote, price)
	price, base, quote, pnl := at.AvgPrice, at.Base, at.Quote, 0.0
	for _, o := range makers {
		if !CheckSpotBase(base, o.Size) {
			continue
		}
		ap := SpotMakerOrderPrice(tk, o)
		//下单价错误
		if ap == nil {
			continue
		}
		//数量不够
		if o.Size < 0 && (base+o.Size) < 0 {
			continue
		}
		//资金不够
		ma := *ap * o.Size * (1 + mfee)
		if o.Size > 0 && (quote-ma) < 0 {
			continue
		}
		//继续挂
		if *ap == 0 {
			orders = append(orders, o)
			continue
		}
		p, b, pl := SpotAvgPrice(price, base, *ap, o.Size)
		//数量不够
		if pl == nil {
			continue
		}
		pnl += *pl
		price = p
		base = b
		quote -= ma
	}
	lbase, lquote := base, quote
	for _, o := range takers {
		if !CheckSpotBase(lbase, o.Size) {
			continue
		}
		ap := SpotTakerOrderPrice(tk, o)
		//挂不上
		if ap == nil {
			continue
		}
		//现货不足
		if o.Size < 0 && (lbase+o.Size) < 0 {
			continue
		}
		//资金不足
		ma := *ap * o.Size * (1 + tfee)
		if o.Size > 0 && (lquote-ma) < 0 {
			continue
		}
		//挂上
		if *ap == 0 {
			orders = append(orders, o)
			continue
		}
		p, b, pl := SpotAvgPrice(price, base, *ap, o.Size)
		if pl == nil {
			continue
		}
		if o.Size >= 0 {
			lquote -= ma
		} else {
			lbase -= o.Size
		}
		pnl += *pl
		price = p
		base = b
		quote -= ma
	}
	at.AvgPrice = price
	at.Base = base
	at.Quote = quote
	at.LastPnl = pnl
	at.LastPrice = (ask + bid) / 2
	at.UnPnl = (at.LastPrice - price) * base
	at.Time = ti
	return orders, at
}

func NewAmount(base, quete, price float64) *Amount {
	amount := &Amount{
		Base:     base,
		Quote:    quete,
		AvgPrice: price,
	}
	return amount
}

func CheckSpotBase(as, bs float64) bool {
	if bs == 0 || as < 0 {
		return false
	}
	return true
}

func SpotAvgPrice(ap, as, bp, bs float64) (float64, float64, *float64) {
	//价格、数量、盈利
	p, s, pnl := ap, as, 0.0
	if bp == 0 {
		return p, s, nil
	}
	if bs >= 0 && (as+bs) > 0 {
		p = (ap*as + bp*bs) / (as + bs)
		s += bs
	} else if bs < 0 && (as+bs) > 0 {
		p = ap
		s -= bs
		pnl = (ap - bp) * bs
	} else {
		return p, s, nil
	}
	return p, s, &pnl
}

func SpotMakerOrderPrice(tk *Ticker, or *exch.Order) *float64 {
	if or.Size == 0 || or.Price < 0 {
		return nil
	}
	price := 0.0
	if or.Price == 0 {
		if or.Size > 0 {
			price = tk.Ask
		}
		if or.Size < 0 {
			price = tk.Bid
		}
	} else {
		if or.Size > 0 && or.Price >= tk.Ask {
			price = or.Price
		}
		if or.Size < 0 && or.Price <= tk.Bid {
			price = or.Price
		}
	}
	return &price
}

func SpotTakerOrderPrice(tk *Ticker, or *exch.Order) *float64 {
	if or.Size == 0 || or.Price < 0 {
		return nil
	}
	price := 0.0
	if or.Price == 0 {
		if or.Size > 0 {
			price = tk.Ask
		}
		if or.Size < 0 {
			price = tk.Bid
		}
	} else {
		if or.Size > 0 && or.Price >= tk.Ask {
			price = tk.Ask
		}
		if or.Size < 0 && or.Price <= tk.Bid {
			price = tk.Bid
		}
	}
	return &price
}
