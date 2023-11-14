package gate

import (
	"context"
	"errors"

	"high-freq-quant-go/core/log"

	"high-freq-quant-go/adapter/text"

	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/exchange/gate/futures_api"
	"high-freq-quant-go/exchange/gate/futures_wss"
)

var FuturesKey = exch.Gate + "_" + exch.Futures

type Futures struct {
	ApiSign string
	Ctx     context.Context
	Wss     *futures_wss.Futures
	Api     *futures_api.GateFuturesApi

	Exchange, Extype string
}

func NewFuturesClient(ctx context.Context) exch.Exchange {
	sign := text.GetString(ctx, exch.ApiSign)
	ft := &Futures{
		ApiSign:  sign,
		Ctx:      ctx,
		Exchange: exch.Gate,
		Extype:   exch.Futures,
	}
	ft.Api = futures_api.NewGateFuturesApi(ctx)
	return ft
}

func (mk *Futures) WssStart() error {
	if mk.Wss == nil {
		mk.Wss = futures_wss.NewGateFuturesWss(mk.Ctx)
	}
	if mk.Wss == nil {
		return errors.New("wss start error")
	}
	return nil
}

func (mk *Futures) GetStatus() int {
	return mk.Wss.GetStatus()
}

func (mk *Futures) GetApiSign() string {
	return mk.ApiSign
}

func (mk *Futures) GetExName() string {
	return mk.Exchange
}

func (mk *Futures) GetExType() string {
	return mk.Extype
}

func (mk *Futures) GetBaseInfo(ctx context.Context) *exch.BaseInfo {
	if mk.Wss == nil {
		symbol := text.GetString(ctx, exch.CtxSymbol)
		res, err := mk.Api.GetBaseInfo(symbol)
		if err != nil {
			log.Warnln(log.Global, mk.ApiSign, symbol, "GetBaseInfo error")
			return nil
		}
		return res
	}
	return mk.Wss.GetBaseInfo(ctx)
}

func (mk *Futures) GetPosition(ctx context.Context) *exch.Position {
	return mk.Wss.GetPosition(ctx)
}

func (mk *Futures) GetOrder(ctx context.Context) map[string]*exch.Order {
	if mk.Wss != nil {
		return mk.Wss.GetOrders(ctx)
	}
	res, err := mk.Api.GetOrder(ctx)
	if err != nil {
		return nil
	}
	return res
}

func (mk *Futures) GetOrderBook(ctx context.Context) *exch.Booker {
	return mk.Wss.GetBook(ctx)
}

func (mk *Futures) GetBalance(ctx context.Context) *exch.Balance {
	return mk.Wss.GetBalance(ctx)
}

func (mk *Futures) GetTradeChan(ctx context.Context) *chan *exch.Order {
	return mk.Wss.GetTradeChan(ctx)
}

func (mk *Futures) ListAsset(ctx context.Context) (*exch.Balance, map[string]*exch.Position) {
	ctx = context.WithValue(ctx, exch.CtxInit, true)
	bas, err := mk.Api.GetBalance(ctx)
	if err != nil {
		return nil, nil
	}
	if _, ok := bas[futures_api.AssetUsdt]; !ok {
		return nil, nil
	}
	bl := bas[futures_api.AssetUsdt]
	res, err := mk.Api.ListPosition(ctx)
	if err != nil {
		return nil, nil
	}
	return bl, res
}

func (mk *Futures) CreateOrder(ctx context.Context) (*exch.Order, error) {
	res, err := mk.Api.CreateOrder(ctx)
	return res, err
}

func (mk *Futures) CreateBatchOrder(ctx context.Context) ([]*exch.Order, error) {
	res, err := mk.Api.CreateBatchOrder(ctx)
	return res, err
}

func (mk *Futures) CannelOrder(ctx context.Context) (*exch.Order, error) {
	res, err := mk.Api.CannelOrder(ctx)
	return res, err
}

func (mk *Futures) CannelAllOrder(ctx context.Context) ([]*exch.Order, error) {
	res, err := mk.Api.CannelAllOrder(ctx)
	return res, err
}

func (mk *Futures) UpdateLeverage(ctx context.Context) (*exch.Position, error) {
	res, err := mk.Api.UpdateLeverage(ctx)
	return res, err
}

func (mk *Futures) UpdateMargin(ctx context.Context) (*exch.Position, error) {
	res, err := mk.Api.UpdateMargin(ctx)
	return res, err
}

func (mk *Futures) SubTicker(ctx context.Context) error {
	err := mk.WssStart()
	if err != nil {
		return err
	}
	return mk.Wss.SubTicker(ctx)
}

func (mk *Futures) SubOrderBook(ctx context.Context) error {
	err := mk.WssStart()
	if err != nil {
		return err
	}
	return mk.Wss.SubOrderBook(ctx)
}

func (mk *Futures) SubUserTrade(ctx context.Context) error {
	err := mk.WssStart()
	if err != nil {
		return err
	}
	return mk.Wss.SubUserTrade(ctx)
}

func (mk *Futures) SubPosition(ctx context.Context) error {
	err := mk.WssStart()
	if err != nil {
		return err
	}
	err = mk.Wss.SubTicker(ctx)
	if err != nil {
		return err
	}
	return mk.Wss.SubPosition(ctx)
}

func (mk *Futures) SubBalance(ctx context.Context) error {
	err := mk.WssStart()
	if err != nil {
		return err
	}
	return mk.Wss.SubBalance(ctx)
}

func (mk *Futures) SubOrder(ctx context.Context) error {
	err := mk.WssStart()
	if err != nil {
		return err
	}
	return mk.Wss.SubOrder(ctx)
}

func init() {
	exch.Register(FuturesKey, NewFuturesClient)
}
