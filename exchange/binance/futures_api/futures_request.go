package futures_api

import (
	"context"

	"high-freq-quant-go/exchange/binance/binanceapi"

	"high-freq-quant-go/exchange/binance/binanceapi/futures"

	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/request"
)

type BinaceFuturesRequest struct {
	ApiSign string

	Ctx    context.Context
	Client *futures.Client

	Times *request.Times
}

func NewBinaceFuturesRequest(ctx context.Context) *BinaceFuturesRequest {
	InitApier.sc.Lock()
	defer InitApier.sc.Unlock()
	sign := text.GetString(ctx, exch.ApiSign)
	if api, ok := InitApier.D.Get(sign); ok {
		return api.(*BinaceFuturesRequest)
	}
	key := text.GetString(ctx, exch.Key)
	secret := text.GetString(ctx, exch.Secret)

	bc := &BinaceFuturesRequest{
		ApiSign: sign,
		Times:   TimeLimiter,
		Ctx:     context.Background(),
		Client:  binanceapi.NewFuturesClient(key, secret),
	}

	InitApier.D.Set(sign, bc)
	return bc
}

func (bc *BinaceFuturesRequest) GetClient() *futures.Client {
	bc.Times.Update()
	return bc.Client
}
