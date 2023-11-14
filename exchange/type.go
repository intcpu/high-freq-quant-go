package exchange

import (
	"high-freq-quant-go/core/exch"
	_ "high-freq-quant-go/exchange/binance"
	_ "high-freq-quant-go/exchange/gate"
)

var (
	ExNames = [2]string{exch.Binance, exch.Gate}
	ExTypes = [3]string{exch.Futures, exch.Spot, exch.Delivery}
)

func AllNameType() map[string]string {
	symbols := map[string]string{}
	for _, n := range ExNames {
		for _, t := range ExTypes {
			exs := n + "_" + t
			symbols[exs] = exs
		}
	}
	return symbols
}
