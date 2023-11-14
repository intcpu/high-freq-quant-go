package futures_wss

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
	UserUsdtWssUrl = "wss://fstream.binance.com/ws/"

	AccountUpdate    = "ACCOUNT_UPDATE"
	OrderTradeUpdate = "ORDER_TRADE_UPDATE"

	AccountConfigUpdate = "ACCOUNT_CONFIG_UPDATE"
	MarginCall          = "MARGIN_CALL"
	ListenKeyExpired    = "listenKeyExpired"
)

const (
	UserWssSign = "binance-user-"
)
