package futures_wss

import (
	"context"
	"encoding/json"
	"time"

	"high-freq-quant-go/adapter/client"
	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/adapter/timer"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/log"
)

type FuturesClient struct {
	//sign
	Sign string
	//param
	Ctx context.Context

	//wss connect
	Wss *client.WssSocket
	//all msg chan
	MsgQueue *exch.MsgQueue

	//reconnect id
	ClientId int64
	//0断开1已连接2连接中
	Status int
	//连接时间
	ConnectTime int64

	//reconnect
	ReConnectMsg []client.SubscribeData
}

func NewFuturesClient(ctx context.Context) (*FuturesClient, error) {
	q := make(exch.MsgQueue, exch.ClientChannelLen)
	sg := text.GetString(ctx, exch.ConnSign)
	cl := &FuturesClient{
		Sign:     sg,
		Ctx:      ctx,
		MsgQueue: &q,
		ClientId: 0,
	}
	err := cl.NewClient()
	if err != nil {
		return nil, err
	}
	return cl, nil
}
func (ws *FuturesClient) NewClient() error {
	ctx := context.WithValue(ws.Ctx, client.WssUrl, UsdtWssUrl)
	ctx = context.WithValue(ctx, client.Keepalive, true)
	ctx = context.WithValue(ctx, client.Timeout, WssTimeout)
	ctx = context.WithValue(ctx, client.MsgLen, exch.WSChannelLen)
	ctx = context.WithValue(ctx, client.Id, ws.Sign)
	wss, err := client.NewWssSocket(ctx, ws.ReConnect)
	if err != nil {
		log.Errorln(log.Wss, ws.Sign, "gate wss FuturesClient error ", err)
		return err
	}
	ws.Wss = wss
	ws.Status = 1
	ws.ConnectTime = timer.MicNow()
	go ws.ReceivedMsg()
	return nil
}

func (ws *FuturesClient) Tickers(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	params := []string{symbol}
	msg, err := ws.SubscribeChannel(ChannelTickers, params)
	if err != nil {
		return err
	}
	ws.RegisterMsg(ws.Tickers, ctx)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	err = ws.Wss.SendMsg(ctx)
	return err
}

func (ws *FuturesClient) OrderBook(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	frequency := "100ms"  // 100ms, 1000ms
	level := OrderBookNum // 20,10,5
	params := []string{symbol, frequency, level}
	msg, err := ws.SubscribeChannel(ChannelDepthUpdate, params)
	if err != nil {
		return err
	}
	//todo error no do
	ws.RegisterMsg(ws.OrderBook, ctx)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	err = ws.Wss.SendMsg(ctx)
	return err
}

func (ws *FuturesClient) UserTrades(ctx context.Context) error {
	uid := text.GetString(ws.Ctx, exch.Uid)
	symbol := text.GetString(ctx, exch.CtxSymbol)
	msg, err := ws.SubscribeChannel(ChannelUserTrade, []string{uid, symbol})
	if err != nil {
		return err
	}
	ws.RegisterMsg(ws.UserTrades, ctx)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	err = ws.Wss.SendMsg(ctx)
	return err
}

func (ws *FuturesClient) Order(ctx context.Context) error {
	uid := text.GetString(ws.Ctx, exch.Uid)
	symbol := text.GetString(ctx, exch.CtxSymbol)
	msg, err := ws.SubscribeChannel(ChannelOrders, []string{uid, symbol})
	if err != nil {
		return err
	}
	ws.RegisterMsg(ws.Order, ctx)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	err = ws.Wss.SendMsg(ctx)
	return err
}

func (ws *FuturesClient) Position(ctx context.Context) error {
	uid := text.GetString(ws.Ctx, exch.Uid)
	symbol := text.GetString(ctx, exch.CtxSymbol)
	msg, err := ws.SubscribeChannel(ChannelPositions, []string{uid, symbol})
	if err != nil {
		return err
	}
	ws.RegisterMsg(ws.Position, ctx)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	err = ws.Wss.SendMsg(ctx)
	return err
}

func (ws *FuturesClient) Balance(ctx context.Context) error {
	uid := text.GetString(ws.Ctx, exch.Uid)
	msg, err := ws.SubscribeChannel(ChannelBalances, []string{uid})
	if err != nil {
		return err
	}
	ws.RegisterMsg(ws.Balance, ctx)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	err = ws.Wss.SendMsg(ctx)
	return err
}

func (ws *FuturesClient) Trades() error {
	return nil
}

func (ws *FuturesClient) ReceivedMsg() {
	mh := NewMsgHandler()
	for {
		select {
		case <-ws.Ctx.Done():
			log.Warnln(log.Wss, ws.Sign, " gate wss FuturesClient ReceivedMsg return by done")
			return
		case msg := <-*ws.Wss.MsgQueue:
			if msg != nil {
				mh.ReadMessage(msg, ws.MsgQueue)
			} else {
				close(*ws.MsgQueue)
				q := make(exch.MsgQueue, exch.ClientChannelLen)
				ws.MsgQueue = &q
			}
		}
	}
}

func (ws *FuturesClient) SubscribeChannel(channel string, payload []string) ([]byte, error) {
	return ws.SignMsg(Subscribe, channel, payload)
}

func (ws *FuturesClient) UnSubscribeChannel(channel string, payload []string) ([]byte, error) {
	return ws.SignMsg(UnSubscribe, channel, payload)
}

func (ws *FuturesClient) SignMsg(subtype, channel string, payload []string) ([]byte, error) {
	// subscribe msg
	t := time.Now().Unix()
	msg := NewMsg(channel, subtype, t, payload)
	key := text.GetString(ws.Ctx, exch.Key)
	secret := text.GetString(ws.Ctx, exch.Secret)
	if key != "" && secret != "" {
		msg.Sign(key, secret)
	}
	msgByte, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return msgByte, nil
}

func (ws *FuturesClient) RegisterMsg(action func(ctx context.Context) error, param context.Context) {
	if text.GetBool(param, client.IsReconnect) {
		ch := exch.GetChan(param)
		if ch != nil {
			*ch <- 1
		}
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

func (ws *FuturesClient) ReConnect(err error) error {
	if ws.Status == 2 {
		return nil
	}
	ws.Status = 2
	conErr := ws.Wss.Connent()
	if conErr != nil {
		ws.Status = 0
		return conErr
	}
	log.Warnln(log.Wss, ws.Sign, "gate wss FuturesClient start reconnect")
	for _, m := range ws.ReConnectMsg {
		fun := *m.Action
		ctx := m.Param
		err := fun(ctx)
		if err != nil {
			ws.Status = 0
			ws.Wss.Connect.Close()
			//todo last link err
			return err
		}
	}
	go ws.Wss.ReadMsg()
	time.Sleep(3 * time.Second)
	ws.ClientId++
	ws.Status = 1
	ws.ConnectTime = timer.MicNow()
	return nil
}
