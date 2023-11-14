package futures_wss

import (
	"encoding/json"
)

const (
	BinanceFuturesU = "binance"
	UsdtWssUrl      = "wss://fstream.binance.com/stream"
	BtcWssUrl       = "wss://dstream.binance.com/stream"
	UsdtHttpUrl     = "https://api.gateio.ws/api/v4/futures/usdt/"
)

const (
	//base url
	SpotUrl     = "https://api.binance.com"
	UFuturesUrl = "https://fapi.binance.com"

	//futures
	OrderBookDepth = "/fapi/v1/depth"
	OrderEndpoint  = "/fapi/v1/order"

	//spot
	Accountinfo = "/api/v3/account"
)

const (
	PublicWssSign = "binance-public-"

	Subscribe                = "subscribe"
	UnSubscribe              = "unsubscribe"
	TypeDepthUpdate          = "depthUpdate"
	TypeTrade                = "trade"
	TypeAggTrade             = "aggTrade"
	TypeTicker               = "24hrTicker"
	TypeMarkPrice            = "markPriceUpdate"
	TypeKline                = "kline"
	InitOrderBookLimit int64 = 1000 //10 20 50 100 500 1000
	KlineLen                 = 50

	WssTimeout int64 = 30
)

// WsTradeMsg
type TradeMsg struct {
	Stream string     `json:"stream"`
	Data   TradeEvent `json:"data"`
}

// WsTradeEvent define websocket trade event
type TradeEvent struct {
	Event     string `json:"e"`
	Time      int64  `json:"E"`
	TradeTime int64  `json:"T"`
	Symbol    string `json:"s"`
	TradeID   int64  `json:"t"`
	Price     string `json:"p"`
	Quantity  string `json:"q"`
	//BuyerOrderID  int64  `json:"b"` // expired
	//SellerOrderID int64  `json:"a"`
	IsBuyerMaker bool `json:"m"`
	Placeholder  bool `json:"M"` // add this field to avoid case insensitive unmarshaling
}

// WsDepthMsg
type DepthMsg struct {
	Stream string     `json:"stream"`
	Data   DepthEvent `json:"data"`
}

// WsDepthEvent define websocket depth event
type DepthEvent struct {
	Event         string `json:"e"`
	Time          int64  `json:"E"`
	TradeTime     int64  `json:"T"`
	Symbol        string `json:"s"`
	UpdateID      int64  `json:"u"`
	FirstUpdateID int64  `json:"U"`
	PU            int64  `json:"pu"` // parent uid
	Bids          []Bid  `json:"b"`
	Asks          []Ask  `json:"a"`
}

type MarkPriceMsg struct {
	Stream string         `json:"stream"`
	Data   MarkPriceEvent `json:"data"`
}

type MarkPriceEvent struct {
	Event           string  `json:"e"`
	Time            int64   `json:"E"`
	Symbol          string  `json:"s"`
	MarkPrice       float64 `json:"p,string,omitempty"`
	IndexPrice      float64 `json:"i,string,omitempty"`
	SettlePrice     float64 `json:"P,string,omitempty"`
	LastFundingRate float64 `json:"r,string,omitempty"`
	NextFundingTime int64   `json:"T"`
}

type TickerMsg struct {
	Stream string      `json:"stream"`
	Data   TickerEvent `json:"data"`
}

type TickerEvent struct {
	Event              string  `json:"e"`
	Time               int64   `json:"E"`
	Symbol             string  `json:"s"`
	PriceChange        float64 `json:"p,string,omitempty"`
	PriceChangePercent float64 `json:"P,string,omitempty"`
	WeightedAvgPrice   float64 `json:"w,string,omitempty"`
	ClosePrice         float64 `json:"c,string,omitempty"`
	CloseQty           float64 `json:"Q,string,omitempty"`
	OpenPrice          float64 `json:"o,string,omitempty"`
	HighPrice          float64 `json:"h,string,omitempty"`
	LowPrice           float64 `json:"l,string,omitempty"`
	BaseVolume         float64 `json:"v,string,omitempty"`
	QuoteVolume        float64 `json:"q,string,omitempty"`
	OpenTime           int64   `json:"O"`
	CloseTime          int64   `json:"C"`
	FirstID            int64   `json:"F"`
	LastID             int64   `json:"L"`
	TradeCount         int64   `json:"n"`
}

// Bid define bid info with price and quantity
type Bid struct {
	Price    string `json:"string,omitempty"`
	Quantity string `json:"string,omitempty"`
}

// Ask define ask info with price and quantity
type Ask struct {
	Price    string `json:"string,omitempty"`
	Quantity string `json:"string,omitempty"`
}

func (bid *Bid) UnmarshalJSON(bz []byte) error {
	var tmp [2]string
	err := json.Unmarshal(bz, &tmp)
	if err != nil {
		return err
	}
	bid.Price = tmp[0]
	bid.Quantity = tmp[1]
	return nil
}

func (ask *Ask) UnmarshalJSON(bz []byte) error {
	var tmp [2]string
	err := json.Unmarshal(bz, &tmp)
	if err != nil {
		return err
	}
	ask.Price = tmp[0]
	ask.Quantity = tmp[1]
	return nil
}

//func (bid *Bid) MarshalJSON() ([]byte, error) {
//	var tmp [2]string
//	tmp[0] = bid.Price
//	tmp[1] = convert.GetFloat64(bid.Quantity)
//
//	return json.Marshal(tmp)
//}
//
//func (ask *Ask) MarshalJSON() ([]byte, error) {
//	var tmp [2]string
//	tmp[0] = ask.Price
//	tmp[1] = ask.Quantity
//
//	return json.Marshal(tmp)
//}

type BestBookTickerMsg struct {
	Stream string         `json:"stream"`
	Data   BestBookTicker `json:"data"`
}

/* raw data
{
  "e":"bookTicker",     // 事件类型
  "u":400900217,        // 更新ID
  "E": 1568014460893,   // 事件推送时间
  "T": 1568014460891,   // 撮合时间
  "s":"BNBUSDT",        // 交易对
  "b":"25.35190000",    // 买单最优挂单价格
  "B":"31.21000000",    // 买单最优挂单数量
  "a":"25.36520000",    // 卖单最优挂单价格
  "A":"40.66000000"     // 卖单最优挂单数量
}
*/
// best book ticker ws message
type BestBookTicker struct {
	BookTicker    string `json:"e"`
	UpdateID      int64  `json:"u"`
	EventTime     int64  `json:"E"`
	LastMatchTime int64  `json:"T"`
	Symbol        string `json:"s"`
	BestBid       string `json:"b"`
	BestBidAmount string `json:"B"`
	BestAsk       string `json:"a"`
	BestAskAmount string `json:"A"`
}

// WsMarketStatEvent define websocket market statistics event
type MarketStatEvent struct {
	Event              string `json:"e"`
	Time               int64  `json:"E"`
	Symbol             string `json:"s"`
	PriceChange        string `json:"p"`
	PriceChangePercent string `json:"P"`
	WeightedAvgPrice   string `json:"w"`
	PrevClosePrice     string `json:"x"`
	LastPrice          string `json:"c"`
	CloseQty           string `json:"Q"`
	BidPrice           string `json:"b"`
	BidQty             string `json:"B"`
	AskPrice           string `json:"a"`
	AskQty             string `json:"A"`
	OpenPrice          string `json:"o"`
	HighPrice          string `json:"h"`
	LowPrice           string `json:"l"`
	BaseVolume         string `json:"v"`
	QuoteVolume        string `json:"q"`
	OpenTime           int64  `json:"O"`
	CloseTime          int64  `json:"C"`
	FirstID            int64  `json:"F"`
	LastID             int64  `json:"L"`
	Count              int64  `json:"n"`
}

type KlineMsg struct {
	Stream string     `json:"stream"`
	Data   KlineEvent `json:"data"`
}

// WsKlineEvent define websocket kline event
type KlineEvent struct {
	Event  string `json:"e"`
	Time   int64  `json:"E"`
	Symbol string `json:"s"`
	Kline  Kline  `json:"k"`
}

type Kline struct {
	StartTime          int64   `json:"t,omitempty" structs:"t,omitempty"`        // 这根K线的起始时间
	EndTime            int64   `json:"T,omitempty" structs:"T,omitempty"`        // // 这根K线的结束时间
	Symbol             string  `json:"s,omitempty" structs:"s,omitempty"`        // 交易对
	Interval           string  `json:"i,omitempty" structs:"i,omitempty"`        // K线间隔
	FirstId            int64   `json:"f,omitempty" structs:"f,omitempty"`        // 这根K线期间第一笔成交ID
	LastId             int64   `json:"L,omitempty" structs:"L,omitempty"`        // 这根K线期间末一笔成交ID
	FirstPrice         float64 `json:"o,string,omitempty" structs:"o,omitempty"` // 这根K线期间第一笔成交价 开盘价
	LastPrice          float64 `json:"c,string,omitempty" structs:"c,omitempty"` // 这根K线期间末一笔成交价 收盘价
	HeightPrice        float64 `json:"h,string,omitempty" structs:"h,omitempty"` // 这根K线期间最高成交价
	LowPrice           float64 `json:"l,string,omitempty" structs:"l,omitempty"` // 这根K线期间最低成交价
	Volume             float64 `json:"v,string,omitempty" structs:"v,omitempty"` // 这根K线期间成交量
	Number             int     `json:"n,omitempty" structs:"n,omitempty"`        // 这根K线期间成交笔数
	TheEnd             bool    `json:"x,omitempty" structs:"x,omitempty"`        // 这根K线是否完结(是否已经开始下一根K线)
	Turnover           float64 `json:"q,string,omitempty" structs:"q,omitempty"` // 这根K线期间成交额
	InitiativeVolume   float64 `json:"V,string,omitempty" structs:"V,omitempty"` // 主动买入的成交量
	InitiativeTurnover float64 `json:"Q,string,omitempty" structs:"Qomitempty"`  // 主动买入的成交额
	B                  string  `json:"B,omitempty" structs:"B,omitempty"`        // 忽略此参数
}

type AggTradeMsg struct {
	Stream string   `json:"stream"`
	Data   AggTrade `json:"data"`
}

type AggTrade struct {
	Event             string `json:"e,omitempty" structs:"e,omitempty"` // 事件类型
	ETime             int64  `json:"E,omitempty" structs:"E,omitempty"` // 事件时间
	Symbol            string `json:"s,omitempty" structs:"s,omitempty"` // 交易对
	CollectionId      int64  `json:"a,omitempty" structs:"a,omitempty"` // 归集成交 ID
	Price             string `json:"p,omitempty" structs:"p,omitempty"` // 成交价格
	Volume            string `json:"q,omitempty" structs:"q,omitempty"` // 成交量
	FirstCollectionId int64  `json:"f,omitempty" structs:"f,omitempty"` // 被归集的首个交易ID
	LastCollectionId  int64  `json:"l,omitempty" structs:"l,omitempty"` // 被归集的末次交易ID
	VTime             int64  `json:"T,omitempty" structs:"T,omitempty"` // 成交时间
	Maker             bool   `json:"m,omitempty" structs:"m,omitempty"` // 买方是否是做市方。如true，则此次成交是一个主动卖出单，否则是一个主动买入单。
}

type BinanceOrderBookSnapshot struct {
	LastUpdateId int64  `json:"lastUpdateId"`
	Symbol       string `json:"symbol"`
	Unit         string `json:"unit"`
	Time         int64  `json:"E"`
	TradeTime    int64  `json:"T"`
	Bids         []Bid  `json:"bids"`
	Asks         []Ask  `json:"asks"`
}
