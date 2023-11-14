package unify

import "high-freq-quant-go/core/exch"

const (
	PosSingle = "single"
	PosLong   = "dual_long"
	PosShort  = "dual_short"

	OrderOpen      = "open"
	OrderClosed    = "closed"
	OrderCancelled = "cancelled"
	EventPut       = "put"
	EventUpdate    = "update"
	EventFinish    = "finish"

	SideSell = "sell"
	SideBuy  = "buy"
)

var (
	SpotOrderMap = map[string]string{
		OrderOpen:      exch.OrderOpen,
		EventPut:       exch.OrderOpen,
		EventUpdate:    exch.OrderOpen,
		EventFinish:    exch.OrderFinished,
		OrderClosed:    exch.OrderFinished,
		OrderCancelled: exch.OrderFinished,
	}

	PositionMap = map[string]string{
		PosSingle: exch.PositionBoth,
		PosLong:   exch.PositionLong,
		PosShort:  exch.PositionShort,
	}
)
