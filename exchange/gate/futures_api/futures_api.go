package futures_api

import (
	"context"
	"math"
	"sync"
	"time"

	"high-freq-quant-go/adapter/timer"

	"high-freq-quant-go/adapter/mather"

	"high-freq-quant-go/adapter/text"

	"high-freq-quant-go/core/log"
	"high-freq-quant-go/exchange/gate/gateapi"

	"github.com/antihax/optional"

	"high-freq-quant-go/adapter/convert"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/exchange/gate/unify"
)

type GateFuturesApi struct {
	Ctx  context.Context
	Api  *GateApiRequest
	lock sync.Mutex
}

func NewGateFuturesApi(ctx context.Context) *GateFuturesApi {
	gf := &GateFuturesApi{
		Ctx: ctx,
		Api: NewGateApiClient(ctx),
	}
	return gf
}

func (gf *GateFuturesApi) CreateOrder(ctx context.Context) (*exch.Order, error) {
	o := exch.GetOrder(ctx)
	if o == nil {
		log.Errorln(log.Http, gf.Api.ApiSign, "GateFuturesApi CreateOrder get order error ")
		return nil, nil
	}
	settle := unify.Settle(o.Symbol)
	unit := gf.GetUnit(o.Symbol)
	pn := gf.GetPriceFloat(o.Symbol)
	isize := int64(o.Size / unit)
	//todo too min size
	if o.Size != 0 && isize == 0 {
		isize = int64(o.Size / math.Abs(o.Size))
	}
	futuresOrder := gateapi.FuturesOrder{
		Contract: o.Symbol,
		Size:     isize,
	}
	price := mather.FloatRound(o.Price, pn)
	if price > 0 {
		futuresOrder.Price = convert.GetString(price)
	} else {
		if o.Price != 0 {
			return nil, nil
		}
	}
	if o.Tif != "" {
		futuresOrder.Tif = o.Tif
	}
	if o.Tif == exch.OrderIoc {
		futuresOrder.Price = "0"
	}
	if o.UUID != "" {
		futuresOrder.Text = OrderPre + o.UUID
	}
	if o.Iceberg != 0 {
		futuresOrder.Iceberg = o.Iceberg
	}
	res, _, err := gf.Api.GetClient().FuturesApi.CreateFuturesOrder(gf.Api.Ctx, settle, futuresOrder)
	if err != nil {
		log.Errorf(log.Http, "%s gate GateFuturesApi CreateOrder error %s %+v \r\n", gf.Api.ApiSign, err, futuresOrder)
		return nil, err
	}
	fsize := unify.UnitSize(float64(res.Size), unit)
	fleft := unify.UnitSize(float64(res.Left), unit)
	ro := &exch.Order{
		Id:         convert.GetString(res.Id),
		UUID:       res.Text,
		Symbol:     res.Contract,
		Status:     res.Status,
		Size:       fsize,
		Price:      convert.GetFloat64(res.Price),
		FillPrice:  convert.GetFloat64(res.FillPrice),
		Left:       fleft,
		CreateTime: int64(res.CreateTime * 1000),
		UpdateTime: int64(res.FinishTime * 1000),
	}
	log.Infoln(log.Http, gf.Api.ApiSign, o.Symbol, "GateFuturesApi CreateOrder success:p,s", ro.Price, ro.Size)
	return ro, nil
}

func (gf *GateFuturesApi) CreateBatchOrder(ctx context.Context) ([]*exch.Order, error) {
	lists := exch.GetOrders(ctx)
	if lists == nil || len(lists) == 0 {
		log.Errorln(log.Http, gf.Api.ApiSign, "gate CreateBatchOrder GetOrders error ", lists)
		return nil, nil
	}
	orders := []*exch.Order{}
	var wg sync.WaitGroup
	symbol := ""
	for _, o := range lists {
		symbol = o.Symbol
		wg.Add(1)
		go func(o *exch.Order) {
			defer wg.Done()
			octx := context.WithValue(context.Background(), exch.CtxOrder, o)
			order, err := gf.CreateOrder(octx)
			if err != nil {
				return
			}
			orders = append(orders, order)
		}(o)
	}
	wg.Wait()
	log.Infoln(log.Http, gf.Api.ApiSign, symbol, "GateFuturesApi CreateBatchOrder success ", len(lists))
	return orders, nil
}

func (gf *GateFuturesApi) CannelOrder(ctx context.Context) (*exch.Order, error) {
	o := exch.GetOrder(ctx)
	if o == nil {
		log.Errorln(log.Http, gf.Api.ApiSign, "GateFuturesApi CannelOrder  GetOrder id error")
		return nil, nil
	}
	settle := unify.Settle(o.Symbol)
	res, _, err := gf.Api.GetClient().FuturesApi.CancelFuturesOrder(gf.Api.Ctx, settle, o.Id)
	if err != nil {
		log.Errorln(log.Http, gf.Api.ApiSign, "GateFuturesApi CannelOrder error ", err)
		return nil, err
	}
	unit := gf.GetUnit(o.Symbol)
	fsize := float64(res.Size) * unit
	fleft := float64(res.Left) * unit
	ro := &exch.Order{
		Id:         convert.GetString(res.Id),
		UUID:       res.Text,
		Symbol:     res.Contract,
		Status:     res.Status,
		Size:       fsize,
		Price:      convert.GetFloat64(res.Price),
		FillPrice:  convert.GetFloat64(res.FillPrice),
		Left:       fleft,
		CreateTime: int64(res.CreateTime),
		UpdateTime: int64(res.FinishTime),
	}
	return ro, nil
}

func (gf *GateFuturesApi) CannelAllOrder(ctx context.Context) ([]*exch.Order, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	settle := unify.Settle(symbol)
	res, _, err := gf.Api.GetClient().FuturesApi.CancelFuturesOrders(gf.Api.Ctx, settle, symbol, &gateapi.CancelFuturesOrdersOpts{})
	if err != nil {
		log.Errorln(log.Http, gf.Api.ApiSign, symbol, "GateFuturesApi CannelAllOrder error ", err)
		return nil, err
	}
	unit := gf.GetUnit(symbol)
	lists := []*exch.Order{}
	for _, s := range res {
		fsize := float64(s.Size) * unit
		fleft := float64(s.Left) * unit
		ro := &exch.Order{
			Id:         convert.GetString(s.Id),
			UUID:       s.Text,
			Symbol:     s.Contract,
			Status:     s.Status,
			Size:       fsize,
			Price:      convert.GetFloat64(s.Price),
			FillPrice:  convert.GetFloat64(s.FillPrice),
			Left:       fleft,
			CreateTime: int64(s.CreateTime),
			UpdateTime: int64(s.FinishTime),
		}
		lists = append(lists, ro)
	}
	//log.Warnln(log.Http, gf.Api.ApiSign, symbol, "GateFuturesApi CannelAllOrder success ", len(lists))
	return lists, nil
}

func (gf *GateFuturesApi) UpdateLeverage(ctx context.Context) (*exch.Position, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	lv := text.GetString(ctx, exch.CtxLv)
	settle := unify.Settle(symbol)
	res, _, err := gf.Api.GetClient().FuturesApi.UpdatePositionLeverage(gf.Api.Ctx, settle, symbol, lv, &gateapi.UpdatePositionLeverageOpts{})
	if err != nil {
		log.Errorln(log.Http, gf.Api.ApiSign, "GateFuturesApi UpdateLeverage error ", err)
		return nil, err
	}
	ti := timer.MicNow()
	mtype := exch.MarginIsolated
	if res.Leverage == "0" {
		mtype = exch.MarginCrossed
	}
	pos := &exch.Position{
		Symbol:         res.Contract,
		Price:          convert.GetFloat64(res.EntryPrice),
		Size:           convert.GetFloat64(res.Size),
		Margin:         convert.GetFloat64(res.Margin),
		UnPnl:          convert.GetFloat64(res.UnrealisedPnl),
		LiqPrice:       convert.GetFloat64(res.LiqPrice),
		MarkPrice:      convert.GetFloat64(res.MarkPrice),
		Lv:             convert.GetFloat64(res.Leverage),
		MarginType:     mtype,
		PositionMode:   unify.PositionMap[res.Mode],
		Value:          convert.GetFloat64(res.Value),
		LastUpdateTime: ti,
	}
	return pos, nil
}

func (gf *GateFuturesApi) UpdatePositionMode(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	settle := unify.Settle(symbol)
	_, _, err := gf.Api.GetClient().FuturesApi.SetDualMode(gf.Api.Ctx, settle, false)
	if err != nil {
		log.Errorln(log.Http, gf.Api.ApiSign, "GateFuturesApi UpdateLeverage error ", err)
	}
	return nil
}

func (gf *GateFuturesApi) UpdateMargin(ctx context.Context) (*exch.Position, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	change := text.GetFloat(ctx, exch.CtxChange)
	if change == 0.0 {
		log.Errorln(log.Http, "GateFuturesApi UpdateMargin error: change is 0 ")
		return nil, nil
	}
	ch := convert.GetString(change)
	settle := unify.Settle(symbol)
	res, _, err := gf.Api.GetClient().FuturesApi.UpdatePositionMargin(gf.Api.Ctx, settle, symbol, ch)
	if err != nil {
		log.Errorln(log.Http, gf.Api.ApiSign, "GateFuturesApi UpdateMargin error ", err)
		return nil, err
	}
	ti := time.Now().Unix()
	mtype := exch.MarginIsolated
	if res.Leverage == "0" {
		mtype = exch.MarginCrossed
	}
	pos := &exch.Position{
		Symbol:         res.Contract,
		Price:          convert.GetFloat64(res.EntryPrice),
		Size:           convert.GetFloat64(res.Size),
		Margin:         convert.GetFloat64(res.Margin),
		UnPnl:          convert.GetFloat64(res.UnrealisedPnl),
		LiqPrice:       convert.GetFloat64(res.LiqPrice),
		MarkPrice:      convert.GetFloat64(res.MarkPrice),
		Lv:             convert.GetFloat64(res.Leverage),
		MarginType:     mtype,
		PositionMode:   unify.PositionMap[res.Mode],
		Value:          convert.GetFloat64(res.Value),
		LastUpdateTime: ti,
	}
	return pos, nil
}

func (gf *GateFuturesApi) GetOrder(ctx context.Context) (map[string]*exch.Order, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	settle := unify.Settle(symbol)
	res, _, err := gf.Api.GetClient().FuturesApi.ListFuturesOrders(gf.Api.Ctx, settle, symbol, OpenOrder, nil)
	if err != nil {
		log.Errorln(log.Http, gf.Api.ApiSign, "GateFuturesApi GetOrder error ", err)
		return nil, err
	}
	unit := gf.GetUnit(symbol)
	ti := time.Now().Unix()
	orders := map[string]*exch.Order{}
	for _, r := range res {
		size := unify.UnitSize(float64(r.Size), unit)
		left := unify.UnitSize(float64(r.Left), unit)
		order := &exch.Order{
			Id:         convert.GetString(r.Id),
			UUID:       r.Text,
			Symbol:     r.Contract,
			Status:     r.Status,
			Size:       size,
			Price:      convert.GetFloat64(r.Price),
			FillPrice:  convert.GetFloat64(r.FillPrice),
			Left:       left,
			Iceberg:    r.Iceberg,
			Tif:        r.Tif,
			CreateTime: int64(r.CreateTime),
			UpdateTime: ti,
		}
		orders[order.Id] = order
	}
	return orders, nil
}

func (gf *GateFuturesApi) GetOrderBook(ctx context.Context) (*gateapi.FuturesOrderBook, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	limit := text.GetString(ctx, OrderBookLimit)
	settle := unify.Settle(symbol)
	limit32 := int32(convert.GetInt64(limit))
	localVarOptionals := &gateapi.ListFuturesOrderBookOpts{
		Limit:  optional.NewInt32(limit32),
		WithId: optional.NewBool(true),
	}
	result, _, err := gf.Api.GetClient().FuturesApi.ListFuturesOrderBook(gf.Api.Ctx, settle, symbol, localVarOptionals)
	if err != nil || result.Asks == nil || len(result.Asks) == 0 {
		if e, ok := err.(gateapi.GateAPIError); ok {
			log.Errorln(log.Http, "GateFuturesApi GetOrderBook error ", e.Error())
		} else {
			log.Errorln(log.Http, "GateFuturesApi GetOrderBook data null ", err.Error())
		}
		return nil, err
	}
	return &result, nil
}

func (gf *GateFuturesApi) GetPosition(ctx context.Context) (*exch.Position, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	settle := unify.Settle(symbol)
	res, _, err := gf.Api.GetClient().FuturesApi.GetPosition(gf.Api.Ctx, settle, symbol)
	if err != nil {
		log.Errorln(log.Http, "GateFuturesApi GetPosition error ", err)
		return nil, err
	}
	ti := timer.MicNow()
	unit := gf.GetUnit(symbol)
	size := unify.UnitSize(float64(res.Size), unit)
	mtype := exch.MarginIsolated
	if res.Leverage == "0" {
		mtype = exch.MarginCrossed
	}
	pos := &exch.Position{
		Symbol:         res.Contract,
		Price:          convert.GetFloat64(res.EntryPrice),
		Size:           size,
		Margin:         convert.GetFloat64(res.Margin),
		UnPnl:          convert.GetFloat64(res.UnrealisedPnl),
		LiqPrice:       convert.GetFloat64(res.LiqPrice),
		MarkPrice:      convert.GetFloat64(res.MarkPrice),
		Lv:             convert.GetFloat64(res.Leverage),
		MarginType:     mtype,
		PositionMode:   unify.PositionMap[res.Mode],
		Value:          convert.GetFloat64(res.Value),
		LastUpdateTime: ti,
	}
	return pos, nil
}

func (gf *GateFuturesApi) GetBalance(ctx context.Context) (map[string]*exch.Balance, error) {
	InitBalance.sc.Lock()
	defer InitBalance.sc.Unlock()
	ti := timer.MicNow()
	isinit := text.GetBool(ctx, exch.CtxInit)
	if isinit && ti-InitBalance.T < 5000 {
		return InitBalance.D, nil
	}
	blmap := map[string]*exch.Balance{}
	res, _, err := gf.Api.GetClient().WalletApi.GetTotalBalance(gf.Api.Ctx, nil)
	if err != nil {
		log.Errorln(log.Http, "GateFuturesApi GetBalance error ", err)
		return blmap, err
	}
	bl := &exch.Balance{}
	bl.ApiSign = gf.Api.ApiSign
	if fu, ok := res.Details[exch.Futures]; ok {
		bl.Asset = fu.Currency
		bl.Total = convert.GetFloat64(fu.Amount)
		bl.Avative = convert.GetFloat64(fu.Amount)
	}
	blmap[AssetUsdt] = bl
	InitBalance.D = blmap
	InitBalance.T = ti
	return blmap, nil
}

func (gf *GateFuturesApi) ListPosition(ctx context.Context) (map[string]*exch.Position, error) {
	InitPosition.sc.Lock()
	defer InitPosition.sc.Unlock()
	ti := timer.MicNow()
	isinit := text.GetBool(ctx, exch.CtxInit)
	if isinit && ti-InitPosition.T < 5000 {
		return InitPosition.D, nil
	}
	res, _, err := gf.Api.GetClient().FuturesApi.ListPositions(gf.Api.Ctx, UsdtUrl)
	if err != nil {
		log.Errorln(log.Http, "GateFuturesApi ListPosition error ", err)
		return nil, err
	}
	posList := map[string]*exch.Position{}
	for _, s := range res {
		unit := gf.GetUnit(s.Contract)
		size := unify.UnitSize(float64(s.Size), unit)
		mtype := exch.MarginIsolated
		if s.Leverage == "0" {
			mtype = exch.MarginCrossed
		}
		pos := &exch.Position{
			Symbol:         s.Contract,
			Price:          convert.GetFloat64(s.EntryPrice),
			Size:           size,
			Margin:         convert.GetFloat64(s.Margin),
			UnPnl:          convert.GetFloat64(s.UnrealisedPnl),
			LiqPrice:       convert.GetFloat64(s.LiqPrice),
			MarkPrice:      convert.GetFloat64(s.MarkPrice),
			Lv:             convert.GetFloat64(s.Leverage),
			MarginType:     mtype,
			PositionMode:   unify.PositionMap[s.Mode],
			Value:          convert.GetFloat64(s.Value),
			LastUpdateTime: ti,
		}
		posList[s.Contract] = pos
	}
	InitPosition.D = posList
	InitPosition.T = ti
	return posList, nil
}

func (gf *GateFuturesApi) GetBaseInfo(symbol string) (*exch.BaseInfo, error) {
	InitInfo.sc.Lock()
	defer InitInfo.sc.Unlock()
	skey := gf.GetSignSymbol(symbol)
	if info, ok := InitInfo.D.Get(skey); ok {
		return info.(*exch.BaseInfo), nil
	}
	settle := unify.Settle(symbol)
	res, _, err := gf.Api.GetClient().FuturesApi.ListFuturesContracts(gf.Api.Ctx, settle)
	if err != nil {
		log.Errorln(log.Http, "GateFuturesApi GetBaseInfo error ", err)
		return nil, err
	}
	//log.Warnln(log.Http, "GateFuturesApi GetBaseInfo success ", symbol)
	ti := timer.MicNow()
	infos := map[string]interface{}{}
	for _, i := range res {
		priceft := mather.GetFloatNum(i.OrderPriceRound)
		unit := convert.GetFloat64(i.QuantoMultiplier)
		sizeft := mather.GetFloatNum(unit)
		deviate := convert.GetFloat64(i.OrderPriceDeviate)
		markPrice := convert.GetFloat64(i.MarkPrice)
		info := &exch.BaseInfo{
			Symbol:           i.Name,
			Unit:             unit,
			PriceFloat:       priceft,
			SizeFloat:        sizeft,
			MinPriceStep:     convert.GetFloat64(i.OrderPriceRound),
			MinSizeStep:      convert.GetFloat64(i.OrderSizeMin) * unit,
			MaxOrderPrice:    markPrice + (markPrice * deviate * 0.95),
			MinOrderPrice:    markPrice - (markPrice * deviate * 1.05),
			MarkPrice:        markPrice,
			IndexPrice:       convert.GetFloat64(i.IndexPrice),
			FundingRate:      convert.GetFloat64(i.FundingRate),
			FundingNextApply: convert.GetFloat64(i.FundingNextApply),
			TakerFeeRate:     convert.GetFloat64(i.TakerFeeRate),
			MakerFeeRate:     convert.GetFloat64(i.MakerFeeRate),
			LastUpdateTime:   ti,
		}
		tkey := gf.GetSignSymbol(i.Name)
		infos[tkey] = info
	}
	InitInfo.D.MSet(infos)
	if info, ok := infos[skey]; ok {
		return info.(*exch.BaseInfo), nil
	}
	return nil, nil
}

func (gf *GateFuturesApi) GetSignSymbol(symbol string) string {
	return gf.Api.ApiSign + "-" + symbol
}

func (gf *GateFuturesApi) GetUnit(symbol string) float64 {
	info, err := gf.GetBaseInfo(symbol)
	if err != nil || info == nil {
		log.Errorln(log.Http, "GateFuturesApi GetUnit error ")
		return 0
	}
	return info.Unit
}

func (gf *GateFuturesApi) GetPriceFloat(symbol string) int {
	info, err := gf.GetBaseInfo(symbol)
	if err != nil || info == nil {
		log.Errorln(log.Http, "GateFuturesApi GetUnit error ")
		return 0
	}
	return info.PriceFloat
}
