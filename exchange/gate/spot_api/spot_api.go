package spot_api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"

	"high-freq-quant-go/core/redis"

	"high-freq-quant-go/adapter/mather"

	"high-freq-quant-go/exchange/gate/unify"

	"github.com/antihax/optional"
	"high-freq-quant-go/adapter/convert"
	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/adapter/timer"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/log"
	"high-freq-quant-go/exchange/gate/gateapi"
)

type GateSpotApi struct {
	Ctx context.Context
	Api *GateApiRequest
}

func NewGateSpotApi(ctx context.Context) *GateSpotApi {
	gf := &GateSpotApi{
		Ctx: ctx,
		Api: NewGateApiClient(ctx),
	}
	return gf
}

func (gs *GateSpotApi) CreateOrder(ctx context.Context) (*exch.Order, error) {
	o := exch.GetOrder(ctx)
	if o == nil {
		log.Errorln(log.Http, gs.Api.ApiSign, "GateSpotApi CreateOrder get order error ")
		return nil, nil
	}
	pn := gs.GetPriceFloat(o.Symbol)
	sn := gs.GetSizeFloat(o.Symbol)
	price := mather.FloatRound(o.Price, pn)
	size := mather.FloatFloor(o.Size, sn)
	size = math.Abs(size)
	if size == 0 {
		log.Errorln(log.Http, gs.Api.ApiSign, o.Symbol, "GateSpotApi CreateOrder size is 0 ", o.Price, o.Size)
		return nil, nil
	}
	side := unify.SideBuy
	if o.Size < 0 {
		side = unify.SideSell
	}
	opt := gateapi.Order{
		CurrencyPair: o.Symbol,
		Account:      exch.Spot,
		Amount:       convert.GetString(size),
		Type:         exch.OrderLimit,
		Side:         side,
		Price:        convert.GetString(price),
	}
	if o.Tif != "" {
		opt.TimeInForce = o.Tif
	}
	if o.UUID != "" {
		opt.Text = OrderPre + o.UUID
	}
	if o.Iceberg != 0 {
		opt.Iceberg = convert.GetString(o.Iceberg)
	}
	res, _, err := gs.Api.GetSpotClient().CreateOrder(gs.Api.Ctx, opt)
	if err != nil {
		log.Errorf(log.Http, "%s %s gate GateSpotApi CreateOrder error %s %+v \r\n", gs.Api.ApiSign, o.Symbol, err, opt)
		return nil, err
	}
	symbol := res.CurrencyPair
	status := unify.SpotOrderMap[res.Status]
	amount := convert.GetFloat64(res.Amount)
	left := convert.GetFloat64(res.Left)
	if res.Side == unify.SideSell {
		amount = -amount
	}
	or := &exch.Order{
		Id:         res.Id,
		Symbol:     symbol,
		UUID:       res.Text,
		Price:      convert.GetFloat64(res.Price),
		Status:     status,
		Size:       amount,
		Left:       left,
		FillPrice:  convert.GetFloat64(res.FillPrice),
		CreateTime: res.CreateTimeMs,
		UpdateTime: res.UpdateTimeMs,
	}
	log.Infoln(log.Http, gs.Api.ApiSign, o.Symbol, "GateSpotApi CreateOrder success:p,s", or.Price, or.Size)
	return or, nil
}

func (gs *GateSpotApi) CreateBatchOrder(ctx context.Context) ([]*exch.Order, error) {
	lists := exch.GetOrders(ctx)
	if lists == nil || len(lists) == 0 {
		log.Errorln(log.Http, gs.Api.ApiSign, "GateSpotApi CreateBatchOrder GetOrders error ", lists)
		return nil, nil
	}

	l := len(lists) - 1
	orders := []*exch.Order{}
	tos, os := []*exch.Order{}, []*exch.Order{}
	var wg sync.WaitGroup
	for i, o := range lists {
		os = append(os, o)
		if (i+1)%MaxBatchOrderNum == 0 || i == l {
			tos = os
			os = []*exch.Order{}
		}
		if len(tos) > 0 {
			wg.Add(1)
			go func(wg *sync.WaitGroup, tos []*exch.Order) {
				defer wg.Done()
				octx := context.WithValue(context.Background(), exch.CtxOrders, tos)
				order, err := gs.CreateLimitBatchOrder(octx)
				if err != nil {
					log.Errorf(log.Http, "%s GateSpotApi CreateBatchOrder CreateLimitBatchOrder error %+v %s \r\n", gs.Api.ApiSign, tos, err)
					return
				}
				orders = append(orders, order...)
			}(&wg, tos)
			tos = []*exch.Order{}
		}
	}
	wg.Wait()
	return orders, nil
}

func (gs *GateSpotApi) CreateLimitBatchOrder(ctx context.Context) ([]*exch.Order, error) {
	lists := exch.GetOrders(ctx)
	if lists == nil || len(lists) == 0 {
		log.Errorln(log.Http, gs.Api.ApiSign, "GateSpotApi CreateLimitBatchOrder get orders error ", lists)
		return nil, nil
	}
	opts := []gateapi.Order{}
	for _, o := range lists {
		pn := gs.GetPriceFloat(o.Symbol)
		sn := gs.GetSizeFloat(o.Symbol)
		price := mather.FloatRound(o.Price, pn)
		amount := mather.FloatRound(o.Size, sn)
		amount = math.Abs(amount)
		if amount == 0 {
			log.Errorln(log.Http, gs.Api.ApiSign, o.Symbol, "GateSpotApi CreateOrder size is 0 ", o.Price, o.Size)
			continue
		}
		side := unify.SideBuy
		if o.Size < 0 {
			side = unify.SideSell
		}
		size := convert.GetString(amount)
		opt := gateapi.Order{
			CurrencyPair: o.Symbol,
			Account:      exch.Spot,
			Amount:       size,
			Type:         exch.OrderLimit,
			Side:         side,
			Price:        convert.GetString(price),
		}
		if o.Tif != "" {
			opt.TimeInForce = o.Tif
		}
		opt.Text = OrderPre
		if o.UUID != "" {
			opt.Text += o.UUID
		} else {
			opt.Text += exch.GetUUID(o.Symbol)
		}
		if o.Iceberg != 0 {
			opt.Iceberg = convert.GetString(o.Iceberg)
		}
		opts = append(opts, opt)
	}
	if len(opts) == 0 {
		log.Errorln(log.Http, gs.Api.ApiSign, "GateSpotApi CreateLimitBatchOrder orders len is 0  ")
		return nil, nil
	}
	client := gs.Api.GetSpotClient()
	result, _, err := client.CreateBatchOrders(gs.Api.Ctx, opts)
	if err != nil {
		log.Errorln(log.Http, gs.Api.ApiSign, "GateSpotApi CreateLimitBatchOrder error", err)
		return nil, err
	}
	symbol := ""
	orders := []*exch.Order{}
	for _, res := range result {
		symbol = res.CurrencyPair
		status := unify.SpotOrderMap[res.Status]
		size := convert.GetFloat64(res.Amount)
		left := convert.GetFloat64(res.Left)
		price := convert.GetFloat64(res.Price)
		if res.Side == unify.SideSell {
			size = -size
		}
		or := &exch.Order{
			Id:         convert.GetString(res.Id),
			Symbol:     res.CurrencyPair,
			UUID:       res.Text,
			Price:      price,
			Status:     status,
			Size:       size,
			Left:       left,
			FillPrice:  convert.GetFloat64(res.FillPrice),
			CreateTime: res.CreateTimeMs,
			UpdateTime: res.UpdateTimeMs,
		}
		log.Infoln(log.Http, gs.Api.ApiSign, or.Symbol, "GateSpotApi CreateOrder success:p,s ", or.Price, or.Size)
		orders = append(orders, or)
	}
	log.Infoln(log.Http, gs.Api.ApiSign, symbol, "GateSpotApi CreateLimitBatchOrder success ", len(orders))
	return orders, nil
}

func (gs *GateSpotApi) CannelOrder(ctx context.Context) (*exch.Order, error) {
	o := exch.GetOrder(ctx)
	if o == nil {
		log.Errorln(log.Http, gs.Api.ApiSign, "GateSpotApi CannelOrder  GetOrder id error")
		return nil, nil
	}
	localVarOptionals := &gateapi.CancelOrderOpts{}
	res, _, err := gs.Api.GetSpotClient().CancelOrder(gs.Api.Ctx, o.Id, o.Symbol, localVarOptionals)
	if err != nil {
		log.Errorln(log.Http, gs.Api.ApiSign, "GateSpotApi CannelOrder error ", err)
		return nil, err
	}
	status := unify.SpotOrderMap[res.Status]
	size := convert.GetFloat64(res.Amount)
	left := convert.GetFloat64(res.Left)
	price := convert.GetFloat64(res.Price)
	if res.Side == unify.SideSell {
		size = -size
	}
	or := &exch.Order{
		Id:         convert.GetString(res.Id),
		UUID:       res.Text,
		Price:      price,
		Status:     status,
		Size:       size,
		Left:       left,
		FillPrice:  convert.GetFloat64(res.FillPrice),
		CreateTime: res.CreateTimeMs,
		UpdateTime: res.UpdateTimeMs,
	}
	return or, nil
}

func (gs *GateSpotApi) CannelAllOrder(ctx context.Context) ([]*exch.Order, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	localVarOptionals := &gateapi.CancelOrdersOpts{}
	res, _, err := gs.Api.GetSpotClient().CancelOrders(gs.Api.Ctx, symbol, localVarOptionals)
	if err != nil {
		log.Errorln(log.Http, gs.Api.ApiSign, symbol, "GateSpotApi CannelAllOrder error ", err)
		return nil, err
	}
	lists := []*exch.Order{}
	for _, s := range res {
		status := unify.SpotOrderMap[s.Status]
		size := convert.GetFloat64(s.Amount)
		left := convert.GetFloat64(s.Left)
		price := convert.GetFloat64(s.Price)
		if s.Side == unify.SideSell {
			size = -size
		}
		ro := &exch.Order{
			Id:         convert.GetString(s.Id),
			UUID:       s.Text,
			Price:      price,
			Status:     status,
			Size:       size,
			Left:       left,
			FillPrice:  convert.GetFloat64(s.FillPrice),
			CreateTime: s.CreateTimeMs,
			UpdateTime: s.UpdateTimeMs,
		}
		lists = append(lists, ro)
	}
	//log.Warnln(log.Http, gf.Api.ApiSign, symbol, "GateSpotApi CannelAllOrder success ", len(lists))
	return lists, nil
}

func (gs *GateSpotApi) GetOrder(ctx context.Context) (map[string]*exch.Order, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	opts := &gateapi.ListOrdersOpts{}
	res, _, err := gs.Api.GetSpotClient().ListOrders(gs.Api.Ctx, symbol, unify.OrderOpen, opts)
	if err != nil {
		log.Errorln(log.Http, gs.Api.ApiSign, "GateSpotApi GetOrder error ", err)
		return nil, err
	}
	orders := map[string]*exch.Order{}
	for _, o := range res {
		status := unify.SpotOrderMap[o.Status]
		size := convert.GetFloat64(o.Amount)
		left := convert.GetFloat64(o.Left)
		price := convert.GetFloat64(o.Price)
		if o.Side == unify.SideSell {
			size = -size
		}
		order := &exch.Order{
			Id:         o.Id,
			Symbol:     o.CurrencyPair,
			Size:       size,
			Price:      price,
			Status:     status,
			FillPrice:  convert.GetFloat64(o.FillPrice),
			Left:       left,
			Text:       o.Text,
			CreateTime: o.CreateTimeMs,
			UpdateTime: o.UpdateTimeMs,
		}
		orders[o.Id] = order
	}
	return orders, nil
}

func (gs *GateSpotApi) GetOrderBook(ctx context.Context) (*gateapi.OrderBook, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	limit := text.GetString(ctx, OrderBookLimit)
	limit32 := int32(convert.GetInt64(limit))
	localVarOptionals := &gateapi.ListOrderBookOpts{
		Limit:  optional.NewInt32(limit32),
		WithId: optional.NewBool(true),
	}
	result, _, err := gs.Api.GetSpotClient().ListOrderBook(gs.Api.Ctx, symbol, localVarOptionals)
	if err != nil || result.Asks == nil || len(result.Asks) == 0 {
		if e, ok := err.(gateapi.GateAPIError); ok {
			log.Errorln(log.Http, "GateSpotApi GetOrderBook error ", e.Error())
		} else {
			log.Errorln(log.Http, "GateSpotApi GetOrderBook data null ", err.Error())
		}
		return nil, err
	}
	return &result, nil
}

func (gs *GateSpotApi) GetPosition(ctx context.Context) (*exch.Position, error) {
	pos := &exch.Position{}
	symbol := text.GetString(ctx, exch.CtxSymbol)
	rd := redis.BookRedis()
	hkey := fmt.Sprintf(redis.SPOT_POSITION_HKEY, gs.Api.ApiSign, symbol)
	result, err := rd.Hget(redis.SPOT_POSITION_KEY, hkey)
	//no spot cache
	if err != nil && err.Error() == "redigo: nil returned" {
		return pos, nil
	}
	if err != nil {
		log.Warnln(log.Http, "GateSpotApi GetPosition Hget data error ", err)
		return pos, err
	}
	err = json.Unmarshal([]byte(result), pos)
	if err != nil {
		log.Errorln(log.Http, "GateSpotApi GetPosition Unmarshal pos data error ", result, err)
		return nil, err
	}
	return pos, nil
}

func (gs *GateSpotApi) GetBalance(ctx context.Context) (map[string]*exch.Balance, error) {
	InitBalance.sc.Lock()
	defer InitBalance.sc.Unlock()
	ti := timer.MicNow()
	isinit := text.GetBool(ctx, exch.CtxInit)
	if isinit && ti-InitBalance.T < 5000 {
		return InitBalance.D, nil
	}
	res, _, err := gs.Api.GetClient().SpotApi.ListSpotAccounts(gs.Api.Ctx, nil)
	if err != nil {
		log.Errorln(log.Http, "GateSpotApi GetBalance error ", err)
		return nil, err
	}
	bl := map[string]*exch.Balance{}
	for _, r := range res {
		avative := convert.GetFloat64(r.Available)
		lock := convert.GetFloat64(r.Locked)
		total := avative + lock
		bl[r.Currency] = &exch.Balance{ApiSign: gs.Api.ApiSign, Asset: r.Currency, Total: total, Avative: avative}
	}
	InitBalance.D = bl
	InitBalance.T = ti
	return bl, nil
}

func (gs *GateSpotApi) ListPosition(ctx context.Context) (map[string]*exch.Position, error) {
	InitPosition.sc.Lock()
	defer InitPosition.sc.Unlock()
	ti := timer.MicNow()
	isinit := text.GetBool(ctx, exch.CtxInit)
	if isinit && ti-InitPosition.T < 2000 {
		return InitPosition.D, nil
	}
	posList := map[string]*exch.Position{}
	rd := redis.BookRedis()
	result, err := rd.Hgetall(redis.SPOT_POSITION_KEY)
	if err != nil {
		log.Errorln(log.Http, "GateSpotApi ListPosition Hgetall error ", err)
		return nil, err
	}
	for key, val := range result {
		ks := strings.Split(key, ":")
		if len(ks) < 2 {
			return nil, errors.New("redis hash key error")
		}
		if ks[0] != gs.Api.ApiSign {
			continue
		}
		k := ks[1]
		pos := &exch.Position{}
		err = json.Unmarshal([]byte(val), pos)
		if err != nil {
			log.Errorln(log.Http, "GateSpotApi ListPosition Unmarshal pos data error ", val, err)
			return nil, err
		}
		posList[k] = pos
	}
	InitPosition.D = posList
	InitPosition.T = ti
	return posList, nil
}

func (gs *GateSpotApi) GetFee() (float64, float64) {
	localVarOptionals := &gateapi.GetFeeOpts{}
	feeRes, _, err := gs.Api.GetSpotClient().GetFee(gs.Api.Ctx, localVarOptionals)
	takerFeeRate, makerFeeRate := 0.0, 0.0
	if err == nil {
		takerFeeRate = convert.GetFloat64(feeRes.TakerFee)
		makerFeeRate = convert.GetFloat64(feeRes.TakerFee)
	} else {
		log.Errorln(log.Http, "GateSpotApi GetFee error:", err)
	}
	return takerFeeRate, makerFeeRate
}

func (gs *GateSpotApi) GetBaseInfo(symbol string) (*exch.BaseInfo, error) {
	InitInfo.sc.Lock()
	defer InitInfo.sc.Unlock()
	skey := gs.GetSignSymbol(symbol)
	if info, ok := InitInfo.D.Get(skey); ok {
		return info.(*exch.BaseInfo), nil
	}
	takerFeeRate, makerFeeRate := gs.GetFee()
	res, _, err := gs.Api.GetSpotClient().ListCurrencyPairs(gs.Api.Ctx)
	if err != nil {
		log.Errorln(log.Http, "GateSpotApi GetBaseInfo error ", err)
		return nil, err
	}
	//todo fee rate
	//log.Warnln(log.Http, "GateSpotApi GetBaseInfo success ", symbol)
	infos := map[string]interface{}{}
	ti := timer.MicNow()
	for _, r := range res {
		fee := convert.GetFloat64(r.Fee)
		if takerFeeRate == 0 {
			takerFeeRate, makerFeeRate = fee, fee
		}
		priceft := convert.GetInt(r.Precision)
		sizeft := convert.GetInt(r.AmountPrecision)
		sizeStep, priceStep := 1.0, 1.0
		if sizeft > 0 {
			sizeStep = float64(1 / sizeft)
		}
		if priceft > 0 {
			priceStep = float64(1 / priceft)
		}
		minBase := convert.GetFloat64(r.MinBaseAmount)
		minQuete := convert.GetFloat64(r.MinQuoteAmount)
		info := &exch.BaseInfo{
			Symbol:         r.Id,
			Base:           r.Base,
			Quote:          r.Quote,
			MinBase:        minBase,
			MinQuote:       minQuete,
			PriceFloat:     priceft,
			SizeFloat:      sizeft,
			MinSizeStep:    sizeStep,
			MinPriceStep:   priceStep,
			TakerFeeRate:   takerFeeRate,
			MakerFeeRate:   makerFeeRate,
			LastUpdateTime: ti,
		}
		tkey := gs.GetSignSymbol(r.Id)
		infos[tkey] = info
	}
	InitInfo.D.MSet(infos)
	if info, ok := infos[skey]; ok {
		return info.(*exch.BaseInfo), nil
	}
	return nil, nil
}

func (gs *GateSpotApi) GetSignSymbol(symbol string) string {
	return gs.Api.ApiSign + "-" + symbol
}

func (gs *GateSpotApi) GetPriceFloat(symbol string) int {
	info, err := gs.GetBaseInfo(symbol)
	if err != nil || info == nil {
		log.Errorln(log.Http, "GateSpotApi GetUnit error ")
		return 0
	}
	return info.PriceFloat
}

func (gs *GateSpotApi) GetSizeFloat(symbol string) int {
	info, err := gs.GetBaseInfo(symbol)
	if err != nil || info == nil {
		log.Errorln(log.Http, "GateSpotApi GetUnit error ")
		return 0
	}
	return info.SizeFloat
}
