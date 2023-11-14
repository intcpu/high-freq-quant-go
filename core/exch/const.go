package exch

type MsgQueue chan interface{}

const (
	Binance  = "binance"
	Gate     = "gate"
	Futures  = "futures"
	Spot     = "spot"
	Delivery = "delivery"
)

const (
	Uid    = "Uid"
	Key    = "Key"
	Secret = "Secret"

	CtxHttp   = "CtxHttp"
	CtxInit   = "CtxInit"
	CtxSymbol = "CtxSymbol"
	CtxBase   = "CtxBase"
	CtxExname = "CtxExname"
	CtxExtype = "CtxExtype"

	CtxOrder  = "CtxOrder"
	CtxOrders = "CtxOrders"
	CtxLv     = "CtxLeverage"
	CtxChange = "CtxChange"
	Symbols   = "CtxSymbols"
	CtxChan   = "CtxChan"

	ApiSign  = "ApiSign"
	ConnSign = "ConnSign"
)

const (
	WSChannelLen     int64 = 6000
	ClientChannelLen int64 = 4000
	MsgChannelLen    int64 = 2000

	BookDataChannel = "BookDataChannel"
	BookMsgChan     = "BookMsgChan"
)
