package futures_wss

import (
	"context"
	"strings"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map"

	"high-freq-quant-go/adapter/convert"
	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/log"
	"high-freq-quant-go/exchange/gate/futures_api"
	"high-freq-quant-go/exchange/gate/unify"
)

type Futures struct {
	//sign
	ApiSign, Sign string
	//api
	Api *futures_api.GateFuturesApi
	//param
	Ctx context.Context

	Units map[string]float64
	//return data
	BaseData     cmap.ConcurrentMap
	Bookers      cmap.ConcurrentMap
	TradeData    map[string]*chan *exch.Order
	OrderData    map[string]map[string]*exch.Order
	PositionData cmap.ConcurrentMap
	BalanceData  cmap.ConcurrentMap

	//wss client
	Cl *FuturesClient

	//struct msg chan
	OrderBookQueue *chan *DepthUpdateAllEvent
	UserDataQueue  *chan interface{}

	//struct msg to public
	BookDataChannel *chan interface{}
	BookMsgChan     *chan interface{}

	ul, bal, bdl, bl, tdl, odl, pdl sync.RWMutex
}

func NewGateFuturesWss(ctx context.Context) *Futures {
	apiSign := text.GetString(ctx, exch.ApiSign)
	bq := make(chan *DepthUpdateAllEvent, exch.MsgChannelLen)
	uq := make(chan interface{}, exch.MsgChannelLen)
	sg := text.GetString(ctx, exch.ConnSign)
	ft := &Futures{
		ApiSign: apiSign,
		Sign:    sg,
		Units:   map[string]float64{},
		Ctx:     ctx,

		BaseData:     cmap.New(),
		Bookers:      cmap.New(),
		TradeData:    map[string]*chan *exch.Order{},
		OrderData:    map[string]map[string]*exch.Order{},
		PositionData: cmap.New(),
		BalanceData:  cmap.New(),

		OrderBookQueue: &bq,
		UserDataQueue:  &uq,
	}
	ft.SetPubChannel()
	cl, err := NewFuturesClient(ctx)
	if err != nil {
		return nil
	}
	ft.Cl = cl
	go ft.MsgEvent()
	go ft.OrderBookEvent()
	go ft.UserDataMsgEvent()
	return ft
}

func (ws *Futures) GetStatus() int {
	return ws.Cl.Status
}

func (ws *Futures) GetBaseInfo(ctx context.Context) *exch.BaseInfo {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if res, ok := ws.BaseData.Get(symbol); ok {
		return res.(*exch.BaseInfo)
	}
	return nil
}

func (ws *Futures) GetBook(ctx context.Context) *exch.Booker {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if res, ok := ws.Bookers.Get(symbol); ok {
		return res.(*exch.Booker)
	}
	return nil
}

func (ws *Futures) GetOrders(ctx context.Context) map[string]*exch.Order {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	ws.odl.RLock()
	defer ws.odl.RUnlock()
	if res, ok := ws.OrderData[symbol]; ok {
		return res
	}
	return map[string]*exch.Order{}
}

func (ws *Futures) GetPosition(ctx context.Context) *exch.Position {
	if ws.PositionData.Count() == 0 {
		return nil
	}
	symbol := text.GetString(ctx, exch.CtxSymbol)
	pos := &exch.Position{}
	if res, ok := ws.PositionData.Get(symbol); ok {
		pos = res.(*exch.Position)
	}
	return pos
}

func (ws *Futures) GetBalance(ctx context.Context) *exch.Balance {
	if ws.BalanceData.Count() == 0 {
		return nil
	}
	base := text.GetString(ctx, exch.CtxBase)
	bl := &exch.Balance{}
	if bli, ok := ws.BalanceData.Get(base); ok {
		bl = bli.(*exch.Balance)
	}
	return bl
}

func (ws *Futures) GetTradeChan(ctx context.Context) *chan *exch.Order {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	ws.tdl.RLock()
	defer ws.tdl.RUnlock()
	if res, ok := ws.TradeData[symbol]; ok {
		return res
	}
	return nil
}

func (ws *Futures) SetPubChannel() {
	if ret, ok := ws.Ctx.Value(exch.BookDataChannel).(*chan interface{}); ok {
		ws.BookDataChannel = ret
	}
	if ret, ok := ws.Ctx.Value(exch.BookMsgChan).(*chan interface{}); ok {
		ws.BookMsgChan = ret
	}
}

func (ws *Futures) SetUnit(ctx context.Context) {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	ws.ul.Lock()
	defer ws.ul.Unlock()
	if u, ok := ws.Units[symbol]; ok && u != 0 {
		return
	}
	if ws.Api == nil {
		ws.Api = futures_api.NewGateFuturesApi(ws.Ctx)
	}
	unit := ws.Api.GetUnit(symbol)
	ws.Units[symbol] = unit
}

func (ws *Futures) SubTicker(ctx context.Context) error {
	err := ws.InitTickers(ctx)
	if err != nil {
		return err
	}
	err = ws.Cl.Tickers(ctx)
	return err
}

func (ws *Futures) InitTickers(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if _, ok := ws.BaseData.Get(symbol); ok {
		return nil
	}
	if ws.Api == nil {
		ws.Api = futures_api.NewGateFuturesApi(ws.Ctx)
	}
	result, err := ws.Api.GetBaseInfo(symbol)
	if err != nil {
		return err
	}
	log.Infof(log.Wss, " %s gate wss InitTickers %+v \r\n", ws.Sign, result)
	ws.BaseData.Set(symbol, result)
	return err
}

func (ws *Futures) SubOrderBook(ctx context.Context) error {
	ws.SetUnit(ctx)
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if symbol == "" {
		return nil
	}
	if _, ok := ws.Bookers.Get(symbol); ok {
		return nil
	}
	books := exch.NewBooker(ws.Ctx)
	ws.Bookers.Set(symbol, books)
	err := ws.Cl.OrderBook(ctx)
	return err
}

func (ws *Futures) SubUserTrade(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if symbol == "" {
		return nil
	}
	ws.tdl.Lock()
	if _, ok := ws.TradeData[symbol]; !ok {
		oq := make(chan *exch.Order, exch.MsgChannelLen)
		ws.TradeData[symbol] = &oq
	}
	ws.tdl.Unlock()
	ws.SetUnit(ctx)
	err := ws.Cl.UserTrades(ctx)
	return err
}

func (ws *Futures) SubPosition(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if _, ok := ws.PositionData.Get(symbol); ok {
		return nil
	}
	ws.SetUnit(ctx)
	err := ws.InitPosition(ctx)
	if err != nil {
		return err
	}
	pc := make(exch.PubChan)
	go ws.ReConnect(ws.InitPosition, ctx, &pc)
	ctx = context.WithValue(ctx, exch.CtxChan, &pc)
	err = ws.Cl.Position(ctx)
	return err
}

func (ws *Futures) InitPosition(ctx context.Context) error {
	if ws.Api == nil {
		ws.Api = futures_api.NewGateFuturesApi(ws.Ctx)
	}
	symbol := text.GetString(ctx, exch.CtxSymbol)
	result, err := ws.Api.GetPosition(ctx)
	if err != nil {
		ws.PositionData.Remove(symbol)
		return err
	}
	log.Infof(log.Wss, "%s gate wss InitPosition %+v \r\n", ws.Sign, result)
	ws.PositionData.Set(symbol, result)
	return err
}

func (ws *Futures) SubBalance(ctx context.Context) error {
	ws.SetUnit(ctx)
	err := ws.InitBalance(ctx)
	if err != nil {
		return err
	}
	err = ws.Cl.Balance(ctx)
	return err
}

func (ws *Futures) InitBalance(ctx context.Context) error {
	if ws.Api == nil {
		ws.Api = futures_api.NewGateFuturesApi(ws.Ctx)
	}
	ctx = context.WithValue(ctx, exch.CtxInit, true)
	result, err := ws.Api.GetBalance(ctx)
	if err != nil {
		ws.BalanceData.Clear()
		return err
	}
	log.Infof(log.Wss, "%s gate wss InitBalance %+v \r\n", ws.Sign, result)
	for k, v := range result {
		ws.BalanceData.Set(k, v)
	}
	return err
}

func (ws *Futures) SubOrder(ctx context.Context) error {
	ws.SetUnit(ctx)
	err := ws.InitOrders(ctx)
	if err != nil {
		return err
	}
	pc := make(exch.PubChan)
	go ws.ReConnect(ws.InitOrders, ctx, &pc)
	ctx = context.WithValue(ctx, exch.CtxChan, &pc)
	err = ws.Cl.Order(ctx)
	return err
}

func (ws *Futures) InitOrders(ctx context.Context) error {
	if ws.Api == nil {
		ws.Api = futures_api.NewGateFuturesApi(ws.Ctx)
	}
	result, err := ws.Api.GetOrder(ctx)
	if err != nil {
		return err
	}
	log.Infof(log.Wss, " %s gate wss InitOrders %+v \r\n", ws.Sign, result)
	symbol := text.GetString(ctx, exch.CtxSymbol)
	ws.odl.Lock()
	defer ws.odl.Unlock()
	ws.OrderData[symbol] = result
	return err
}

func (ws *Futures) MsgEvent() {
	for {
		select {
		case <-ws.Ctx.Done():
			log.Warnln(log.Wss, ws.Sign, " gatewss GateFutures MsgEvent return by done")
			return
		case msg := <-*ws.Cl.MsgQueue:
			switch msg.(type) {
			case *DepthUpdateAllEvent: // handle order book update
				//m.DepthUpdateEventHandler(rtype)
				*ws.OrderBookQueue <- msg.(*DepthUpdateAllEvent)
			default:
				*ws.UserDataQueue <- msg
			}
		}
	}
}

func (ws *Futures) OrderBookEvent() {
	for {
		select {
		case <-ws.Ctx.Done():
			log.Warnln(log.Wss, ws.Sign, " GateFutures OrderBookEvent return by done")
			return
		case msg := <-*ws.OrderBookQueue:
			//st := time.Now().UnixNano() / 1000000
			symbol := strings.ToUpper(msg.Result.Symbol)
			var book *exch.Booker
			if booki, ok := ws.Bookers.Get(symbol); !ok {
				log.Warnln(log.Wss, ws.Sign, symbol, "no symbolbook")
				continue
			} else {
				book = booki.(*exch.Booker)
			}
			if u, ok := ws.Units[symbol]; !ok || u == 0 {
				log.Warnln(log.Wss, ws.Sign, symbol, "symbol ws.Units is error", u)
				continue
			}
			unit := ws.Units[symbol]
			if book.UpdateID == 0 || msg.Result.FirstId > book.UpdateID+1 {
				log.Debugf(log.Wss, ws.Sign, symbol, "gate init Orderbook start FirstId LastId LastUpdateID ", msg.Result.FirstId, msg.Result.LastId, book.UpdateID)
				ws.InitOrderbook(symbol, unit)
				if booki, ok := ws.Bookers.Get(symbol); !ok {
					continue
				} else {
					book = booki.(*exch.Booker)
				}
				if ws.BookMsgChan != nil {
					*ws.BookMsgChan <- book
				}
			}
			if book.GetAskLen() == 0 || book.GetBidLen() == 0 {
				log.Warnln(log.Wss, ws.Sign, symbol, "gate init Orderbook error asks bids:", book.GetAskLen(), book.GetBidLen())
				continue
			}
			if msg.Result.LastId < book.UpdateID {
				continue
			}
			book.Rw.Lock()
			if !book.IsReady {
				log.Infoln(log.Wss, ws.Sign, symbol, "gate orderBook msg IsReading  FirstId LastId LastUpdateID ", msg.Result.FirstId, msg.Result.LastId, book.UpdateID)
				book.IsReady = true
			}
			var wg sync.WaitGroup
			wg.Add(2)
			//todo Quantity
			go func(result DepthUpdateResult, book *exch.Booker, wg *sync.WaitGroup) {
				defer wg.Done()
				for _, bid := range result.Bids {
					quantity := unify.UnitSize(bid.Quantity, unit)
					book.UpdateBid(bid.Price, quantity)
				}
			}(msg.Result, book, &wg)
			go func(result DepthUpdateResult, book *exch.Booker, wg *sync.WaitGroup) {
				defer wg.Done()
				for _, ask := range result.Asks {
					quantity := unify.UnitSize(ask.Quantity, unit)
					book.UpdateAsk(ask.Price, quantity)
				}
			}(msg.Result, book, &wg)
			wg.Wait()
			book.UpdateID = msg.Result.LastId
			book.ResponTime = msg.Result.Time
			book.UpdateTime = time.Now().UnixNano() / 1000000
			book.Rw.Unlock()
			if ws.BookMsgChan != nil {
				*ws.BookMsgChan <- msg
			}
			if ws.BookDataChannel != nil {
				*ws.BookDataChannel <- book
			}
		}
	}
}

func (ws *Futures) InitOrderbook(symbol string, unit float64) {
	if ws.Api == nil {
		ws.Api = futures_api.NewGateFuturesApi(ws.Ctx)
	}
	ctx := context.WithValue(context.Background(), exch.CtxSymbol, symbol)
	ctx = context.WithValue(ctx, futures_api.OrderBookLimit, OrderBookNum)
	result, _ := ws.Api.GetOrderBook(ctx)
	if result == nil {
		return
	}
	book := exch.NewBooker(ws.Ctx)
	book.Symbol = symbol
	book.IsReady = false
	askMap, bidMap := map[string]float64{}, map[string]float64{}
	var wg sync.WaitGroup
	wg.Add(2)
	go func(bidMap map[string]float64, wg *sync.WaitGroup) {
		defer wg.Done()
		for _, bid := range result.Bids {
			if bid.S == 0 {
				continue
			}
			bidMap[bid.P] = unify.UnitSize(float64(bid.S), unit)
		}
	}(bidMap, &wg)
	go func(askMap map[string]float64, wg *sync.WaitGroup) {
		defer wg.Done()
		for _, ask := range result.Asks {
			if ask.S == 0 {
				continue
			}
			askMap[ask.P] = unify.UnitSize(float64(ask.S), unit)
		}
	}(askMap, &wg)
	wg.Wait()
	book.SetBook(askMap, bidMap)
	book.UpdateID = result.Id
	book.ResponTime = int64(result.Current * 1000)
	book.UpdateTime = time.Now().UnixNano() / 1000000
	ws.Bookers.Set(symbol, book)
	log.Debugf(log.Wss, "%s %s gate init Orderbook success [ResponTime=%d] [UpdateTime=%d][bookName=%s][lastid=%d] \r\n", ws.Sign, symbol, book.ResponTime, book.UpdateTime, book.Name, book.UpdateID)
}

func (ws *Futures) UserDataMsgEvent() {
	for {
		select {
		case <-ws.Ctx.Done():
			log.Warnln(log.Wss, ws.Sign, " GateFutures UserTradeMsgEvent return by done")
			return
		case msg := <-*ws.UserDataQueue:
			switch msg.(type) {
			case *UserTradeEvent:
				ws.UpdateUserTrade(msg.(*UserTradeEvent))
			case *TickersEvent:
				ws.UpdateTickers(msg.(*TickersEvent))
			case *PositionsEvent:
				ws.UpdatePositions(msg.(*PositionsEvent))
			case *BalancesEvent:
				ws.UpdateBalances(msg.(*BalancesEvent))
			case *OrdersEvent:
				ws.UpdateOrders(msg.(*OrdersEvent))
			default:
				log.Errorln(log.Wss, ws.Sign, "gate unknown msg type", msg)
			}
		}
	}
}

func (ws *Futures) UpdateTickers(data *TickersEvent) {
	for _, v := range data.Result {
		baseinfo := &exch.BaseInfo{}
		if info, ok := ws.BaseData.Get(v.Symbol); ok {
			baseinfo = info.(*exch.BaseInfo)
		}
		baseinfo.MarkPrice = v.MarkPrice
		baseinfo.IndexPrice = v.IndexPrice
		baseinfo.FundingRate = v.FundingRate
		baseinfo.ChangeRate = v.ChangePercentage
		baseinfo.DayVolume = v.Volume24hSettle
		baseinfo.LastUpdateTime = data.Time
		ws.BaseData.Set(v.Symbol, baseinfo)
		if posi, ok := ws.PositionData.Get(v.Symbol); ok {
			pos := posi.(*exch.Position)
			size := pos.Size
			pos.MarkPrice = v.MarkPrice
			pos.Value = v.MarkPrice * size
		}
		//log.Debugf(log.Wss, "gate wss UpdateTickers  %+v %d \r\n", baseinfo, ws.Cl.ClientId)
	}

}

func (ws *Futures) UpdateUserTrade(data *UserTradeEvent) {
	for _, v := range data.Result {
		unit := ws.Units[v.Symbol]
		size := unify.UnitSize(float64(v.Size), unit)
		or := exch.Order{
			ApiSign:    ws.ApiSign,
			Id:         v.OrderId,
			TradeId:    v.Id,
			Symbol:     v.Symbol,
			Price:      convert.GetFloat64(v.Price),
			Size:       size,
			Role:       v.Role,
			CreateTime: v.CreateTime,
		}
		ws.tdl.Lock()
		if _, ok := ws.TradeData[v.Symbol]; !ok {
			oq := make(chan *exch.Order, exch.MsgChannelLen)
			ws.TradeData[v.Symbol] = &oq
		}
		*ws.TradeData[v.Symbol] <- &or
		ws.tdl.Unlock()
		log.Infof(log.Wss, " %s gate user wss  UpdateUserTrade ClientId %d result %+v \r\n", ws.Sign, ws.Cl.ClientId, or)
	}
}

func (ws *Futures) UpdatePositions(data *PositionsEvent) {
	for _, res := range data.Result {
		unit := ws.Units[res.Symbol]
		size := unify.UnitSize(float64(res.Size), unit)
		mtype := exch.MarginIsolated
		if res.Leverage == 0 {
			mtype = exch.MarginCrossed
		}
		pos := &exch.Position{
			Symbol:         res.Symbol,
			Price:          convert.GetFloat64(res.EntryPrice),
			Size:           size,
			Margin:         convert.GetFloat64(res.Margin),
			UnPnl:          convert.GetFloat64(res.RealisedPnl),
			LiqPrice:       convert.GetFloat64(res.LiqPrice),
			Lv:             convert.GetFloat64(res.Leverage),
			MarginType:     mtype,
			PositionMode:   unify.PositionMap[res.Mode],
			LastUpdateTime: res.TimeMs,
		}
		ws.PositionData.Set(res.Symbol, pos)
		log.Infof(log.Wss, " %s %s gate user wss  UpdatePositions result %+v \r\n", ws.Sign, res.Symbol, pos)
	}
}

func (ws *Futures) UpdateBalances(data *BalancesEvent) {
	bls := map[string]interface{}{}
	for _, res := range data.Result {
		bls[futures_api.AssetUsdt] = &exch.Balance{
			ApiSign: ws.ApiSign,
			Asset:   futures_api.AssetUsdt,
			Avative: res.Balance,
			Total:   res.Balance,
		}
	}
	ws.BalanceData.MSet(bls)
}

func (ws *Futures) UpdateOrders(data *OrdersEvent) {
	for _, res := range data.Result {
		unit := ws.Units[res.Symbol]
		size := unify.UnitSize(float64(res.Size), unit)
		left := unify.UnitSize(float64(res.Left), unit)
		o := exch.Order{
			Id:         convert.GetString(res.Id),
			UUID:       res.Text,
			Symbol:     res.Symbol,
			Status:     res.Status,
			Size:       size,
			Price:      convert.GetFloat64(res.Price),
			FillPrice:  convert.GetFloat64(res.FillPrice),
			Left:       left,
			Iceberg:    res.Iceberg,
			Tif:        res.Tif,
			CreateTime: res.CreateTimeMs,
			UpdateTime: res.FinishTimeMs,
		}
		ws.odl.Lock()
		if _, ok := ws.OrderData[res.Symbol]; !ok {
			ws.OrderData[res.Symbol] = map[string]*exch.Order{}
		}
		ws.OrderData[res.Symbol][o.Id] = &o
		if o.Status == exch.OrderFinished {
			delete(ws.OrderData[res.Symbol], o.Id)
		}
		ws.odl.Unlock()
		log.Debugf(log.Wss, " %s gate user wss  UpdateOrders result %+v \r\n", ws.Sign, ws.OrderData)
	}
}

func (ws *Futures) ReConnect(call func(ctx context.Context) error, ctx context.Context, pc *exch.PubChan) {
	for {
		select {
		case <-ws.Ctx.Done():
			return
		case <-*pc:
			call(ctx)
		}
	}
}
