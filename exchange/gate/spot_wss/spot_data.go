package spot_wss

import (
	"context"
	"strings"
	"sync"

	cmap "github.com/orcaman/concurrent-map"

	"high-freq-quant-go/exchange/gate/unify"

	"high-freq-quant-go/adapter/timer"

	"high-freq-quant-go/adapter/convert"
	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/log"
	"high-freq-quant-go/exchange/gate/spot_api"
)

type SpotWss struct {
	//sign
	ApiSign, Sign string
	//api
	Api *spot_api.GateSpotApi
	//param
	Ctx context.Context

	//return data
	BaseData     cmap.ConcurrentMap
	Bookers      cmap.ConcurrentMap
	TradeData    map[string]*chan *exch.Order
	OrderData    map[string]map[string]*exch.Order
	PositionData cmap.ConcurrentMap
	BalanceData  cmap.ConcurrentMap
	//BalanceData  map[string]*exch.Balance

	//wss client
	Cl *SpotClient

	//struct msg chan
	OrderBookQueue *chan *DepthUpdateAllEvent
	UserDataQueue  *chan interface{}

	//struct msg to public
	BookDataChannel *chan interface{}
	BookMsgChan     *chan interface{}

	bal, bdl, bl, tdl, odl, pdl sync.RWMutex
}

func NewGateSpotWss(ctx context.Context) *SpotWss {
	apiSign := text.GetString(ctx, exch.ApiSign)
	bq := make(chan *DepthUpdateAllEvent, exch.MsgChannelLen)
	uq := make(chan interface{}, exch.MsgChannelLen)
	sg := text.GetString(ctx, exch.ConnSign)
	ft := &SpotWss{
		ApiSign: apiSign,
		Sign:    sg,
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
	cl, err := NewSpotClient(ctx)
	if err != nil {
		return nil
	}
	ft.Cl = cl
	go ft.MsgEvent()
	go ft.OrderBookEvent()
	go ft.UserDataMsgEvent()
	return ft
}

func (ws *SpotWss) GetStatus() int {
	return ws.Cl.Status
}

func (ws *SpotWss) GetBaseInfo(ctx context.Context) *exch.BaseInfo {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	info := &exch.BaseInfo{}
	if infoi, ok := ws.BaseData.Get(symbol); ok {
		info = infoi.(*exch.BaseInfo)
	} else {
		return nil
	}
	return info
}

func (ws *SpotWss) GetBook(ctx context.Context) *exch.Booker {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if res, ok := ws.Bookers.Get(symbol); ok {
		return res.(*exch.Booker)
	}
	return nil
}

func (ws *SpotWss) GetOrders(ctx context.Context) map[string]*exch.Order {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	ws.odl.RLock()
	defer ws.odl.RUnlock()
	if res, ok := ws.OrderData[symbol]; ok {
		return res
	}
	return map[string]*exch.Order{}
}

func (ws *SpotWss) GetPosition(ctx context.Context) *exch.Position {
	if ws.PositionData.Count() == 0 {
		return nil
	}
	symbol := text.GetString(ctx, exch.CtxSymbol)
	pos := &exch.Position{}
	if posi, ok := ws.PositionData.Get(symbol); ok {
		pos = posi.(*exch.Position)
	}
	return pos
}

func (ws *SpotWss) GetBalance(ctx context.Context) *exch.Balance {
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

func (ws *SpotWss) GetTradeChan(ctx context.Context) *chan *exch.Order {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	ws.tdl.RLock()
	defer ws.tdl.RUnlock()
	if res, ok := ws.TradeData[symbol]; ok {
		return res
	}
	return nil
}

func (ws *SpotWss) SetPubChannel() {
	if ret, ok := ws.Ctx.Value(exch.BookDataChannel).(*chan interface{}); ok {
		ws.BookDataChannel = ret
	}
	if ret, ok := ws.Ctx.Value(exch.BookMsgChan).(*chan interface{}); ok {
		ws.BookMsgChan = ret
	}
}

func (ws *SpotWss) SubTicker(ctx context.Context) error {
	err := ws.InitTickers(ctx)
	if err != nil {
		return err
	}
	err = ws.Cl.Tickers(ctx)
	return err
}

func (ws *SpotWss) InitTickers(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if ws.Api == nil {
		ws.Api = spot_api.NewGateSpotApi(ws.Ctx)
	}
	result, err := ws.Api.GetBaseInfo(symbol)
	if err != nil {
		log.Errorf(log.Wss, " %s %s gate wss InitTickers GetBaseInfo error %+v \r\n", ws.Sign, symbol, err)
		return err
	}
	log.Debugf(log.Wss, " %s gate wss InitTickers %+v \r\n", ws.Sign, result)
	ws.BaseData.Set(symbol, result)
	return err
}

func (ws *SpotWss) SubOrderBook(ctx context.Context) error {
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

func (ws *SpotWss) SubUserTrade(ctx context.Context) error {
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
	err := ws.Cl.UserTrades(ctx)
	return err
}

func (ws *SpotWss) SubPosition(ctx context.Context) error {
	err := ws.InitPosition(ctx)
	if err != nil {
		return err
	}
	pc := make(exch.PubChan)
	go ws.ReConnect(ws.InitPosition, ctx, &pc)
	ctx = context.WithValue(ctx, exch.CtxChan, &pc)
	err = ws.Cl.UserTrades(ctx)
	return err
}

func (ws *SpotWss) InitPosition(ctx context.Context) error {
	if ws.Api == nil {
		ws.Api = spot_api.NewGateSpotApi(ws.Ctx)
	}
	ictx := context.WithValue(ctx, exch.CtxInit, true)
	result, err := ws.Api.ListPosition(ictx)
	log.Infof(log.Wss, "%s gate spot wss InitPosition %+v \r\n", ws.Sign, result)
	if err != nil {
		ws.PositionData.Clear()
		return err
	}
	for k, v := range result {
		ws.PositionData.Set(k, v)
	}
	symbol := text.GetString(ctx, exch.CtxSymbol)
	pos := &exch.Position{}
	if posi, ok := ws.PositionData.Get(symbol); ok {
		pos = posi.(*exch.Position)
	} else {
		ws.PositionData.Set(symbol, pos)
	}
	log.Infof(log.Wss, "%s %s binance user wss InitPosition PositionData success %+v \r\n", ws.Sign, symbol, pos)
	return err
}

func (ws *SpotWss) SubBalance(ctx context.Context) error {
	err := ws.InitBalance(ctx)
	if err != nil {
		return err
	}
	err = ws.Cl.Balance(ctx)
	return err
}

func (ws *SpotWss) InitBalance(ctx context.Context) error {
	if ws.Api == nil {
		ws.Api = spot_api.NewGateSpotApi(ws.Ctx)
	}
	symbol := text.GetString(ctx, exch.CtxSymbol)
	ictx := context.WithValue(ctx, exch.CtxInit, true)
	result, err := ws.Api.GetBalance(ictx)
	if err != nil {
		ws.BalanceData.Clear()
		log.Errorln(log.Wss, ws.Sign, symbol, "gate user wss InitBalance GetBalance error ", err)
		return err
	}
	for k, v := range result {
		ws.BalanceData.Set(k, v)
	}
	base, quote := exch.GetBaseQuote(symbol)
	if _, ok := ws.BalanceData.Get(base); !ok {
		ws.BalanceData.Set(base, &exch.Balance{})
	}
	if _, ok := ws.BalanceData.Get(quote); !ok {
		ws.BalanceData.Set(quote, &exch.Balance{})
	}
	log.Infof(log.Wss, "%s gate wss InitBalance %+v \r\n", ws.Sign, result)
	return err
}

func (ws *SpotWss) SubOrder(ctx context.Context) error {
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

func (ws *SpotWss) InitOrders(ctx context.Context) error {
	if ws.Api == nil {
		ws.Api = spot_api.NewGateSpotApi(ws.Ctx)
	}
	result, err := ws.Api.GetOrder(ctx)
	log.Infof(log.Wss, " %s gate wss InitOrders %+v \r\n", ws.Sign, result)
	symbol := text.GetString(ctx, exch.CtxSymbol)
	ws.odl.Lock()
	defer ws.odl.Unlock()
	if err != nil {
		ws.OrderData[symbol] = map[string]*exch.Order{}
		return err
	}
	ws.OrderData[symbol] = result
	return err
}

func (ws *SpotWss) MsgEvent() {
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

func (ws *SpotWss) OrderBookEvent() {
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
			if book.UpdateID == 0 || msg.Result.FirstId > book.UpdateID+1 {
				log.Debugln(log.Wss, ws.Sign, symbol, "gate init Orderbook start FirstId LastId LastUpdateID ", msg.Result.FirstId, msg.Result.LastId, book.UpdateID)
				ws.InitOrderbook(symbol)
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
					size := convert.GetFloat64(bid[1])
					book.UpdateBid(bid[0], size)
				}
			}(msg.Result, book, &wg)
			go func(result DepthUpdateResult, book *exch.Booker, wg *sync.WaitGroup) {
				defer wg.Done()
				for _, ask := range result.Asks {
					size := convert.GetFloat64(ask[1])
					book.UpdateAsk(ask[0], size)
				}
			}(msg.Result, book, &wg)
			wg.Wait()
			book.UpdateID = msg.Result.LastId
			book.ResponTime = msg.Result.UpdateTimeMs
			book.UpdateTime = timer.MicNow()
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

func (ws *SpotWss) InitOrderbook(symbol string) {
	if ws.Api == nil {
		ws.Api = spot_api.NewGateSpotApi(ws.Ctx)
	}
	ctx := context.WithValue(context.Background(), exch.CtxSymbol, symbol)
	ctx = context.WithValue(ctx, spot_api.OrderBookLimit, OrderBookNum)
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
			ps := bid[0]
			bidMap[ps] = convert.GetFloat64(bid[1])
		}
	}(bidMap, &wg)
	go func(askMap map[string]float64, wg *sync.WaitGroup) {
		defer wg.Done()
		for _, ask := range result.Asks {
			ps := ask[0]
			askMap[ps] = convert.GetFloat64(ask[1])
		}
	}(askMap, &wg)
	wg.Wait()
	book.SetBook(askMap, bidMap)
	book.UpdateID = result.Id
	book.ResponTime = int64(result.Current * 1000)
	book.UpdateTime = timer.MicNow()
	ws.Bookers.Set(symbol, book)
	log.Debugf(log.Wss, "%s %s gate init Orderbook success [ResponTime=%d] [UpdateTime=%d][bookName=%s][lastid=%d] \r\n", ws.Sign, symbol, book.ResponTime, book.UpdateTime, book.Name, book.UpdateID)
}

func (ws *SpotWss) UserDataMsgEvent() {
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

func (ws *SpotWss) UpdateTickers(data *TickersEvent) {
	v := data.Result
	baseinfo := &exch.BaseInfo{}
	if info, ok := ws.BaseData.Get(v.Symbol); ok {
		baseinfo = info.(*exch.BaseInfo)
	}
	baseinfo.ChangeRate = v.ChangePercentage
	baseinfo.MarkPrice = v.Last
	baseinfo.LastUpdateTime = timer.MicNow()
	ws.BaseData.Set(v.Symbol, baseinfo)
	//log.Debugf(log.Wss, "gate wss UpdateTickers  %+v %d \r\n", baseinfo, ws.Cl.ClientId)
}

func (ws *SpotWss) UpdateUserTrade(data *UserTradeEvent) {
	for _, v := range data.Result {
		size := v.Amount
		if v.Side == unify.SideSell {
			size = -size
		}
		or := exch.Order{
			ApiSign:    ws.ApiSign,
			Id:         v.OrderId,
			TradeId:    convert.GetString(v.Id),
			Symbol:     v.Symbol,
			Price:      convert.GetFloat64(v.Price),
			Size:       size,
			Role:       v.Role,
			CreateTime: int64(v.CreateTimeMs),
			UpdateTime: int64(v.CreateTimeMs),
		}
		ws.tdl.Lock()
		if _, ok := ws.TradeData[v.Symbol]; !ok {
			oq := make(chan *exch.Order, exch.MsgChannelLen)
			ws.TradeData[v.Symbol] = &oq
		}
		*ws.TradeData[v.Symbol] <- &or
		ws.tdl.Unlock()

		log.Infof(log.Wss, " %s gate user wss  UpdateUserTrade ClientId %d result %+v \r\n", ws.Sign, ws.Cl.ClientId, or)
		pos := &exch.Position{}
		if posi, ok := ws.PositionData.Get(v.Symbol); ok {
			pos = posi.(*exch.Position)
		}
		p, s, pnl, _ := exch.SumPosAvgPrice(pos.Price, pos.Size, or.Price, or.Size)
		pos.Price = p
		pos.Size = s
		pos.Pnl = pos.Pnl + pnl
		pos.UnPnl = (or.Price - pos.Price) * pos.Size
		ws.PositionData.Set(v.Symbol, pos)
	}
}

func (ws *SpotWss) UpdateBalances(data *BalancesEvent) {
	//todo lock change no push
	ds := map[string]interface{}{}
	for _, res := range data.Result {
		d := &exch.Balance{
			ApiSign: ws.ApiSign,
			Asset:   res.Currency,
			Total:   res.Total,
			Avative: res.Available,
		}
		ds[res.Currency] = d
	}
	ws.BalanceData.MSet(ds)
}

func (ws *SpotWss) UpdateOrders(data *OrdersEvent) {
	for _, res := range data.Result {
		size := res.Amount
		left := res.Left
		if res.Side == unify.SideSell {
			size = -size
		}
		status := unify.SpotOrderMap[res.Event]
		o := exch.Order{
			ApiSign:    ws.ApiSign,
			Id:         convert.GetString(res.Id),
			UUID:       res.Text,
			Symbol:     res.Symbol,
			Status:     status,
			Size:       size,
			Price:      convert.GetFloat64(res.Price),
			Left:       left,
			CreateTime: res.CreateTimeMs,
			UpdateTime: res.UpdateTimeMs,
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

func (ws *SpotWss) ReConnect(call func(ctx context.Context) error, ctx context.Context, pc *exch.PubChan) {
	for {
		select {
		case <-ws.Ctx.Done():
			return
		case <-*pc:
			call(ctx)
		}
	}
}
