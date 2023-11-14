package spot_wss

import "sync"

var (
	BinaceUserWss = NewBinaceUserWssMap()
)

type BinaceUserWssMap struct {
	Wss  map[string]*UserWss
	lock sync.Mutex
}

func NewBinaceUserWssMap() *BinaceUserWssMap {
	m := &BinaceUserWssMap{
		Wss: map[string]*UserWss{},
	}
	return m
}

const (
	UserWssUrl = "wss://stream.binance.com:9443/ws/"

	AccountUpdate    = "outboundAccountPosition"
	OrderTradeUpdate = "executionReport"

	ListenKeyExpired = "listenKeyExpired"
)

const (
	UserWssSign = "binance-user-"
)
