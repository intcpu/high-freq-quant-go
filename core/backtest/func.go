package backtest

import (
	"os"

	"high-freq-quant-go/core/exch"
)

func FileExist(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}

func NewTicker(ti int64, ask, bid float64) *Ticker {
	ticker := &Ticker{
		Ask:  ask,
		Bid:  bid,
		Time: ti,
	}
	return ticker
}

func NewMakerOrders() []*exch.Order {
	orders := []*exch.Order{}
	return orders
}
