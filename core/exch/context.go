package exch

import "context"

func GetOrder(ctx context.Context) *Order {
	if ret, ok := ctx.Value(CtxOrder).(*Order); ok {
		return ret
	}
	return nil
}

func GetOrders(ctx context.Context) []*Order {
	if ret, ok := ctx.Value(CtxOrders).([]*Order); ok {
		return ret
	}
	return nil
}

func GetChan(ctx context.Context) *PubChan {
	if ret, ok := ctx.Value(CtxChan).(*PubChan); ok {
		return ret
	}
	return nil
}
