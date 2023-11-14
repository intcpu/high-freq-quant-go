package backtest

import "high-freq-quant-go/core/exch"

type SpotBox struct {
	Mfee, Tfee float64
	Pnl        float64

	Amount  *Amount
	Orders  []*exch.Order
	Profits []*Profit
}

type FutureBox struct {
	Mfee, Tfee float64
	Pnl        float64

	Pos     *TradePosBlc
	Orders  []*exch.Order
	Profits []*Profit
}

func NewSpotBox(base, quote, price float64) *SpotBox {
	amount := &Amount{
		Base:     base,
		Quote:    quote,
		AvgPrice: price,
	}
	spot := &SpotBox{
		Mfee:    0.002,
		Tfee:    0.002,
		Amount:  amount,
		Orders:  []*exch.Order{},
		Profits: []*Profit{},
	}
	return spot
}

func NewFutureBox(asset, lv float64) *FutureBox {
	pos := &TradePosBlc{
		Asset: asset,
		Lv:    lv,
	}
	futures := &FutureBox{
		Mfee:    0.0002,
		Tfee:    0.0004,
		Pos:     pos,
		Orders:  []*exch.Order{},
		Profits: []*Profit{},
	}
	return futures
}
