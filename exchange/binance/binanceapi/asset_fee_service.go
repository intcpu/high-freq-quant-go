package binanceapi

import (
	"context"
	"encoding/json"
	"net/http"
)

// GetAssetFeeService user trade fee
//
// See https://binance-docs.github.io/apidocs/spot/cn/#user_data-13
type GetAssetFeeService struct {
	c      *Client
	symbol *string
}

// Asset sets the asset parameter.
func (s *GetAssetFeeService) Symbol(symbol string) *GetAssetFeeService {
	s.symbol = &symbol
	return s
}

// Do sends the request.
func (s *GetAssetFeeService) Do(ctx context.Context) (res []*TradeFee, err error) {
	r := &request{
		method:   http.MethodGet,
		endpoint: "/sapi/v1/asset/tradeFee",
		secType:  secTypeSigned,
	}
	if s.symbol != nil {
		r.setParam("symbol", *s.symbol)
	}
	data, err := s.c.callAPI(ctx, r)
	if err != nil {
		return
	}
	res = make([]*TradeFee, 0)
	err = json.Unmarshal(data, &res)
	if err != nil {
		return
	}
	return res, nil
}

type TradeFee struct {
	Symbol          string `json:"symbol"`
	MakerCommission string `json:"makerCommission"`
	TakerCommission string `json:"takerCommission"`
}
