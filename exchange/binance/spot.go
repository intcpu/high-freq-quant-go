package binance

import (
	"context"
	"sync"

	"high-freq-quant-go/exchange/binance/spot_api"
	"high-freq-quant-go/exchange/binance/spot_wss"

	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/core/exch"
)

var SpotKey = exch.Binance + "_" + exch.Spot

type SpotClient struct {
	ApiSign string
	Ctx     context.Context
	Api     *spot_api.BinaceSpotApi
	PubWss  *spot_wss.Futures
	PriWss  *spot_wss.UserWss

	Exchange, Extype string

	pul, prl sync.Mutex
}

func NewSpotClient(ctx context.Context) exch.Exchange {
	sign := text.GetString(ctx, exch.ApiSign)
	ft := &SpotClient{
		ApiSign:  sign,
		Ctx:      ctx,
		Exchange: exch.Binance,
		Extype:   exch.Spot,
	}
	ft.Api = spot_api.NewBinanceApi(ctx)
	return ft
}

func (mk *SpotClient) StartWss() error {
	mk.pul.Lock()
	defer mk.pul.Unlock()
	if mk.PubWss == nil {
		mk.PubWss = spot_wss.NewFutures(mk.Ctx)
	}
	return nil
}

func (mk *SpotClient) StartUser() error {
	mk.prl.Lock()
	defer mk.prl.Unlock()
	if mk.PriWss == nil {
		mk.PriWss = spot_wss.NewFuturesUser(mk.Ctx)
	}
	return nil
}

func (mk *SpotClient) GetApiSign() string {
	return mk.ApiSign
}

func (mk *SpotClient) GetExName() string {
	return mk.Exchange
}

func (mk *SpotClient) GetExType() string {
	return mk.Extype
}

func (mk *SpotClient) GetStatus() int {
	if mk.PubWss != nil && mk.PubWss.GetStatus() == 0 {
		return 0
	}
	if mk.PriWss != nil && mk.PriWss.GetStatus() == 0 {
		return 0
	}
	return 1
}

func (mk *SpotClient) GetBaseInfo(ctx context.Context) *exch.BaseInfo {
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

func (mk *SpotClient) GetBalance(ctx context.Context) *exch.Balance {
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

func (mk *SpotClient) GetPosition(ctx context.Context) *exch.Position {
	res, err := mk.Api.GetPosition(ctx)
	if err != nil {
		return nil
	}
	return res
}

func (mk *SpotClient) GetOrder(ctx context.Context) map[string]*exch.Order {
	if mk.PriWss != nil {
		return mk.PriWss.GetOrder(ctx)
	}
	res, err := mk.Api.GetOrder(ctx)
	if err != nil {
		return nil
	}
	return res
}

func (mk *SpotClient) GetOrderBook(ctx context.Context) *exch.Booker {
	book := mk.PubWss.GetBook(ctx)
	return book
}

func (mk *SpotClient) ListAsset(ctx context.Context) (*exch.Balance, map[string]*exch.Position) {
	bas, err := mk.Api.GetBalance(ctx)
	if err != nil {
		return nil, nil
	}
	if _, ok := bas[spot_api.AssetUsdt]; !ok {
		bas[spot_api.AssetUsdt] = &exch.Balance{}
	}
	pos := bas[spot_api.AssetUsdt]
	res, err := mk.Api.ListPosition(ctx)
	if err != nil {
		return nil, nil
	}
	return pos, res
}

func (mk *SpotClient) CreateOrder(ctx context.Context) (*exch.Order, error) {
	res, err := mk.Api.CreateOrder(ctx)
	return res, err
}

func (mk *SpotClient) CreateBatchOrder(ctx context.Context) ([]*exch.Order, error) {
	res, err := mk.Api.CreateBatchOrder(ctx)
	return res, err
}

func (mk *SpotClient) CannelOrder(ctx context.Context) (*exch.Order, error) {
	res, err := mk.Api.CannelOrder(ctx)
	return res, err
}

func (mk *SpotClient) CannelAllOrder(ctx context.Context) ([]*exch.Order, error) {
	res, err := mk.Api.CannelAllOrder(ctx)
	return res, err
}

func (mk *SpotClient) UpdateLeverage(ctx context.Context) (*exch.Position, error) {
	return nil, nil
}

func (mk *SpotClient) UpdateMargin(ctx context.Context) (*exch.Position, error) {
	return nil, nil
}

func (mk *SpotClient) SubTicker(ctx context.Context) error {
	err := mk.StartWss()
	if err != nil {
		return err
	}
	mk.PubWss.SubTicker(ctx)
	return nil
}
func (mk *SpotClient) SubOrderBook(ctx context.Context) error {
	err := mk.StartWss()
	if err != nil {
		return err
	}
	mk.PubWss.SubOrderBook(ctx)
	return nil
}

func (mk *SpotClient) GetTradeChan(ctx context.Context) *chan *exch.Order {
	mk.StartUser()
	return mk.PriWss.GetTradeChan(ctx)
}

func (mk *SpotClient) SubUserTrade(ctx context.Context) error {
	mk.StartUser()
	return mk.PriWss.SubUserTrade(ctx)
}

func (mk *SpotClient) SubPosition(ctx context.Context) error {
	mk.StartUser()
	return mk.PriWss.SubPosition(ctx)
}

func (mk *SpotClient) SubBalance(ctx context.Context) error {
	mk.StartUser()
	return mk.PriWss.SubBalance(ctx)
}

func (mk *SpotClient) SubOrder(ctx context.Context) error {
	mk.StartUser()
	return mk.PriWss.SubOrder(ctx)
}

func init() {
	exch.Register(SpotKey, NewSpotClient)
}
