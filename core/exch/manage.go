package exch

import (
	"context"
	"sync"

	"high-freq-quant-go/adapter/text"

	"high-freq-quant-go/core/config"

	"high-freq-quant-go/core/log"
)

type Exchanger struct {
	Sign     string
	NameType string
	Id       string
	Ex       Exchange
	Ctx      context.Context
	Cancel   context.CancelFunc
	sc       sync.Mutex
}

func NewExchanger(ctx context.Context, id string) *Exchanger {
	AllManage.sc.Lock()
	defer AllManage.sc.Unlock()
	mn := &Exchanger{
		Ctx: ctx,
		Id:  id,
	}
	mn.GetSign()
	if mn.Sign == "" || mn.NameType == "" {
		log.Warnln(log.Conn, "exchange Sign or NameType is empty; Sign:", mn.Sign, "NameType:", mn.NameType)
		return nil
	}
	m := AllManage.GetEx(mn.Sign)
	if m != nil {
		log.Warnln(log.Conn, "exchange conn is useing NameType, cid:", m.NameType, id)
		return m
	}
	ctx = context.WithValue(ctx, ConnSign, mn.Sign)
	mn.Ctx, mn.Cancel = context.WithCancel(ctx)
	conn := NewConn(mn.Ctx, mn.NameType)
	if conn == nil {
		mn.Cancel()
		return nil
	}
	mn.Ex = conn
	AllManage.SetEx(mn.Sign, mn)
	return mn
}

func (mn *Exchanger) GetSign() {
	cname := text.GetString(mn.Ctx, CtxExname)
	if cname == "" {
		return
	}
	ctype := text.GetString(mn.Ctx, CtxExtype)
	if ctype == "" {
		return
	}
	apiSign := text.GetString(mn.Ctx, ApiSign)
	if apiSign == "" {
		return
	}
	mn.NameType = cname + "_" + ctype
	mn.Sign = apiSign + "_" + mn.Id
}

func Delete(apiSign, id string) {
	sign := apiSign + "_" + id
	AllManage.DelEx(sign)
}

func ApiCtx(api *config.ApiUser) context.Context {
	ctx := context.Background()
	if api == nil {
		return ctx
	}
	ctx = context.WithValue(ctx, ApiSign, api.ApiSign)
	ctx = context.WithValue(ctx, CtxExname, api.ExName)
	ctx = context.WithValue(ctx, CtxExtype, api.ExType)
	ctx = context.WithValue(ctx, Uid, api.Uid)
	ctx = context.WithValue(ctx, Key, api.Key)
	ctx = context.WithValue(ctx, Secret, api.Secret)
	return ctx
}
