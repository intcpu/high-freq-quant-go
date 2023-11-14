package client

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"high-freq-quant-go/adapter/text"

	"high-freq-quant-go/core/log"

	"github.com/gorilla/websocket"
)

type WssSocket struct {
	Ctx          context.Context
	Id           string
	wssUrl       string
	Dialer       *websocket.Dialer
	Connect      *websocket.Conn
	Keepalive    bool
	Timeout      time.Duration
	MsgLen       int64
	MsgQueue     *chan *[]byte
	Status       int //0:init 1:opened 2closed 3:reconnecting
	ReConnectMsg []context.Context
	ErrorHandle  func(err error) error
	lastResponse int64
}

func NewWssSocket(ctx context.Context, errorHandle func(error) error) (*WssSocket, error) {
	sc := WssSocket{
		Ctx:         ctx,
		ErrorHandle: errorHandle,
	}
	sc.Init()
	err := sc.Connent()
	if err != nil {
		return &sc, err
	}
	go sc.ReadMsg()
	if sc.Keepalive {
		go sc.PingPong()
	}
	//go sc.TestClose()
	return &sc, nil
}

func (sc *WssSocket) Init() {
	sc.Dialer = websocket.DefaultDialer
	sc.Dialer.TLSClientConfig = &tls.Config{RootCAs: nil, InsecureSkipVerify: true}
	pUrl := text.GetString(sc.Ctx, ProxyUrl)
	if pUrl != "" {
		uProxy, _ := url.Parse(pUrl)
		sc.Dialer.Proxy = http.ProxyURL(uProxy)
	}
	sc.Id = text.GetString(sc.Ctx, Id)
	sc.wssUrl = text.GetString(sc.Ctx, WssUrl)
	sc.Keepalive = text.GetBool(sc.Ctx, Keepalive)
	sc.Timeout = time.Duration(text.GetInt64(sc.Ctx, Timeout))
	sc.MsgLen = text.GetInt64(sc.Ctx, MsgLen)
	sc.ReConnectMsg = []context.Context{}
}

func (sc *WssSocket) Connent() error {
	c, _, err := sc.Dialer.Dial(sc.wssUrl, nil)
	ti := time.Now().Unix()
	if err != nil {
		log.Errorln(log.Wss, sc.Id, sc.wssUrl, "wss connect create error", ti)
		return err
	}
	log.Infoln(log.Wss, sc.Id, sc.wssUrl, "wss connect create success at time ", ti)
	if sc.Connect != nil {
		sc.Connect.Close()
	}
	sc.Connect = c
	sc.Status = 1
	sc.lastResponse = ti
	if sc.MsgQueue != nil {
		close(*sc.MsgQueue)
	}
	q := make(chan *[]byte, sc.MsgLen)
	sc.MsgQueue = &q
	sc.Connect.SetPongHandler(func(msg string) error {
		//todo
		//log.Infoln(log.Wss, sc.Id, " WssSocket PingPong end")
		sc.lastResponse = time.Now().Unix()
		return nil
	})
	return nil
}

func (sc *WssSocket) TestClose() {
	// 假装断线
	ticker := time.NewTicker(35 * time.Second)
	for {
		select {
		case <-sc.Ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			sc.Connect.Close()
		}
	}
}

func (sc *WssSocket) SendMsg(ctx context.Context) error {
	msg := text.GetByte(ctx, SendMsg)
	log.Infoln(log.Wss, sc.Id, " SendMsg:", string(msg))
	sc.RegisterMsg(ctx)
	return sc.Connect.WriteMessage(websocket.TextMessage, msg)
}

func (sc *WssSocket) CancelMsg(ctx context.Context) error {
	msg := text.GetByte(ctx, SendMsg)
	log.Infoln(log.Wss, sc.Id, " CancelMsg:", string(msg))
	return sc.Connect.WriteMessage(websocket.TextMessage, msg)
}

func (sc *WssSocket) ReadMsg() {
	if sc.Connect == nil {
		return
	}
	for {
		select {
		case <-sc.Ctx.Done():
			sc.Closed()
			log.Warnln(log.Wss, sc.Id, " WssSocket ReadMsg return by done")
			return
		default:
			msgType, message, err := sc.Connect.ReadMessage()
			if err != nil {
				//if err.Error() == "websocket: close 1006 (abnormal closure): unexpected EOF" {
				//	continue
				//}
				log.Errorln(log.Wss, sc.Id, " wss ReadMsg error: ", err)
				sc.Closed()
				reErr := sc.ErrorReConnect(err)
				if reErr != nil {
					log.Errorln(log.Wss, sc.Id, " wss ReConnect error: ", reErr)
					time.Sleep(sc.Timeout * time.Second)
					continue
				}
				return
			}
			sc.lastResponse = time.Now().Unix()
			if sc.MsgQueue == nil {
				continue
			}
			switch msgType {
			case websocket.TextMessage: //文本数据
				*sc.MsgQueue <- &message
			case websocket.BinaryMessage: //二进制数据
				*sc.MsgQueue <- &message
			case websocket.CloseMessage: //关闭
				log.Warnln(log.Wss, sc.Id, "received close")
			case websocket.PingMessage: //Ping
				log.Infoln(log.Wss, sc.Id, "received ping")
			case websocket.PongMessage: //Pong
				log.Infoln(log.Wss, sc.Id, "received pong")
			default:
				log.Warnln(log.Wss, sc.Id, "unkown msgType", msgType)
			}
		}
	}
}

func (sc *WssSocket) ErrorReConnect(err error) error {
	// error handle, no return
	var reErr error
	if sc.ErrorHandle == nil {
		reErr = sc.ReConnect()
	} else {
		reErr = sc.ErrorHandle(err)
	}
	return reErr
}

func (sc *WssSocket) PingPong() {
	timeout := sc.Timeout * time.Second
	tr := time.NewTicker(timeout)
	for {
		select {
		case <-sc.Ctx.Done():
			tr.Stop()
			log.Warnln(log.Wss, sc.Id, " WssSocket PingPong return by done")
			return
		case <-tr.C:
			//todo
			//log.Infoln(log.Wss, sc.Id, " WssSocket PingPong start")
			deadline := time.Now().Add(timeout)
			err := sc.Connect.WriteControl(websocket.PingMessage, []byte{}, deadline)
			if err != nil {
				log.Errorln(log.Wss, sc.Id, "ping err", err)
				sc.Closed()
			}
		}
	}

}

func (sc *WssSocket) RegisterMsg(param context.Context) {
	if text.GetBool(param, IsReconnect) {
		return
	}
	param = context.WithValue(param, IsReconnect, true)
	sc.ReConnectMsg = append(sc.ReConnectMsg, param)
	return
}

func (sc *WssSocket) ReConnect() error {
	if sc.Status != 2 {
		return nil
	}
	sc.Status = 3
	err := sc.Connent()
	if err != nil {
		sc.Status = 2
		return err
	}
	go sc.ReadMsg()
	log.Infoln(log.Wss, sc.Id, "wss restart connect")
	for _, m := range sc.ReConnectMsg {
		cerr := sc.SendMsg(m)
		if cerr != nil {
			log.Errorln(log.Wss, sc.Id, "wss restart send msg err", cerr)
		}
	}
	sc.Status = 1
	return nil
}

func (sc *WssSocket) Closed() {
	sc.Status = 2
	sc.Connect.Close()
}
