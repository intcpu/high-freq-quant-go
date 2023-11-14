package spot_api

import (
	"context"
	"high-freq-quant-go/core/request"

	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/exchange/gate/gateapi"
)

type GateApiRequest struct {
	ApiSign string

	Ctx    context.Context
	Client *gateapi.APIClient

	Times *request.Times
}

func NewGateApiClient(ctx context.Context) *GateApiRequest {
	InitApier.sc.Lock()
	defer InitApier.sc.Unlock()
	sign := text.GetString(ctx, exch.ApiSign)
	if api, ok := InitApier.D.Get(sign); ok {
		return api.(*GateApiRequest)
	}
	gc := &GateApiRequest{
		ApiSign: sign,
		Times:   request.NewTimes(),
		Ctx:     context.Background(),
		Client:  gateapi.NewAPIClient(gateapi.NewConfiguration()),
	}
	key := text.GetString(ctx, exch.Key)
	sevret := text.GetString(ctx, exch.Secret)
	if key != "" && sevret != "" {
		gc.Ctx = context.WithValue(context.Background(), gateapi.ContextGateAPIV4, gateapi.GateAPIV4{
			Key:    key,
			Secret: sevret,
		})
	}
	InitApier.D.Set(sign, gc)
	return gc
}

func (gc *GateApiRequest) GetClient() *gateapi.APIClient {
	gc.Times.Update()
	return gc.Client
}

func (gc *GateApiRequest) GetFuturesClient() *gateapi.FuturesApiService {
	gc.Times.Update()
	return gc.Client.FuturesApi
}

func (gc *GateApiRequest) GetSpotClient() *gateapi.SpotApiService {
	gc.Times.Update()
	return gc.Client.SpotApi
}

func (gc *GateApiRequest) GetWalletClient() *gateapi.WalletApiService {
	gc.Times.Update()
	return gc.Client.WalletApi
}
