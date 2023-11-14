package unify

import (
	"sync"

	"high-freq-quant-go/core/exch"
	fs "high-freq-quant-go/exchange/binance/binanceapi/futures"
)

type UnifyMap struct {
	Pos         map[fs.PositionSideType]string
	OrderType   map[fs.TimeInForceType]string
	OrderStatus map[fs.OrderStatusType]string

	pl, tl, sl sync.Mutex
}

func NewUnifyMap() UnifyMap {
	unify := UnifyMap{}
	unify.Pos = map[fs.PositionSideType]string{
		fs.PositionSideTypeBoth:  exch.PositionBoth,
		fs.PositionSideTypeLong:  exch.PositionLong,
		fs.PositionSideTypeShort: exch.PositionShort,
	}
	unify.OrderStatus = map[fs.OrderStatusType]string{
		fs.OrderStatusTypeNew:             exch.OrderOpen,
		fs.OrderStatusTypePartiallyFilled: exch.OrderOpen,
		fs.OrderStatusTypeFilled:          exch.OrderFinished,
		fs.OrderStatusTypeCanceled:        exch.OrderFinished,
		fs.OrderStatusTypeRejected:        exch.OrderFinished,
		fs.OrderStatusTypeExpired:         exch.OrderFinished,
		fs.OrderStatusTypeNewInsurance:    exch.OrderFinished,
		fs.OrderStatusTypeNewADL:          exch.OrderFinished,
	}
	unify.OrderType = map[fs.TimeInForceType]string{
		fs.TimeInForceTypeGTC: exch.OrderGtc,
		fs.TimeInForceTypeIOC: exch.OrderIoc,
		fs.TimeInForceTypeFOK: exch.OrderFok,
		fs.TimeInForceTypeGTX: exch.OrderPoc,
	}
	return unify
}
