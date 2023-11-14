package gate

import (
	"context"
	"errors"

	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/core/log"

	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/exchange/gate/spot_api"
	"high-freq-quant-go/exchange/gate/spot_wss"
)

var SpotKey = exch.Gate + "_" + exch.Spot

type Spot struct {
	ApiSign string
	Ctx     context.Context
	Wss     *spot_wss.SpotWss
	Api     *spot_api.GateSpotApi

	Exchange, Extype string
}

func NewSpotClient(ctx context.Context) exch.Exchange {
	sign := text.GetString(ctx, exch.ApiSign)
	ft := &Spot{
		ApiSign:  sign,
		Ctx:      ctx,
		Exchange: exch.Gate,
		Extype:   exch.Spot,
	}
	ft.Api = spot_api.NewGateSpotApi(ctx)
	return ft
}

func (mk *Spot) WssStart() error {
	if mk.Wss == nil {
		mk.Wss = spot_wss.NewGateSpotWss(mk.Ctx)
	}
	if mk.Wss == nil {
		return errors.New("wss start error")
	}
	return nil
}

func (mk *Spot) GetApiSign() string {
	return mk.ApiSign
}

func (mk *Spot) GetExName() string {
	return mk.Exchange
}

func (mk *Spot) GetExType() string {
	return mk.Extype
}

func (mk *Spot) GetStatus() int {
	return mk.Wss.GetStatus()
}

func (mk *Spot) GetBaseInfo(ctx context.Context) *exch.BaseInfo {
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

func (mk *Spot) GetPosition(ctx context.Context) *exch.Position {
	res := mk.Wss.GetPosition(ctx)
	return res
}

func (mk *Spot) GetOrder(ctx context.Context) map[string]*exch.Order {
	if mk.Wss != nil {
		return mk.Wss.GetOrders(ctx)
	}
	res, err := mk.Api.GetOrder(ctx)
	if err != nil {
		return nil
	}
	return res
}

func (mk *Spot) GetOrderBook(ctx context.Context) *exch.Booker {
	return mk.Wss.GetBook(ctx)
}

func (mk *Spot) GetBalance(ctx context.Context) *exch.Balance {
	return mk.Wss.GetBalance(ctx)
}

func (mk *Spot) GetTradeChan(ctx context.Context) *chan *exch.Order {
	return mk.Wss.GetTradeChan(ctx)
}

func (mk *Spot) ListAsset(ctx context.Context) (*exch.Balance, map[string]*exch.Position) {
	bas, err := mk.Api.GetBalance(ctx)
	if err != nil {
		return nil, nil
	}
	if _, ok := bas[spot_api.AssetUsdt]; !ok {
		return nil, nil
	}
	pos := bas[spot_api.AssetUsdt]
	res, err := mk.Api.ListPosition(ctx)
	if err != nil {
		return nil, nil
	}
	return pos, res
}

func (mk *Spot) CreateOrder(ctx context.Context) (*exch.Order, error) {
	res, err := mk.Api.CreateOrder(ctx)
	return res, err
}

func (mk *Spot) CreateBatchOrder(ctx context.Context) ([]*exch.Order, error) {
	res, err := mk.Api.CreateBatchOrder(ctx)
	return res, err
}

func (mk *Spot) CannelOrder(ctx context.Context) (*exch.Order, error) {
	res, err := mk.Api.CannelOrder(ctx)
	return res, err
}

func (mk *Spot) CannelAllOrder(ctx context.Context) ([]*exch.Order, error) {
	res, err := mk.Api.CannelAllOrder(ctx)
	return res, err
}

func (mk *Spot) UpdateLeverage(ctx context.Context) (*exch.Position, error) {
	return nil, nil
}

func (mk *Spot) UpdateMargin(ctx context.Context) (*exch.Position, error) {
	return nil, nil
}

func (mk *Spot) SubTicker(ctx context.Context) error {
	err := mk.WssStart()
	if err != nil {
		return err
	}
	return mk.Wss.SubTicker(ctx)
}

func (mk *Spot) SubOrderBook(ctx context.Context) error {
	err := mk.WssStart()
	if err != nil {
		return err
	}
	return mk.Wss.SubOrderBook(ctx)
}

func (mk *Spot) SubUserTrade(ctx context.Context) error {
	err := mk.WssStart()
	if err != nil {
		return err
	}
	return mk.Wss.SubUserTrade(ctx)
}

func (mk *Spot) SubPosition(ctx context.Context) error {
	err := mk.WssStart()
	if err != nil {
		return err
	}
	return mk.Wss.SubPosition(ctx)
}

func (mk *Spot) SubBalance(ctx context.Context) error {
	err := mk.WssStart()
	if err != nil {
		return err
	}
	return mk.Wss.SubBalance(ctx)
}

func (mk *Spot) SubOrder(ctx context.Context) error {
	err := mk.WssStart()
	if err != nil {
		return err
	}
	return mk.Wss.SubOrder(ctx)
}

func init() {
	exch.Register(SpotKey, NewSpotClient)
}
