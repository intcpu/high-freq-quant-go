package unify

import (
	"strings"

	"high-freq-quant-go/core/log"
)

func Settle(symbol string) string {
	pq := strings.Split(symbol, "_")
	if len(pq) != 2 {
		log.Errorln(log.Conn, "get symbol settle error", symbol)
		return ""
	}
	return strings.ToLower(pq[1])
}

func UnitSize(size, unit float64) float64 {
	if unit < 1 {
		size = size / (1 / unit)
	} else {
		size = size * unit
	}
	return size
}
