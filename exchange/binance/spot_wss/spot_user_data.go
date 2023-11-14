package spot_wss

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	cmap "github.com/orcaman/concurrent-map"
	"math"
	"sync"
	"time"

	"high-freq-quant-go/exchange/binance/unify"

	"high-freq-quant-go/adapter/convert"

	"high-freq-quant-go/exchange/binance/binanceapi"
	"high-freq-quant-go/exchange/binance/spot_api"

	"high-freq-quant-go/adapter/client"
	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/adapter/timer"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/log"
)

type UserWss struct {
	//sign
	ApiSign, Sign string
	//param
	Ctx context.Context

	//return data
	TradeData    map[string]*chan *exch.Order
	OrderData    map[string]map[string]*exch.Order
	PositionData cmap.ConcurrentMap
	BalanceData  cmap.ConcurrentMap

	//User
	UserStream *spot_api.UserStream
	//wss connect
	Wss *client.WssSocket

	//done
	CloseMsg context.Context
	//close
	CannelMsg context.CancelFunc

	//reconnect id
	ClientId int64
	//0断开1已连接2连接中
	Status      int
	ConnectTime int64

	//reconnect
	ReConnectMsg []client.SubscribeData

	lk, tdl, odl sync.RWMutex
	pdl, bdl     sync.Mutex
}

func NewFuturesUser(ctx context.Context) *UserWss {
	sign := text.GetString(ctx, exch.ConnSign)
	apiSign := text.GetString(ctx, exch.ApiSign)
	wss := &UserWss{
		ApiSign:      apiSign,
		Sign:         UserWssSign + sign,
		Ctx:          ctx,
		TradeData:    map[string]*chan *exch.Order{},
		OrderData:    map[string]map[string]*exch.Order{},
		PositionData: cmap.New(),
		BalanceData:  cmap.New(),
	}
	BinaceUserWss.lock.Lock()
	defer BinaceUserWss.lock.Unlock()
	if fu, ok := BinaceUserWss.Wss[apiSign]; ok {
		return fu
	}
	BinaceUserWss.Wss[apiSign] = wss
	return wss
}

func (ws *UserWss) GetStatus() int {
	return ws.Status
}

func (ws *UserWss) GetOrder(ctx context.Context) map[string]*exch.Order {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	ws.odl.RLock()
	defer ws.odl.RUnlock()
	if res, ok := ws.OrderData[symbol]; ok {
		return res
	}
	return nil
}

func (ws *UserWss) GetPosition(ctx context.Context) *exch.Position {
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

func (ws *UserWss) GetBalance(ctx context.Context) *exch.Balance {
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

func (ws *UserWss) GetTradeChan(ctx context.Context) *chan *exch.Order {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	ws.tdl.RLock()
	defer ws.tdl.RUnlock()
	if res, ok := ws.TradeData[symbol]; ok {
		return res
	}
	return nil
}

func (ws *UserWss) Create() error {
	ws.lk.Lock()
	defer ws.lk.Unlock()
	if ws.Wss != nil {
		return nil
	}
	if ws.UserStream == nil {
		ws.UserStream = spot_api.NewUserStream(ws.Ctx)
	}
	err := ws.CreateNewClient()
	return err
}

func (ws *UserWss) CreateNewClient() error {
	ListenKey := ws.UserStream.GetListenKey()
	if ListenKey == "" {
		return errors.New(fmt.Sprintf(" %s %d binance user wss ListenKey is Empty : %s ", ws.Sign, ws.ClientId, ListenKey))
	}
	err := ws.NewClient(ListenKey)
	return err
}

func (ws *UserWss) NewClient(ListenKey string) error {
	ctx := context.WithValue(context.Background(), client.WssUrl, UserWssUrl+ListenKey)
	ctx = context.WithValue(ctx, client.Keepalive, true)
	ctx = context.WithValue(ctx, client.Timeout, WssTimeout)
	ctx = context.WithValue(ctx, client.MsgLen, exch.WSChannelLen)
	ctx = context.WithValue(ctx, client.Id, ws.Sign)
	pUrl := text.GetString(ws.Ctx, client.ProxyUrl)
	ctx = context.WithValue(ctx, client.ProxyUrl, pUrl)

	ctx, cannel := context.WithCancel(ctx)
	wss, err := client.NewWssSocket(ctx, ws.ReConnect)
	if err != nil {
		log.Errorln(log.Wss, ws.Sign, ws.ClientId, "binance user wss  NewClient NewWssSocket error ", err)
		return err
	}
	if ws.CannelMsg != nil {
		ws.CannelMsg()
		time.Sleep(100 * time.Millisecond)
	}
	ws.Wss = wss
	ws.Status = 1
	ws.ConnectTime = timer.MicNow()
	ws.CloseMsg = ctx
	ws.CannelMsg = cannel
	go ws.ReceivedMsg()
	return nil
}

func (ws *UserWss) SubOrder(ctx context.Context) error {
	ws.odl.Lock()
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if _, ok := ws.TradeData[symbol]; ok {
		ws.odl.Unlock()
		return nil
	}
	ws.odl.Unlock()
	err := ws.InitOrder(ctx)
	if err != nil {
		return err
	}
	ws.RegisterMsg(ws.InitOrder, ctx)
	err = ws.Create()
	return err
}

func (ws *UserWss) SubUserTrade(ctx context.Context) error {
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
	err := ws.Create()
	return err
}

func (ws *UserWss) SubBalance(ctx context.Context) error {
	ws.bdl.Lock()
	defer ws.bdl.Unlock()
	symbol := text.GetString(ctx, exch.CtxSymbol)
	ctx = context.WithValue(ctx, exch.CtxInit, true)
	if ws.BalanceData.Count() != 0 {
		base, quote := exch.GetBaseQuote(symbol)
		if _, ok := ws.BalanceData.Get(base); !ok {
			ws.BalanceData.Set(base, &exch.Balance{})
		}
		if _, ok := ws.BalanceData.Get(quote); !ok {
			ws.BalanceData.Set(quote, &exch.Balance{})
		}
		return nil
	}
	err := ws.InitBalance(ctx)
	if err != nil {
		return err
	}
	ws.RegisterMsg(ws.InitBalance, ctx)
	err = ws.Create()
	return err
}

func (ws *UserWss) SubPosition(ctx context.Context) error {
	ws.pdl.Lock()
	defer ws.pdl.Unlock()
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if ws.PositionData.Count() != 0 {
		if _, ok := ws.PositionData.Get(symbol); !ok {
			ws.PositionData.Set(symbol, &exch.Position{})
		}
		return nil
	}
	err := ws.InitPosition(ctx)
	if err != nil {
		log.Errorln(log.Wss, ws.Sign, symbol, "binance user wss SubPosition error ", err)
		return err
	}
	ws.RegisterMsg(ws.InitPosition, ctx)
	err = ws.Create()
	return err
}

func (ws *UserWss) InitOrder(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	res, err := spot_api.NewBinanceApi(ws.Ctx).GetOrder(ctx)
	if err != nil {
		log.Errorln(log.Wss, ws.Sign, "binance user wss InitOrder error ", err)
		return err
	}
	ws.odl.Lock()
	ws.OrderData[symbol] = res
	ws.odl.Unlock()
	log.Infof(log.Wss, "%s %s binance user wss InitOrder %+v \r\n", ws.Sign, symbol, ws.OrderData[symbol])
	return nil
}

func (ws *UserWss) InitPosition(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	ictx := context.WithValue(ctx, exch.CtxInit, true)
	res, err := spot_api.NewBinanceApi(ws.Ctx).ListPosition(ictx)
	if err != nil {
		ws.PositionData.Clear()
		log.Errorln(log.Wss, ws.Sign, symbol, "binance user wss InitPosition error ", err)
		return err
	}
	for k, v := range res {
		ws.PositionData.Set(k, v)
	}
	pos := &exch.Position{}
	if _, ok := ws.PositionData.Get(symbol); !ok {
		ws.PositionData.Set(symbol, pos)
	}
	log.Infof(log.Wss, "%s %s binance user wss InitPosition PositionData success %+v \r\n", ws.Sign, symbol, res)
	return nil
}

func (ws *UserWss) InitBalance(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	result, err := spot_api.NewBinanceApi(ws.Ctx).GetBalance(ctx)
	if err != nil {
		ws.BalanceData.Clear()
		log.Errorln(log.Wss, ws.Sign, symbol, "binance user wss InitPosition GetBalance error ", err)
		return err
	}
	for k, v := range result {
		ws.BalanceData.Set(k, v)
	}
	base, quote := exch.GetBaseQuote(symbol)
	if _, ok := ws.BalanceData.Get(base); !ok {
		ws.BalanceData.Set(base, &exch.Position{})
	}
	if _, ok := ws.BalanceData.Get(quote); !ok {
		ws.BalanceData.Set(quote, &exch.Position{})
	}
	log.Infof(log.Wss, "%s %s binance user wss InitPosition BalanceData success %+v \r\n", ws.Sign, symbol, result)
	return nil
}

func (ws *UserWss) ReceivedMsg() {
	tr := time.NewTicker(30 * time.Minute)
	for {
		select {
		case <-ws.CloseMsg.Done():
			log.Warnln(log.Wss, ws.Sign, "binance user wss FuturesClient ReceivedMsg return by close")
			return
		case <-tr.C:
			log.Infoln(log.Wss, ws.Sign, "binance user wss UpdateListenKey")
			ws.UserStream.UpdateListenKey()
		case msg := <-*ws.Wss.MsgQueue:
			if msg != nil {
				ws.ReadUserMessage(msg)
			}
		}
	}
}

func (ws *UserWss) ReadUserMessage(message *[]byte) {
	var event binanceapi.WsUserDataEvent
	err := json.Unmarshal(*message, &event)
	if err != nil {
		log.Warnln(log.Wss, ws.Sign, " binance user wss json error", string(*message), err)
		return
	}
	switch event.Event {
	case AccountUpdate:
		ws.AccountUpdate(event)
	case OrderTradeUpdate:
		err = json.Unmarshal(*message, &event.OrderUpdate)
		if err != nil {
			log.Warnln(log.Wss, ws.Sign, " binance user wss Order json error", string(*message), err)
			return
		}
		ws.OrderTradeUpdate(event)
	case ListenKeyExpired:
		log.Warnln(log.Wss, ws.Sign, "------binance user wss ListenKeyExpired------")
		ws.Status = 0
		ws.UserStream.ResetListenKey()
		ws.ReConnect(nil)
		return
	default:
		log.Errorln(log.Wss, ws.Sign, "binance user wss Unknown type", string(*message))
		return
	}
}

func (ws *UserWss) AccountUpdate(data binanceapi.WsUserDataEvent) {
	for _, d := range data.AccountUpdate {
		lock := convert.GetFloat64(d.Locked)
		free := convert.GetFloat64(d.Free)
		total := free + lock
		bl := &exch.Balance{
			ApiSign: ws.ApiSign,
			Asset:   d.Asset,
			Total:   total,
			Avative: free,
		}
		ws.BalanceData.Set(d.Asset, bl)
		log.Infof(log.Wss, " %s binance user wss AccountUpdate Balances %s %+v \r\n", ws.Sign, d.Asset, bl)
	}
}

func (ws *UserWss) OrderTradeUpdate(data binanceapi.WsUserDataEvent) {
	o := data.OrderUpdate
	symbol := unify.BToSymbol(o.Symbol)
	status := unify.SpotOrderStatus[binanceapi.OrderStatusType(o.Status)]
	size := convert.GetFloat64(o.Volume)
	tsize := convert.GetFloat64(o.LatestVolume)
	lsize := size - convert.GetFloat64(o.FilledVolume)
	lsize = math.Abs(lsize)
	if o.Side == string(binanceapi.SideTypeSell) {
		size = -size
		tsize = -tsize
	}
	price := convert.GetFloat64(o.Price)
	fprice := convert.GetFloat64(o.LatestPrice)
	role := exch.OrderTaker
	if o.IsMaker {
		role = exch.OrderMaker
	}
	or := exch.Order{
		Id:         convert.GetString(o.Id),
		UUID:       o.ClientOrderId,
		Symbol:     symbol,
		Status:     status,
		Size:       size,
		Price:      price,
		FillPrice:  fprice,
		Left:       lsize,
		Role:       role,
		CreateTime: o.CreateTime,
		UpdateTime: o.TransactionTime,
	}
	ws.odl.Lock()
	if _, ok := ws.OrderData[symbol]; !ok {
		ws.OrderData[symbol] = map[string]*exch.Order{}
	}
	ws.OrderData[symbol][or.Id] = &or
	if or.Status == exch.OrderFinished {
		delete(ws.OrderData[symbol], or.Id)
	}
	ws.odl.Unlock()
	log.Debugf(log.Wss, "%s $s binance user wss  OrderTradeUpdate result %+v \r\n", ws.Sign, symbol, ws.OrderData)

	//只要成交 不要下单
	if o.ExecutionType == "TRADE" {
		log.Infof(log.Global, "%s %s binance user wss OrderTradeUpdate old order %+v \r\n", ws.Sign, symbol, o)
		tprice := convert.GetFloat64(o.LatestPrice)
		to := exch.Order{
			Id:         convert.GetString(o.Id),
			UUID:       o.ClientOrderId,
			Symbol:     symbol,
			Status:     status,
			Size:       tsize,
			Price:      tprice,
			Role:       role,
			CreateTime: o.CreateTime,
			UpdateTime: o.TransactionTime,
		}
		ws.tdl.Lock()
		if _, ok := ws.TradeData[symbol]; ok {
			*ws.TradeData[symbol] <- &to
		}
		ws.tdl.Unlock()

		pos := &exch.Position{}
		if posi, ok := ws.PositionData.Get(symbol); ok {
			pos = posi.(*exch.Position)
		}
		log.Infof(log.Wss, "%s %s binance user wss InitPosition PositionData success %+v \r\n", ws.Sign, symbol, pos)
		p, s, pnl, _ := exch.SumPosAvgPrice(pos.Price, pos.Size, or.Price, or.Size)
		pos.Price = p
		pos.Size = s
		pos.Pnl = pos.Pnl + pnl
		pos.UnPnl = (or.Price - pos.Price) * pos.Size
		ws.PositionData.Set(symbol, pos)
		log.Infof(log.Global, "%s %s binance user wss OrderTradeUpdate push order %+v \r\n", ws.Sign, symbol, or)
	}
}

func (ws *UserWss) RegisterMsg(action func(ctx context.Context) error, param context.Context) {
	if text.GetBool(param, client.IsReconnect) {
		return
	}
	param = context.WithValue(param, client.IsReconnect, true)
	method := client.SubscribeData{
		Action: &action,
		Param:  param,
	}
	ws.ReConnectMsg = append(ws.ReConnectMsg, method)
	return
}

func (ws *UserWss) ReConnect(err error) error {
	if ws.Status == 2 {
		return nil
	}
	ws.Status = 2
	conErr := ws.CreateNewClient()
	if conErr != nil {
		ws.Status = 0
		return conErr
	}
	log.Warnln(log.Wss, ws.Sign, "binance user wss start reconnect")
	for _, m := range ws.ReConnectMsg {
		fun := *m.Action
		ctx := m.Param
		nerr := fun(ctx)
		if nerr != nil {
			ws.Status = 0
			ws.Wss.Connect.Close()
			log.Errorln(log.Wss, ws.Sign, "binance user wss reconnect false")
			return nerr
		}
	}
	time.Sleep(3 * time.Second)
	ws.ClientId++
	ws.Status = 1
	ws.ConnectTime = timer.MicNow()
	return nil
}
