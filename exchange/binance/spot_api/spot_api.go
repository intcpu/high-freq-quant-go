package spot_api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"high-freq-quant-go/adapter/timer"
	"math"
	"strings"
	"sync"
	"time"

	"high-freq-quant-go/adapter/convert"
	"high-freq-quant-go/adapter/mather"
	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/log"
	"high-freq-quant-go/core/redis"
	"high-freq-quant-go/exchange/binance/binanceapi"
	"high-freq-quant-go/exchange/binance/unify"
)

type BinaceSpotApi struct {
	Ctx context.Context
	Api *BinaceSpotRequest
}

func NewBinanceApi(ctx context.Context) *BinaceSpotApi {
	gf := &BinaceSpotApi{
		Ctx: ctx,
		Api: NewBinaceRequest(ctx),
	}
	return gf
}

func (bs *BinaceSpotApi) CreateOrder(ctx context.Context) (*exch.Order, error) {
	o := exch.GetOrder(ctx)
	if o == nil {
		log.Errorln(log.Http, bs.Api.ApiSign, "BinaceSpotApi CreateOrder get order error ")
		return nil, nil
	}
	client := bs.Api.GetClient()
	service := bs.CreateOrderService(client, o)
	if service == nil {
		return nil, nil
	}
	res, err := service.Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bs.Api.ApiSign, "BinaceSpotApi CreateOrder error ", err)
		return nil, err
	}
	size := unify.QuantityToFloat(o.Symbol, res.ExecutedQuantity)
	lsize := unify.QuantityToFloat(o.Symbol, res.OrigQuantity) - size
	price := convert.GetFloat64(unify.PriceToStr(o.Symbol, res.Price))
	if res.Side == binanceapi.SideTypeSell {
		size = -size
	}
	or := &exch.Order{
		Id:         convert.GetString(res.OrderID),
		UUID:       res.ClientOrderID,
		Symbol:     o.Symbol,
		Size:       size,
		Price:      price,
		Left:       lsize,
		CreateTime: o.UpdateTime,
		UpdateTime: o.UpdateTime,
	}
	log.Infoln(log.Http, bs.Api.ApiSign, or.Symbol, "BinaceSpotApi CreateOrder success:p,s ", or.Price, or.Size)
	return or, nil
}

func (bs *BinaceSpotApi) CreateBatchOrder(ctx context.Context) ([]*exch.Order, error) {
	lists := exch.GetOrders(ctx)
	if lists == nil || len(lists) == 0 {
		log.Errorln(log.Http, bs.Api.ApiSign, "BinaceSpotApi  CreateBatchOrder GetOrders error ", lists)
		return nil, nil
	}
	orders := []*exch.Order{}
	var wg sync.WaitGroup
	for _, o := range lists {
		wg.Add(1)
		go func(wg *sync.WaitGroup, o *exch.Order, orders []*exch.Order) {
			defer wg.Done()
			octx := context.WithValue(context.Background(), exch.CtxOrder, o)
			order, err := bs.CreateOrder(octx)
			if err != nil {
				log.Errorf(log.Http, "%s BinaceSpotApi CreateBatchOrder CreateOrder error %+v %s \r\n", bs.Api.ApiSign, err)
				return
			}
			orders = append(orders, order)
		}(&wg, o, orders)
	}
	wg.Wait()
	return orders, nil
}

func (bs *BinaceSpotApi) CreateOrderService(client *binanceapi.Client, o *exch.Order) *binanceapi.CreateOrderService {
	bsymbol := unify.SymbolToB(o.Symbol)
	info, err := bs.GetBaseInfo(o.Symbol)
	if err != nil {
		log.Errorln(log.Http, bs.Api.ApiSign, o.Symbol, "BinaceSpotApi GetBaseInfo error", o, err)
		return nil
	}
	price := mather.FloatRound(o.Price, info.PriceFloat)
	size := mather.FloatRound(o.Size, info.SizeFloat)
	quantity := convert.GetString(math.Abs(size))
	service := client.NewCreateOrderService().Symbol(bsymbol).Quantity(quantity)
	orderType := binanceapi.OrderTypeMarket
	if price != 0 {
		orderType = binanceapi.OrderTypeLimit
		//todo tif no do
		service = service.TimeInForce(binanceapi.TimeInForceTypeGTC).Price(convert.GetString(price))
	} else if o.Price != 0 {
		return nil
	}
	if size > 0 {
		service.Type(orderType).Side(binanceapi.SideTypeBuy).NewOrderRespType(binanceapi.NewOrderRespTypeRESULT)
	} else if size < 0 {
		service.Type(orderType).Side(binanceapi.SideTypeSell).NewOrderRespType(binanceapi.NewOrderRespTypeRESULT)
	} else {
		log.Errorln(log.Http, bs.Api.ApiSign, o.Symbol, "BinaceSpotApi size is 0", size)
		return nil
	}
	if o.UUID != "" {
		service.NewClientOrderID(o.UUID)
	}
	return service
}

func (bs *BinaceSpotApi) CannelOrder(ctx context.Context) (*exch.Order, error) {
	o := exch.GetOrder(ctx)
	if o == nil {
		log.Errorln(log.Http, bs.Api.ApiSign, "BinaceSpotApi CannelOrder GetOrder id error")
		return nil, nil
	}
	orderId := convert.GetInt64(o.Id)
	bsymbol := unify.SymbolToB(o.Symbol)
	res, err := bs.Api.GetClient().NewCancelOrderService().Symbol(bsymbol).OrderID(orderId).Do(context.Background())
	if err != nil {
		return nil, err
	}
	size := convert.GetFloat64(res.OrigQuantity)
	lsize := size - convert.GetFloat64(res.ExecutedQuantity)
	lsize = math.Abs(lsize)
	if res.Side == binanceapi.SideTypeSell {
		size = -size
	}
	or := &exch.Order{
		Id:         convert.GetString(res.OrderID),
		UUID:       res.ClientOrderID,
		Symbol:     o.Symbol,
		Price:      o.Price,
		Size:       size,
		Left:       lsize,
		Status:     unify.SpotOrderStatus[res.Status],
		CreateTime: o.CreateTime,
		UpdateTime: o.UpdateTime,
	}
	return or, nil
}

func (bs *BinaceSpotApi) CannelAllOrder(ctx context.Context) ([]*exch.Order, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	bsymbol := unify.SymbolToB(symbol)
	//todo no result
	_, err := bs.Api.GetClient().NewCancelOpenOrdersService().Symbol(bsymbol).Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bs.Api.ApiSign, "BinaceSpotApi  CannelAllOrder error", err)
		return nil, err
	}
	//log.Infoln(log.Http, bf.Api.ApiSign, bsymbol, "BinaceSpotApi CannelAllOrder success ")
	return nil, nil
}

func (bs *BinaceSpotApi) GetOrder(ctx context.Context) (map[string]*exch.Order, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	bsymbol := unify.SymbolToB(symbol)
	res, err := bs.Api.GetClient().NewListOpenOrdersService().Symbol(bsymbol).Do(context.Background())
	if err != nil {
		return nil, err
	}
	orders := map[string]*exch.Order{}
	for _, o := range res {
		status := unify.SpotOrderStatus[o.Status]
		size := convert.GetFloat64(o.OrigQuantity)
		if o.Side == binanceapi.SideTypeSell {
			size = -size
		}
		left := size - convert.GetFloat64(o.ExecutedQuantity)
		or := exch.Order{
			Id:         convert.GetString(o.OrderID),
			UUID:       o.ClientOrderID,
			Price:      convert.GetFloat64(o.Price),
			Size:       size,
			Left:       left,
			Status:     status,
			CreateTime: o.Time,
			UpdateTime: o.UpdateTime,
		}
		orders[or.Id] = &or
	}
	return orders, nil
}

func (bs *BinaceSpotApi) GetOrderBook(ctx context.Context) (*binanceapi.DepthResponse, error) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	bsymbol := unify.SymbolToB(symbol)
	limit := text.GetInt64(ctx, OrderBookLimit)
	res, err := bs.Api.GetClient().NewDepthService().Symbol(bsymbol).Limit(int(limit)).Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bs.Api.ApiSign, symbol, "BinaceSpotApi GetOrderBook error", err)
	}
	return res, err
}

func (bs *BinaceSpotApi) GetPosition(ctx context.Context) (*exch.Position, error) {
	pos := &exch.Position{}
	symbol := text.GetString(ctx, exch.CtxSymbol)
	rd := redis.BookRedis()
	hkey := fmt.Sprintf(redis.SPOT_POSITION_HKEY, bs.Api.ApiSign, symbol)
	result, err := rd.Hget(redis.SPOT_POSITION_KEY, hkey)
	//no spot cache
	if err != nil && err.Error() == "redigo: nil returned" {
		return pos, nil
	}
	if err != nil {
		log.Warnln(log.Http, "BinaceSpotApi GetPosition Hget data error ", symbol, err)
		return pos, err
	}
	err = json.Unmarshal([]byte(result), pos)
	if err != nil {
		log.Errorln(log.Http, "BinaceSpotApi GetPosition Unmarshal pos data error ", symbol, result, err)
		return nil, err
	}
	return pos, err
}

func (bs *BinaceSpotApi) GetBalance(ctx context.Context) (map[string]*exch.Balance, error) {
	InitBalance.sc.Lock()
	defer InitBalance.sc.Unlock()
	ti := timer.MicNow()
	isinit := text.GetBool(ctx, exch.CtxInit)
	if isinit && ti-InitBalance.T < 5000 {
		return InitBalance.D, nil
	}
	res, err := bs.Api.GetClient().NewGetAccountService().Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bs.Api.ApiSign, "BinaceSpotApi GetBalance error", err)
		return nil, err
	}
	assets := map[string]*exch.Balance{}
	for _, r := range res.Balances {
		lock := convert.GetFloat64(r.Locked)
		free := convert.GetFloat64(r.Free)
		total := free + lock
		asset := &exch.Balance{
			ApiSign: bs.Api.ApiSign,
			Asset:   r.Asset,
			Total:   total,
			Avative: free,
		}
		assets[r.Asset] = asset
	}
	InitBalance.D = assets
	InitBalance.T = ti
	return assets, nil
}

func (bs *BinaceSpotApi) ListPosition(ctx context.Context) (map[string]*exch.Position, error) {
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
		if ks[0] != bs.Api.ApiSign {
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

func (bs *BinaceSpotApi) GetBaseInfo(symbol string) (*exch.BaseInfo, error) {
	InitInfo.sc.Lock()
	defer InitInfo.sc.Unlock()
	if res, ok := InitInfo.D.Get(symbol); ok {
		return res.(*exch.BaseInfo), nil
	}
	res, err := bs.Api.GetClient().NewExchangeInfoService().Do(context.Background())
	if err != nil {
		log.Errorln(log.Http, bs.Api.ApiSign, symbol, "BinaceSpotApi  GetBaseInfo error", err)
		return nil, err
	}
	//log.Warnln(log.Http, bf.Api.ApiSign, symbol, "BinaceSpotApi GetBaseInfo success ")
	ti := time.Now().Unix()
	infos := map[string]interface{}{}
	for _, r := range res.Symbols {
		info := &exch.BaseInfo{}
		s := unify.BToSymbol(r.Symbol)
		//todo ETHBTC
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
		info.LastUpdateTime = ti
		infos[s] = info
	}
	InitInfo.D.MSet(infos)
	if info, ok := InitInfo.D.Get(symbol); ok {
		return info.(*exch.BaseInfo), nil
	}
	return nil, errors.New("no symbol info")
}

func (bs *BinaceSpotApi) GetSignSymbol(symbol string) string {
	return bs.Api.ApiSign + "-" + symbol
}
