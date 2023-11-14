package unify

import (
	"high-freq-quant-go/core/exch"
	ba "high-freq-quant-go/exchange/binance/binanceapi"
	fs "high-freq-quant-go/exchange/binance/binanceapi/futures"
)

var (
	SpotOrderStatus = map[ba.OrderStatusType]string{
		ba.OrderStatusTypeNew:             exch.OrderOpen,
		ba.OrderStatusTypePartiallyFilled: exch.OrderOpen,
		ba.OrderStatusTypeFilled:          exch.OrderFinished,
		ba.OrderStatusTypeCanceled:        exch.OrderFinished,
		ba.OrderStatusTypePendingCancel:   exch.OrderFinished,
		ba.OrderStatusTypeRejected:        exch.OrderFinished,
		ba.OrderStatusTypeExpired:         exch.OrderFinished,
	}

	SpotOrderType = map[ba.TimeInForceType]string{
		ba.TimeInForceTypeGTC: exch.OrderGtc,
		ba.TimeInForceTypeIOC: exch.OrderIoc,
		ba.TimeInForceTypeFOK: exch.OrderFok,
	}

	PosMap = map[fs.PositionSideType]string{
		fs.PositionSideTypeBoth:  exch.PositionBoth,
		fs.PositionSideTypeLong:  exch.PositionLong,
		fs.PositionSideTypeShort: exch.PositionShort,
	}

	UnifyOrderStatus = map[fs.OrderStatusType]string{
		fs.OrderStatusTypeNew:             exch.OrderOpen,
		fs.OrderStatusTypePartiallyFilled: exch.OrderOpen,
		fs.OrderStatusTypeFilled:          exch.OrderFinished,
		fs.OrderStatusTypeCanceled:        exch.OrderFinished,
		fs.OrderStatusTypeRejected:        exch.OrderFinished,
		fs.OrderStatusTypeExpired:         exch.OrderFinished,
		fs.OrderStatusTypeNewInsurance:    exch.OrderFinished,
		fs.OrderStatusTypeNewADL:          exch.OrderFinished,
	}

	UnifyOrderType = map[fs.TimeInForceType]string{
		fs.TimeInForceTypeGTC: exch.OrderGtc,
		fs.TimeInForceTypeIOC: exch.OrderIoc,
		fs.TimeInForceTypeFOK: exch.OrderFok,
		fs.TimeInForceTypeGTX: exch.OrderPoc,
	}
)
