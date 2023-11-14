package exch

import (
	"context"

	"high-freq-quant-go/core/log"
)

type Exchange interface {
	GetStatus() int                                 //获取交易所状态 1可用 其他不可用
	GetApiSign() string                             //获取Api标识
	GetExName() string                              //获取交易所名称
	GetExType() string                              //获取交易所类型
	GetBaseInfo(ctx context.Context) *BaseInfo      //交易基本信息
	GetPosition(ctx context.Context) *Position      //获取仓位
	GetOrder(ctx context.Context) map[string]*Order //获取当前委托单
	GetOrderBook(ctx context.Context) *Booker       //获取订单薄
	GetBalance(ctx context.Context) *Balance        //获取当前账号资金

	GetTradeChan(ctx context.Context) *chan *Order //获取成交推送队列

	ListAsset(ctx context.Context) (*Balance, map[string]*Position) //获取资产及头寸列表

	CreateOrder(ctx context.Context) (*Order, error)        //创建订单
	CreateBatchOrder(ctx context.Context) ([]*Order, error) //批量创建订单
	CannelOrder(ctx context.Context) (*Order, error)        //取消订单
	CannelAllOrder(ctx context.Context) ([]*Order, error)   //取消所有订单

	UpdateLeverage(ctx context.Context) (*Position, error) //更新杠杠(逐仓)
	UpdateMargin(ctx context.Context) (*Position, error)   //更新保证金

	SubTicker(ctx context.Context) error    //基础信息
	SubOrderBook(ctx context.Context) error //订阅订单薄
	SubOrder(ctx context.Context) error     //订阅用户委托单
	SubUserTrade(ctx context.Context) error //订阅用户成交单
	SubPosition(ctx context.Context) error  //订阅用户仓位
	SubBalance(ctx context.Context) error   //订阅账号资金
}

type ConnInstance func(ctx context.Context) Exchange

var AllConn = make(map[string]ConnInstance)

func Register(name string, adapter ConnInstance) {
	if adapter == nil {
		panic("exch.Connect: Register adapter is nil")
	}
	if _, ok := AllConn[name]; ok {
		panic("exch.Connect: Register called twice for adapter " + name)
	}
	AllConn[name] = adapter
}

func NewConn(ctx context.Context, exchName string) (conn Exchange) {
	instanceFunc, ok := AllConn[exchName]
	if !ok {
		log.Errorf(log.Conn, "exchange conn is not register %s (forgot to import?)", exchName)
		return nil
	}
	conn = instanceFunc(ctx)
	return
}
