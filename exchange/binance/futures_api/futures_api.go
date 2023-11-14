package futures_api

import (
	"context"
	"errors"
	"high-freq-quant-go/adapter/timer"
	"math"
	"sync"
	"time"

	"high-freq-quant-go/adapter/convert"
	"high-freq-quant-go/adapter/mather"
	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/log"
	"high-freq-quant-go/exchange/binance/binanceapi/futures"
	"high-freq-quant-go/exchange/binance/unify"
)

type BinaceFuturesApi struct {
	Ctx context.Context
	Api *BinaceFuturesRequest

	//持仓模式
	DualSide bool
}

func NewBinanceApi(ctx context.Context) *BinaceFuturesApi {
	gf := &BinaceFuturesApi{
		Ctx:      ctx,
		Api:      NewBinaceFuturesRequest(ctx),
		DualSide: true,
	}
	return gf
}

func (bf *BinaceFuturesApi) CreateOrder(ctx context.Context) (*exch.Order, error) {
	o := exch.GetOrder(ctx)
	if o == nil {
		log.Errorln(log.Http, bf.Api.ApiSign, "BinaceFuturesApi CreateOrder get order error ")
		return nil, nil
	}
	client := bf.Api.GetClient()
	service := bf.CreateOrderService(client, o)
	if service == nil {
		return nil, nil
	}
	res, err := service.Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, "BinaceFuturesApi CreateOrder error ", err)
		return nil, err
	}
	status := unify.UnifyOrderStatus[res.Status]
	size := unify.QuantityToFloat(o.Symbol, res.OrigQuantity)
	lsize := size - unify.QuantityToFloat(o.Symbol, res.ExecutedQuantity)
	if res.Side == futures.SideTypeSell {
		size = -size
		lsize = -lsize
	}
	price := convert.GetFloat64(unify.PriceToStr(o.Symbol, res.Price))
	fprice := convert.GetFloat64(unify.PriceToStr(o.Symbol, res.AvgPrice))
	or := &exch.Order{
		Id:         convert.GetString(res.OrderID),
		UUID:       res.ClientOrderID,
		Symbol:     o.Symbol,
		Status:     status,
		Size:       size,
		Price:      price,
		FillPrice:  fprice,
		Left:       lsize,
		CreateTime: o.UpdateTime,
		UpdateTime: o.UpdateTime,
	}
	log.Infoln(log.Http, bf.Api.ApiSign, or.Symbol, "BinaceFuturesApi CreateOrder success:p,s ", or.Price, or.Size)
	return or, nil
}

func (bf *BinaceFuturesApi) CreateBatchOrder(ctx context.Context) ([]*exch.Order, error) {
	lists := exch.GetOrders(ctx)
	if lists == nil || len(lists) == 0 {
		log.Errorln(log.Http, bf.Api.ApiSign, "BinaceFuturesApi  CreateBatchOrder GetOrders error ", lists)
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
				order, err := bf.CreateLimitBatchOrder(octx)
				if err != nil {
					log.Errorf(log.Http, "%s BinaceFuturesApi CreateBatchOrder CreateLimitBatchOrder error %+v %s \r\n", bf.Api.ApiSign, tos, err)
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

func (bf *BinaceFuturesApi) CreateLimitBatchOrder(ctx context.Context) ([]*exch.Order, error) {
	lists := exch.GetOrders(ctx)
	if lists == nil || len(lists) == 0 {
		log.Errorln(log.Http, bf.Api.ApiSign, "BinaceFuturesApi CreateLimitBatchOrder get orders error ", lists)
		return nil, nil
	}
	client := bf.Api.GetClient()
	var sers []*futures.CreateOrderService
	for _, o := range lists {
		ser := bf.CreateOrderService(client, o)
		if ser == nil {
			continue
		}
		sers = append(sers, ser)
	}
	if len(sers) == 0 {
		return nil, nil
	}
	result, err := client.NewCreateBatchOrdersService().OrderList(sers).Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, "BinaceFuturesApi CreateLimitBatchOrder error", err)
		return nil, err
	}
	orders := []*exch.Order{}
	symbol := ""
	for _, res := range result.Orders {
		symbol = unify.BToSymbol(res.Symbol)
		status := unify.UnifyOrderStatus[res.Status]
		size := unify.QuantityToFloat(symbol, res.OrigQuantity)
		lsize := size - unify.QuantityToFloat(symbol, res.ExecutedQuantity)
		if res.Side == futures.SideTypeSell {
			size = -size
			lsize = -lsize
		}
		price := convert.GetFloat64(unify.PriceToStr(symbol, res.Price))
		fprice := convert.GetFloat64(unify.PriceToStr(symbol, res.AvgPrice))
		or := &exch.Order{
			Id:         convert.GetString(res.OrderID),
			UUID:       res.ClientOrderID,
			Symbol:     symbol,
			Status:     status,
			Size:       size,
			Price:      price,
			FillPrice:  fprice,
			Left:       lsize,
			CreateTime: res.Time,
			UpdateTime: res.UpdateTime,
		}
		log.Infoln(log.Http, bf.Api.ApiSign, or.Symbol, "BinaceFuturesApi CreateOrder success:p,s ", or.Price, or.Size)
		orders = append(orders, or)
	}
	log.Infoln(log.Http, bf.Api.ApiSign, symbol, "BinaceFuturesApi CreateLimitBatchOrder success ", len(orders))
	return orders, nil
}

func (bf *BinaceFuturesApi) CreateOrderService(client *futures.Client, o *exch.Order) *futures.CreateOrderService {
	bsymbol := unify.SymbolToB(o.Symbol)
	side := futures.SideTypeBuy
	if o.Size < 0 {
		side = futures.SideTypeSell
	}
	info, err := bf.GetBaseInfo(o.Symbol)
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, o.Symbol, "BinaceFuturesApi GetBaseInfo error", o, err)
		return nil
	}
	price := mather.FloatRound(o.Price, info.PriceFloat)
	size := mather.FloatRound(o.Size, info.SizeFloat)
	quantity := convert.GetString(math.Abs(size))
	service := client.NewCreateOrderService().Symbol(bsymbol).Side(side).Quantity(quantity)
	orderType := futures.OrderTypeMarket
	if price != 0 {
		orderType = futures.OrderTypeLimit
		orderTif := unify.GetOrderTif(o.Tif)
		service = service.Price(convert.GetString(price)).TimeInForce(orderTif)
	} else if o.Price != 0 {
		return nil
	}
	if o.UUID != "" {
		service.NewClientOrderID(o.UUID)
	}
	return service.Type(orderType).NewOrderResponseType(futures.NewOrderRespTypeRESULT)
}

func (bf *BinaceFuturesApi) CannelOrder(ctx context.Context) (*exch.Order, error) {
	o := exch.GetOrder(ctx)
	if o == nil {
		log.Errorln(log.Http, bf.Api.ApiSign, "BinaceFuturesApi CannelOrder GetOrder id error")
		return nil, nil
	}
	orderId := convert.GetInt64(o.Id)
	bsymbol := unify.SymbolToB(o.Symbol)
	res, err := bf.Api.GetClient().NewCancelOrderService().Symbol(bsymbol).OrderID(orderId).Do(context.Background())
	if err != nil {
		return nil, err
	}
	status := unify.UnifyOrderStatus[res.Status]
	size := unify.QuantityToFloat(o.Symbol, res.OrigQuantity)
	lsize := size - unify.QuantityToFloat(o.Symbol, res.ExecutedQuantity)
	if res.Side == futures.SideTypeSell {
		size = -size
		lsize = -lsize
	}
	price := convert.GetFloat64(unify.PriceToStr(o.Symbol, res.Price))
	or := &exch.Order{
		Id:         convert.GetString(res.OrderID),
		UUID:       res.ClientOrderID,
		Symbol:     o.Symbol,
		Status:     status,
		Size:       size,
		Price:      price,
		Left:       lsize,
		CreateTime: o.UpdateTime,
		UpdateTime: o.UpdateTime,
	}
	return or, nil
}

func (bf *BinaceFuturesApi) CannelAllOrder(ctx context.Context) ([]*exch.Order, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	bsymbol := unify.SymbolToB(symbol)
	err := bf.Api.GetClient().NewCancelAllOpenOrdersService().Symbol(bsymbol).Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, "BinaceFuturesApi  CannelAllOrder error", err)
		return nil, err
	}
	//log.Infoln(log.Http, bf.Api.ApiSign, bsymbol, "BinaceFuturesApi CannelAllOrder success ")
	return nil, nil
}

func (bf *BinaceFuturesApi) UpdateLeverage(ctx context.Context) (*exch.Position, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	bsymbol := unify.SymbolToB(symbol)
	lv := convert.GetInt(text.GetString(ctx, exch.CtxLv))
	_, err := bf.Api.GetClient().NewChangeLeverageService().Symbol(bsymbol).Leverage(lv).Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, symbol, "BinaceFuturesApi  UpdateLeverage error", err)
	}
	return nil, err
}

func (bf *BinaceFuturesApi) UpdateMarginType(ctx context.Context) (*exch.Position, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	bsymbol := unify.SymbolToB(symbol)
	marginType := futures.MarginTypeIsolated
	err := bf.Api.GetClient().NewChangeMarginTypeService().Symbol(bsymbol).MarginType(marginType).Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, symbol, "BinaceFuturesApi  UpdateMarginType error", err)
	}
	if err == nil || err.Error() == "<APIError> code=-4046, msg=No need to change margin type." {
		return nil, nil
	}
	return nil, err
}

func (bf *BinaceFuturesApi) UpdatePositionMode(ctx context.Context) error {
	if bf.DualSide == false {
		return nil
	}
	bf.DualSide = false
	err := bf.Api.GetClient().NewChangePositionModeService().DualSide(false).Do(context.Background())
	if err != nil {
		bf.DualSide = true
		log.Errorln(log.Http, bf.Api.ApiSign, "BinaceFuturesApi  UpdatePositionMode error", err)
	}
	return err
}

func (bf *BinaceFuturesApi) UpdateMargin(ctx context.Context) (*exch.Position, error) {
	change := text.GetFloat(ctx, exch.CtxChange)
	if change == 0.0 {
		log.Errorln(log.Http, bf.Api.ApiSign, "BinaceFuturesApi  UpdateMargin error: change is 0 ")
		return nil, nil
	}
	symbol := text.GetString(ctx, exch.CtxSymbol)
	bsymbol := unify.SymbolToB(symbol)

	amount := convert.GetString(change)
	actionType := 1
	if change < 0 {
		amount = convert.GetString(-change)
		actionType = 2
	}
	err := bf.Api.GetClient().NewUpdatePositionMarginService().Symbol(bsymbol).Amount(amount).Type(actionType).Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, "BinaceFuturesApi UpdateMargin error", err)
	}
	return nil, err
}

func (bf *BinaceFuturesApi) GetOrder(ctx context.Context) (map[string]*exch.Order, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	bsymbol := unify.SymbolToB(symbol)
	res, err := bf.Api.GetClient().NewListOpenOrdersService().Symbol(bsymbol).Do(context.Background())
	if err != nil {
		return nil, err
	}
	orders := map[string]*exch.Order{}
	for _, o := range res {
		symb := unify.BToSymbol(o.Symbol)
		status := unify.UnifyOrderStatus[o.Status]
		size := unify.QuantityToFloat(symb, o.OrigQuantity)
		lsize := size - unify.QuantityToFloat(symb, o.ExecutedQuantity)
		if o.Side == futures.SideTypeSell {
			size = -size
			lsize = -lsize
		}
		price := convert.GetFloat64(unify.PriceToStr(symb, o.Price))
		fprice := convert.GetFloat64(unify.PriceToStr(symb, o.AvgPrice))
		or := exch.Order{
			Id:         convert.GetString(o.OrderID),
			UUID:       o.ClientOrderID,
			Symbol:     symb,
			Status:     status,
			Size:       size,
			Price:      price,
			FillPrice:  fprice,
			Left:       lsize,
			Tif:        unify.UnifyOrderType[o.TimeInForce],
			CreateTime: o.UpdateTime,
			UpdateTime: o.UpdateTime,
		}
		orders[or.Id] = &or
	}
	return orders, nil
}

func (bf *BinaceFuturesApi) GetOrderBook(ctx context.Context) (*futures.DepthResponse, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	bsymbol := unify.SymbolToB(symbol)
	limit := text.GetInt64(ctx, OrderBookLimit)
	res, err := bf.Api.GetClient().NewDepthService().Symbol(bsymbol).Limit(int(limit)).Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, symbol, "BinaceFuturesApi GetOrderBook error", err)
	}
	return res, err
}

func (bf *BinaceFuturesApi) GetPosition(ctx context.Context) (*exch.Position, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	bsymbol := unify.SymbolToB(symbol)
	res, err := bf.Api.GetClient().NewGetPositionRiskService().Symbol(bsymbol).Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, symbol, "BinaceFuturesApi  GetPosition error", err)
		return nil, err
	}
	var pos *exch.Position
	for _, r := range res {
		price := convert.GetFloat64(r.EntryPrice)
		mprice := convert.GetFloat64(r.MarkPrice)
		lprice := convert.GetFloat64(r.LiquidationPrice)

		price = unify.PriceToFloat(symbol, price)
		mprice = unify.PriceToFloat(symbol, mprice)
		lprice = unify.PriceToFloat(symbol, lprice)
		size := unify.QuantityToFloat(symbol, r.PositionAmt)
		pos = &exch.Position{
			Symbol:         r.Symbol,
			Price:          price,
			Size:           size,
			Margin:         convert.GetFloat64(r.IsolatedMargin),
			UnPnl:          convert.GetFloat64(r.UnRealizedProfit),
			LiqPrice:       lprice,
			MarkPrice:      mprice,
			Lv:             convert.GetFloat64(r.Leverage),
			MarginType:     r.MarginType,
			PositionMode:   r.PositionSide,
			Value:          mprice * math.Abs(size),
			LastUpdateTime: r.UpdateTime,
		}
	}
	return pos, err
}

func (bf *BinaceFuturesApi) GetBalance(ctx context.Context) (map[string]*exch.Balance, error) {
	InitBalance.sc.Lock()
	defer InitBalance.sc.Unlock()
	ti := timer.MicNow()
	isinit := text.GetBool(ctx, exch.CtxInit)
	if isinit && ti-InitBalance.T < 3000 {
		return InitBalance.D, nil
	}
	blmap := map[string]*exch.Balance{}
	bl, err := bf.Api.GetClient().NewGetBalanceService().Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, "BinaceFuturesApi GetBalance error", err)
		return blmap, err
	}
	for _, a := range bl {
		blmap[a.Asset] = &exch.Balance{
			ApiSign: bf.Api.ApiSign,
			Total:   convert.GetFloat64(a.Balance),
			Avative: convert.GetFloat64(a.AvailableBalance),
		}
	}
	InitBalance.D = blmap
	InitBalance.T = ti
	return blmap, nil
}

func (bf *BinaceFuturesApi) ListPosition(ctx context.Context) (map[string]*exch.Position, error) {
	InitPosition.sc.Lock()
	defer InitPosition.sc.Unlock()
	ti := timer.MicNow()
	isinit := text.GetBool(ctx, exch.CtxInit)
	if isinit && ti-InitPosition.T < 2000 {
		return InitPosition.D, nil
	}
	res, err := bf.Api.GetClient().NewGetAccountService().Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, "BinaceFuturesApi ListPosition error", err)
		return nil, err
	}
	//bamap := map[string]*exch.Balance{}
	//for _, a := range res.Assets {
	//	bamap[a.Asset] = &exch.Balance{
	//		ApiSign: bf.Api.ApiSign,
	//		Total:   convert.GetFloat64(a.WalletBalance),
	//		Avative: convert.GetFloat64(a.MaxWithdrawAmount),
	//	}
	//}
	posList := map[string]*exch.Position{}
	for _, p := range res.Positions {
		symbol := unify.BToSymbol(p.Symbol)
		price := convert.GetFloat64(p.EntryPrice)
		price = unify.PriceToFloat(symbol, price)
		size := unify.QuantityToFloat(symbol, p.PositionAmt)
		mtype := exch.MarginIsolated
		if !p.Isolated {
			mtype = exch.MarginCrossed
		}
		pos := &exch.Position{
			Symbol:         symbol,
			Price:          price,
			Size:           size,
			Margin:         convert.GetFloat64(p.MaintMargin),
			UnPnl:          convert.GetFloat64(p.UnrealizedProfit),
			Lv:             convert.GetFloat64(p.Leverage),
			MarginType:     mtype,
			PositionMode:   unify.PosMap[p.PositionSide],
			Value:          price * math.Abs(size),
			LastUpdateTime: convert.GetInt64(p.UpdateTime),
		}
		posList[symbol] = pos
	}
	InitPosition.D = posList
	InitPosition.T = ti
	return posList, nil
}

func (bf *BinaceFuturesApi) GetMarketPrice(symbol string) (map[string]*exch.BaseInfo, error) {
	res, err := bf.Api.GetClient().NewPremiumIndexService().Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, symbol, "BinaceFuturesApi GetMarketPrice error", err)
		return nil, err
	}
	ti := time.Now().Unix()
	infos := map[string]*exch.BaseInfo{}
	for _, r := range res {
		info := &exch.BaseInfo{}
		s := unify.BToSymbol(r.Symbol)
		info.Symbol = s
		info.MarkPrice = convert.GetFloat64(r.MarkPrice)
		info.FundingRate = convert.GetFloat64(r.LastFundingRate)
		info.FundingNextApply = convert.GetFloat64(r.NextFundingTime)
		info.LastUpdateTime = ti
		infos[s] = info
	}
	//log.Warnln(log.Http, bf.Api.ApiSign, symbol, "BinaceFuturesApi GetMarketPrice success ")
	return infos, err
}

func (bf *BinaceFuturesApi) GetBaseInfo(symbol string) (*exch.BaseInfo, error) {
	InitInfo.sc.Lock()
	defer InitInfo.sc.Unlock()
	if res, ok := InitInfo.D.Get(symbol); ok {
		return res.(*exch.BaseInfo), nil
	}
	prices, err := bf.GetMarketPrice(symbol)
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, symbol, "BinaceFuturesApi GetMarketPrice error", err)
		return nil, err
	}
	res, err := bf.Api.GetClient().NewExchangeInfoService().Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bf.Api.ApiSign, symbol, "BinaceFuturesApi GetBaseInfo error", err)
		return nil, err
	}
	//log.Warnln(log.Http, bf.Api.ApiSign, symbol, "BinaceFuturesApi GetBaseInfo success ")
	ti := time.Now().Unix()
	infos := map[string]interface{}{}
	for _, r := range res.Symbols {
		info := &exch.BaseInfo{}
		s := unify.BToSymbol(r.Symbol)
		info.Symbol = s
		pf := r.PriceFilter()
		sf := r.LotSizeFilter()
		//mf := r.MinNotionalFilter()
		priceft := mather.GetFloatNum(pf.TickSize)
		sizeft := mather.GetFloatNum(sf.StepSize)
		minQty := convert.GetFloat64(sf.MinQuantity)
		info.Unit = minQty
		info.PriceFloat = priceft
		info.SizeFloat = sizeft
		info.MaxOrderPrice = convert.GetFloat64(pf.MaxPrice) * 0.95
		info.MinOrderPrice = convert.GetFloat64(pf.MinPrice) * 1.05
		info.MinPriceStep = convert.GetFloat64(pf.TickSize)
		info.MinSizeStep = convert.GetFloat64(sf.StepSize)
		info.TakerFeeRate = TakerFeeRate
		info.MakerFeeRate = MakerFeeRate
		if _, ok := prices[s]; ok {
			info.MarkPrice = prices[s].MarkPrice
			info.FundingRate = prices[s].FundingRate
			info.FundingNextApply = prices[s].FundingNextApply
		}
		info.LastUpdateTime = ti
		infos[s] = info
	}
	InitInfo.D.MSet(infos)
	if info, ok := InitInfo.D.Get(symbol); ok {
		return info.(*exch.BaseInfo), nil
	}
	return nil, errors.New("no symbol info")
}

func (bf *BinaceFuturesApi) GetSignSymbol(symbol string) string {
	return bf.Api.ApiSign + "-" + symbol
}
