package spot_wss

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

type SpotClient struct {
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

func NewSpotClient(ctx context.Context) (*SpotClient, error) {
	q := make(exch.MsgQueue, exch.ClientChannelLen)
	sg := text.GetString(ctx, exch.ConnSign)
	cl := &SpotClient{
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
func (sc *SpotClient) NewClient() error {
	ctx := context.WithValue(sc.Ctx, client.WssUrl, WssSubUrl)
	ctx = context.WithValue(ctx, client.Keepalive, true)
	ctx = context.WithValue(ctx, client.Timeout, WssTimeout)
	ctx = context.WithValue(ctx, client.MsgLen, exch.WSChannelLen)
	ctx = context.WithValue(ctx, client.Id, sc.Sign)
	wss, err := client.NewWssSocket(ctx, sc.ReConnect)
	if err != nil {
		log.Errorln(log.Wss, sc.Sign, "gate wss SpotClient error ", err)
		return err
	}
	sc.Wss = wss
	sc.Status = 1
	sc.ConnectTime = timer.MicNow()
	go sc.ReceivedMsg()
	return nil
}

func (sc *SpotClient) Tickers(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	params := []string{symbol}
	msg, err := sc.SubscribeChannel(ChannelTickers, params)
	if err != nil {
		return err
	}
	sc.RegisterMsg(sc.Tickers, ctx)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	err = sc.Wss.SendMsg(ctx)
	return err
}

func (sc *SpotClient) OrderBook(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	frequency := "100ms" // 100ms, 1000ms
	params := []string{symbol, frequency}
	msg, err := sc.SubscribeChannel(ChannelDepthUpdate, params)
	if err != nil {
		return err
	}
	//todo error no do
	sc.RegisterMsg(sc.OrderBook, ctx)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	err = sc.Wss.SendMsg(ctx)
	return err
}

func (sc *SpotClient) UserTrades(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	msg, err := sc.SubscribeChannel(ChannelUserTrade, []string{symbol})
	if err != nil {
		return err
	}
	sc.RegisterMsg(sc.UserTrades, ctx)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	err = sc.Wss.SendMsg(ctx)
	return err
}

func (sc *SpotClient) Order(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	msg, err := sc.SubscribeChannel(ChannelOrders, []string{symbol})
	if err != nil {
		return err
	}
	sc.RegisterMsg(sc.Order, ctx)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	err = sc.Wss.SendMsg(ctx)
	return err
}

func (sc *SpotClient) Position(ctx context.Context) error {
	msg, err := sc.SubscribeChannel(ChannelBalances, []string{})
	if err != nil {
		return err
	}
	sc.RegisterMsg(sc.Position, ctx)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	err = sc.Wss.SendMsg(ctx)
	return err
}

func (sc *SpotClient) Balance(ctx context.Context) error {
	uid := text.GetString(sc.Ctx, exch.Uid)
	msg, err := sc.SubscribeChannel(ChannelBalances, []string{uid})
	if err != nil {
		return err
	}
	sc.RegisterMsg(sc.Balance, ctx)
	ctx = context.WithValue(ctx, client.SendMsg, msg)
	err = sc.Wss.SendMsg(ctx)
	return err
}

func (sc *SpotClient) Trades() error {
	return nil
}

func (sc *SpotClient) ReceivedMsg() {
	mh := NewMsgHandler()
	for {
		select {
		case <-sc.Ctx.Done():
			log.Warnln(log.Wss, sc.Sign, " gate wss SpotClient ReceivedMsg return by done")
			return
		case msg := <-*sc.Wss.MsgQueue:
			if msg != nil {
				mh.ReadMessage(msg, sc.MsgQueue)
			} else {
				close(*sc.MsgQueue)
				q := make(exch.MsgQueue, exch.ClientChannelLen)
				sc.MsgQueue = &q
			}
		}
	}
}

func (sc *SpotClient) SubscribeChannel(channel string, payload []string) ([]byte, error) {
	return sc.SignMsg(Subscribe, channel, payload)
}

func (sc *SpotClient) UnSubscribeChannel(channel string, payload []string) ([]byte, error) {
	return sc.SignMsg(UnSubscribe, channel, payload)
}

func (sc *SpotClient) SignMsg(subtype, channel string, payload []string) ([]byte, error) {
	// subscribe msg
	t := time.Now().Unix()
	msg := NewMsg(channel, subtype, t, payload)
	key := text.GetString(sc.Ctx, exch.Key)
	secret := text.GetString(sc.Ctx, exch.Secret)
	if key != "" && secret != "" {
		msg.Sign(key, secret)
	}
	msgByte, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return msgByte, nil
}

func (sc *SpotClient) RegisterMsg(action func(ctx context.Context) error, param context.Context) {
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
	sc.ReConnectMsg = append(sc.ReConnectMsg, method)
	return
}

func (sc *SpotClient) ReConnect(err error) error {
	if sc.Status == 2 {
		return nil
	}
	sc.Status = 2
	conErr := sc.Wss.Connent()
	if conErr != nil {
		sc.Status = 0
		return conErr
	}
	log.Warnln(log.Wss, sc.Sign, "gate wss SpotClient start reconnect")
	for _, m := range sc.ReConnectMsg {
		fun := *m.Action
		ctx := m.Param
		err := fun(ctx)
		if err != nil {
			sc.Status = 0
			sc.Wss.Connect.Close()
			//todo last link err
			return err
		}
	}
	go sc.Wss.ReadMsg()
	time.Sleep(3 * time.Second)
	sc.ClientId++
	sc.Status = 1
	sc.ConnectTime = timer.MicNow()
	return nil
}
