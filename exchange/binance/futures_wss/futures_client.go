package futures_wss

import (
	"context"
	"encoding/json"
	"math/rand"
	"strings"
	"time"

	"high-freq-quant-go/adapter/client"
	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/adapter/timer"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/log"
	"high-freq-quant-go/exchange/binance/unify"
)

type FuturesClient struct {
	//sign
	Sign string
	//param
	Ctx context.Context

	//all msg chan
	MsgQueue *exch.MsgQueue
	//wss connect
	Wss *client.WssSocket

	//reconnect id
	ClientId int64
	//0断开1已连接2连接中
	Status      int
	ConnectTime int64

	//reconnect
	ReConnectMsg []client.SubscribeData
}

func NewFuturesClient(ctx context.Context) *FuturesClient {
	q := make(exch.MsgQueue, exch.ClientChannelLen)
	sign := text.GetString(ctx, exch.ConnSign)
	ws := &FuturesClient{
		Sign:     PublicWssSign + sign,
		Ctx:      ctx,
		MsgQueue: &q,
	}
	return ws
}

func (ws *FuturesClient) NewClient() error {
	ctx := context.WithValue(ws.Ctx, client.WssUrl, UsdtWssUrl)
	ctx = context.WithValue(ctx, client.Keepalive, true)
	ctx = context.WithValue(ctx, client.Timeout, WssTimeout)
	ctx = context.WithValue(ctx, client.MsgLen, exch.WSChannelLen)
	ctx = context.WithValue(ctx, client.Id, ws.Sign)
	wss, err := client.NewWssSocket(ctx, ws.ReConnect)
	if err != nil {
		log.Errorln(log.Wss, ws.Sign, "binance wss NewClient error ", err)
		return err
	}
	ws.Wss = wss
	ws.Status = 1
	ws.ConnectTime = timer.MicNow()
	go ws.ReceivedMsg()
	return nil
}

func (ws *FuturesClient) OrderBook(ctx context.Context) error {
	if ws.Wss == nil {
		ws.NewClient()
	}
	symbol := WssSymbol(ctx)
	param := []string{
		symbol + "@depth@100ms",
	}
	msg := ws.SubscribeChannel(param)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	ws.RegisterMsg(ws.OrderBook, ctx)
	err := ws.Wss.SendMsg(ctx)
	return err
}

func (ws *FuturesClient) MarkPrice(ctx context.Context) error {
	if ws.Wss == nil {
		ws.NewClient()
	}
	symbol := WssSymbol(ctx)
	param := []string{
		symbol + "@markPrice@1s",
	}
	msg := ws.SubscribeChannel(param)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	ws.RegisterMsg(ws.MarkPrice, ctx)
	err := ws.Wss.SendMsg(ctx)
	return err
}

func (ws *FuturesClient) Ticker(ctx context.Context) error {
	if ws.Wss == nil {
		ws.NewClient()
	}
	symbol := WssSymbol(ctx)
	param := []string{
		symbol + "@ticker",
	}
	msg := ws.SubscribeChannel(param)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	ws.RegisterMsg(ws.Ticker, ctx)
	err := ws.Wss.SendMsg(ctx)
	return err
}

func (ws *FuturesClient) SubscribeChannel(param []string) []byte {
	request := make(map[string]interface{})
	request["method"] = "SUBSCRIBE"
	request["params"] = param
	request["id"] = rand.Int()
	msg, _ := json.Marshal(request)
	return msg
}

func (ws *FuturesClient) ReceivedMsg() {
	for {
		select {
		case <-ws.Ctx.Done():
			log.Warnln(log.Wss, ws.Sign, "binance wss ReceivedMsg return by done")
			return
		case msg := <-*ws.Wss.MsgQueue:
			if msg != nil {
				ReadPublicMessage(msg, ws.MsgQueue)
			} else {
				close(*ws.MsgQueue)
				q := make(exch.MsgQueue, exch.ClientChannelLen)
				ws.MsgQueue = &q
			}
		}
	}
}

func (ws *FuturesClient) RegisterMsg(action func(ctx context.Context) error, param context.Context) {
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
	log.Warnln(log.Wss, ws.Sign, ws.Wss.Id, "binance wss start reconnect")
	for _, m := range ws.ReConnectMsg {
		fun := *m.Action
		ctx := m.Param
		err := fun(ctx)
		if err != nil {
			ws.Status = 0
			ws.Wss.Connect.Close()
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

func WssSymbol(ctx context.Context) string {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	symbol = unify.SymbolToB(symbol)
	return strings.ToLower(symbol)
}
