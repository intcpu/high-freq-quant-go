package spot_wss

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
)

const (
	WssSubUrl = "wss://api.gateio.ws/ws/v4/"
)

const (
	Subscribe          = "subscribe"
	UnSubscribe        = "unsubscribe"
	TypeDepthAll       = "all"
	MsgTypeUpdate      = "update"
	ChannelDepth       = "spot.order_book"
	ChannelTickers     = "spot.tickers"
	ChannelDepthUpdate = "spot.order_book_update"
	ChannelTrade       = "spot.trades"
	ChannelUserTrade   = "spot.usertrades"
	ChannelOrders      = "spot.orders"
	ChannelBalances    = "spot.balances"
	ChannelPositions   = "spot.balances"

	OrderBookNum              = "100"
	UnitCurrencyDecimal       = 5
	WssTimeout          int64 = 30
)

// gate message
type Msg struct {
	Time    int64    `json:"time"`
	Channel string   `json:"channel"`
	Event   string   `json:"event"`
	Payload []string `json:"payload"`
	Auth    *Auth    `json:"auth"`
}

func NewMsg(channel, event string, t int64, payload []string) *Msg {
	return &Msg{
		Time:    t,
		Channel: channel,
		Event:   event,
		Payload: payload,
	}
}

type Auth struct {
	Method string `json:"method"`
	KEY    string `json:"KEY"`
	SIGN   string `json:"SIGN"`
}

func sign(secret, channel, event string, t int64) string {
	message := fmt.Sprintf("channel=%s&event=%s&time=%d", channel, event, t)
	h2 := hmac.New(sha512.New, []byte(secret))
	_, err := io.WriteString(h2, message)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(h2.Sum(nil))
}

func (msg *Msg) Sign(key, secret string) {
	signStr := sign(secret, msg.Channel, msg.Event, msg.Time)
	msg.Auth = &Auth{
		Method: "api_key",
		KEY:    key,
		SIGN:   signStr,
	}
}

type DepthUpdateAllEvent struct {
	Time    int               `json:"time"`
	Channel string            `json:"channel"`
	Event   string            `json:"event"`
	Result  DepthUpdateResult `json:"result"`
}

type DepthUpdateResult struct {
	UpdateTimeMs int64      `json:"t"`
	E            string     `json:"e"`
	UpdateTime   int        `json:"E"`
	Symbol       string     `json:"s"`
	FirstId      int64      `json:"U"`
	LastId       int64      `json:"u"`
	Bids         [][]string `json:"b"`
	Asks         [][]string `json:"a"`
}

type OrderBookUpdate struct {
	Channel string                  `json:"channel"`
	Event   string                  `json:"event"`
	Time    int64                   `json:"time"`
	Result  []OrderBookUpdateResult `json:"result"`
}
type OrderBookUpdateResult struct {
	Price    string  `json:"p"`
	Quantity float64 `json:"s"`
	Symbol   string  `json:"c"`
	Id       int64   `json:"id"`
}

type OrderBookAll struct {
	Channel string             `json:"channel"`
	Event   string             `json:"event"`
	Time    int64              `json:"time"`
	Result  OrderBookAllResult `json:"result"`
}
type OrderBookAllResult struct {
	Symbol string `json:"contract"`
	Bids   []Bid  `json:"bids"`
	Asks   []Ask  `json:"asks"`
}

type Bid struct {
	Price    string  `json:"p"`
	Quantity float64 `json:"s"`
}

type Ask struct {
	Price    string  `json:"p"`
	Quantity float64 `json:"s"`
}

type TickersEvent struct {
	Time    int     `json:"time"`
	Channel string  `json:"channel"`
	Event   string  `json:"event"`
	Result  Tickers `json:"result"`
}

type Tickers struct {
	Symbol           string  `json:"currency_pair"`
	Last             float64 `json:"last,omitempty,string"`
	LowestAsk        float64 `json:"lowest_ask,omitempty,string"`
	HighestBid       float64 `json:"highest_bid,omitempty,string"`
	ChangePercentage float64 `json:"change_percentage,omitempty,string"`
	BaseVolume       float64 `json:"base_volume,omitempty,string"`
	QuoteVolume      float64 `json:"quote_volume,omitempty,string"`
	High24H          float64 `json:"high_24h,omitempty,string"`
	Low24H           float64 `json:"low_24h,omitempty,string"`
}

type UserTradeEvent struct {
	Time    int         `json:"time"`
	Channel string      `json:"channel"`
	Event   string      `json:"event"`
	Result  []UserTrade `json:"result"`
}

type UserTrade struct {
	Id           int64   `json:"id"`
	UserId       int64   `json:"user_id"`
	OrderId      string  `json:"order_id"`
	Symbol       string  `json:"currency_pair"`
	CreateTime   int64   `json:"create_time"`
	CreateTimeMs float64 `json:"create_time_ms,omitempty,string"`
	Side         string  `json:"side"`
	Amount       float64 `json:"amount,omitempty,string"`
	Role         string  `json:"role"`
	Price        float64 `json:"price,omitempty,string"`
	Fee          float64 `json:"fee,omitempty,string"`
	FeeCurrency  string  `json:"fee_currency"`
	PointFee     float64 `json:"point_fee,omitempty,string"`
	GtFee        float64 `json:"gt_fee,omitempty,string"`
	Text         string  `json:"text"`
}

type OrdersEvent struct {
	Time    int      `json:"time"`
	Channel string   `json:"channel"`
	Event   string   `json:"event"`
	Result  []Orders `json:"result"`
}

type Orders struct {
	Id                 string  `json:"id"`
	Text               string  `json:"text"`
	CreateTime         int64   `json:"create_time,omitempty,string"`
	UpdateTime         int64   `json:"update_time,omitempty,string"`
	Symbol             string  `json:"currency_pair"`
	Type               string  `json:"type"`
	Account            string  `json:"account"`
	Side               string  `json:"side"`
	Amount             float64 `json:"amount,omitempty,string"`
	Price              float64 `json:"price,omitempty,string"`
	TimeInForce        string  `json:"time_in_force"`
	Left               float64 `json:"left,omitempty,string"`
	FilledTotal        float64 `json:"filled_total,omitempty,string"`
	Fee                float64 `json:"fee,omitempty,string"`
	FeeCurrency        string  `json:"fee_currency"`
	PointFee           float64 `json:"point_fee,omitempty,string"`
	GtFee              float64 `json:"gt_fee,omitempty,string"`
	GtDiscount         bool    `json:"gt_discount"`
	RebatedFee         float64 `json:"rebated_fee,omitempty,string"`
	RebatedFeeCurrency string  `json:"rebated_fee_currency"`
	CreateTimeMs       int64   `json:"create_time_ms,omitempty,string"`
	UpdateTimeMs       int64   `json:"update_time_ms,omitempty,string"`
	User               int     `json:"user"`
	Event              string  `json:"event"`
}

type BalancesEvent struct {
	Channel string     `json:"channel"`
	Event   string     `json:"event"`
	Time    int64      `json:"time"`
	Result  []Balances `json:"result"`
}

type Balances struct {
	Timestamp   int64   `json:"timestamp,omitempty,string"`
	TimestampMs int64   `json:"timestamp_ms,omitempty,string"`
	User        int64   `json:"user,omitempty,string"`
	Currency    string  `json:"currency"`
	Change      float64 `json:"change,omitempty,string"`
	Total       float64 `json:"total,omitempty,string"`
	Available   float64 `json:"available,omitempty,string"`
}

type TradeEvent struct {
	Time    int     `json:"time"`
	Channel string  `json:"channel"`
	Event   string  `json:"event"`
	Result  []Trade `json:"result"`
}

type Trade struct {
	Id           int64   `json:"id"`
	UserId       int64   `json:"user_id"`
	OrderId      int64   `json:"order_id,omitempty,string"`
	Symbol       string  `json:"currency_pair"`
	CreateTime   int64   `json:"create_time,omitempty,string"`
	CreateTimeMs int64   `json:"create_time_ms,omitempty,string"`
	Side         string  `json:"side"`
	Amount       float64 `json:"amount,omitempty,string"`
	Role         string  `json:"role"`
	Price        float64 `json:"price,omitempty,string"`
	Fee          float64 `json:"fee,omitempty,string"`
	PointFee     float64 `json:"point_fee,omitempty,string"`
	GtFee        float64 `json:"gt_fee,omitempty,string"`
	Text         string  `json:"text"`
}
