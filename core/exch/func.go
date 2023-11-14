package exch

import (
	"fmt"
	"strings"
	"time"

	"high-freq-quant-go/core/stty"
)

var LastOrderCode string
var LastOrderId int64

func GetBaseQuote(symbol string) (string, string) {
	base, quote := "", ""
	quotes := []string{"USDT", "BTC", "ETH", "_USD", "BUSD", "BNB", "AUD",
		"BIDR", "BRL", "EUR", "GBP", "RUB", "TRY", "TUSD", "USDC", "DAI",
		"IDRT", "UAH", "NGN", "VAI", "USDP", "UST", "XRP", "TRX", "DOGE", "DOT"}
	ln := len(symbol)
	for _, q := range quotes {
		l := ln - len(q)
		suf := symbol[l:]
		if suf == q {
			pre := symbol[:l]
			base = strings.TrimRight(pre, "_")
			quote = q
			break
		}
	}
	return base, quote
}

// asset 成交保证金变化
func SumPosAvgPrice(ap, as, bp, bs float64) (float64, float64, float64, float64) {
	p, s, pnl, asset := ap, as, 0.0, 0.0
	if (as >= 0 && bs > 0) || (as <= 0 && bs < 0) {
		asset -= bp * bs
		p = (ap*as + bp*bs) / (as + bs)
		s = as + bs
	} else if (as > 0 && bs < 0) || (as < 0 && bs > 0) {
		size := as + bs
		side := size * as
		if size == 0 {
			pnl = (ap - bp) * bs
			p = 0
			s = size
			asset += ap * as
		} else if side > 0 {
			pnl = (ap - bp) * bs
			p = ap
			s = size
			asset += ap * bs
		} else {
			pnl = (bp - ap) * as
			p = bp
			s = size
			asset += (ap * as) - (bp * bs)
		}
	}
	return p, s, pnl, asset
}

func GetUUID(symbol string) string {
	code := symbol + time.Now().Format(stty.TimeFormatStr)
	if LastOrderCode != code {
		LastOrderCode = code
		LastOrderId = 0
	}
	LastOrderId++
	code = fmt.Sprintf("%s%04d", code, LastOrderId)
	return code
}
