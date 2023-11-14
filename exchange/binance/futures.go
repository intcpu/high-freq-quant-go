package binance

import (
	"context"
	"sync"

	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/exchange/binance/futures_api"
	"high-freq-quant-go/exchange/binance/futures_wss"
)

var DefaultKey = exch.Binance + "_" + exch.Futures

type Futures struct {
	ApiSign string
	Ctx     context.Context
	Api     *futures_api.BinaceFuturesApi
	PubWss  *futures_wss.Futures
	PriWss  *futures_wss.UserWss

	Exchange, Extype string

	pul, prl sync.Mutex
}

func NewClient(ctx context.Context) exch.Exchange {
	sign := text.GetString(ctx, exch.ApiSign)
	ft := &Futures{
		ApiSign:  sign,
		Ctx:      ctx,
		Exchange: exch.Binance,
		Extype:   exch.Futures,
	}
	ft.Api = futures_api.NewBinanceApi(ctx)
	return ft
}

func (mk *Futures) StartWss() error {
	mk.pul.Lock()
	defer mk.pul.Unlock()
	if mk.PubWss == nil {
		mk.PubWss = futures_wss.NewFutures(mk.Ctx)
	}
	return nil
}

func (mk *Futures) StartUser() error {
	mk.prl.Lock()
	defer mk.prl.Unlock()
	if mk.PriWss == nil {
		mk.PriWss = futures_wss.NewFuturesUser(mk.Ctx)
	}
	return nil
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

func (mk *Futures) GetStatus() int {
	if mk.PubWss != nil && mk.PubWss.GetStatus() == 0 {
		return 0
	}
	if mk.PriWss != nil && mk.PriWss.GetStatus() == 0 {
		return 0
	}
	return 1
}

func (mk *Futures) GetBaseInfo(ctx context.Context) *exch.BaseInfo {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	http := text.GetBool(ctx, exch.CtxHttp)
	if !http && mk.PubWss != nil {
		return mk.PubWss.GetBaseinfo(ctx)
	}
	res, err := mk.Api.GetBaseInfo(symbol)
	if err != nil {
		return nil
	}
	return res
}

func (mk *Futures) GetBalance(ctx context.Context) *exch.Balance {
	if mk.PriWss != nil {
		return mk.PriWss.GetBalance(ctx)
	}
	res, err := mk.Api.GetBalance(ctx)
	if err != nil {
		return nil
	}
	base := text.GetString(ctx, exch.CtxBase)
	if _, ok := res[base]; ok {
		return res[base]
	}
	return nil
}

func (mk *Futures) GetPosition(ctx context.Context) *exch.Position {
	http := text.GetBool(ctx, exch.CtxHttp)
	if !http && mk.PriWss != nil {
		return mk.PriWss.GetPosition(ctx)
	}
	res, err := mk.Api.GetPosition(ctx)
	if err != nil {
		return nil
	}
	return res
}

func (mk *Futures) GetOrder(ctx context.Context) map[string]*exch.Order {
	if mk.PriWss != nil {
		return mk.PriWss.GetOrder(ctx)
	}
	res, err := mk.Api.GetOrder(ctx)
	if err != nil {
		return nil
	}
	return res
}

func (mk *Futures) GetOrderBook(ctx context.Context) *exch.Booker {
	book := mk.PubWss.GetBook(ctx)
	return book
}

func (mk *Futures) ListAsset(ctx context.Context) (*exch.Balance, map[string]*exch.Position) {
	bas, err := mk.Api.GetBalance(ctx)
	if err != nil {
		return nil, nil
	}
	if _, ok := bas[futures_api.AssetUsdt]; !ok {
		bas[futures_api.AssetUsdt] = &exch.Balance{}
	}
	bls := bas[futures_api.AssetUsdt]
	pos, err := mk.Api.ListPosition(ctx)
	if err != nil {
		return nil, nil
	}
	return bls, pos
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
	//err := ft.Futures.UpdatePositionMode(ctx)
	//if err != nil {
	//	return nil, err
	//}
	_, err := mk.Api.UpdateMarginType(ctx)
	if err != nil {
		return nil, err
	}
	res, err := mk.Api.UpdateLeverage(ctx)
	if err != nil {
		return res, err
	}
	return res, err
}

func (mk *Futures) UpdateMargin(ctx context.Context) (*exch.Position, error) {
	res, err := mk.Api.UpdateMargin(ctx)
	return res, err
}

func (mk *Futures) SubTicker(ctx context.Context) error {
	err := mk.StartWss()
	if err != nil {
		return err
	}
	mk.PubWss.SubTicker(ctx)
	return nil
}
func (mk *Futures) SubOrderBook(ctx context.Context) error {
	err := mk.StartWss()
	if err != nil {
		return err
	}
	mk.PubWss.SubOrderBook(ctx)
	return nil
}

func (mk *Futures) GetTradeChan(ctx context.Context) *chan *exch.Order {
	mk.StartUser()
	return mk.PriWss.GetTradeChan(ctx)
}

func (mk *Futures) SubUserTrade(ctx context.Context) error {
	mk.StartUser()
	return mk.PriWss.SubUserTrade(ctx)
}

func (mk *Futures) SubPosition(ctx context.Context) error {
	mk.StartUser()
	return mk.PriWss.SubPosition(ctx)
}

func (mk *Futures) SubBalance(ctx context.Context) error {
	mk.StartUser()
	return mk.PriWss.SubBalance(ctx)
}

func (mk *Futures) SubOrder(ctx context.Context) error {
	mk.StartUser()
	return mk.PriWss.SubOrder(ctx)
}

func init() {
	exch.Register(DefaultKey, NewClient)
}
