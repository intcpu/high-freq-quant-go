package spot_api

import (
	"context"

	"high-freq-quant-go/exchange/binance/binanceapi"

	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/request"
)

type BinaceSpotRequest struct {
	ApiSign string

	Ctx    context.Context
	Client *binanceapi.Client

	Times *request.Times
}

func NewBinaceRequest(ctx context.Context) *BinaceSpotRequest {
	InitApier.sc.Lock()
	defer InitApier.sc.Unlock()
	sign := text.GetString(ctx, exch.ApiSign)
	if api, ok := InitApier.D.Get(sign); ok {
		return api.(*BinaceSpotRequest)
	}
	key := text.GetString(ctx, exch.Key)
	secret := text.GetString(ctx, exch.Secret)

	bc := &BinaceSpotRequest{
		ApiSign: sign,
		Times:   TimeLimiter,
		Ctx:     context.Background(),
		Client:  binanceapi.NewClient(key, secret),
	}
	InitApier.D.Set(sign, bc)
	return bc
}

func (bc *BinaceSpotRequest) GetClient() *binanceapi.Client {
	bc.Times.Update()
	return bc.Client
}
