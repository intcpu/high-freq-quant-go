package futures_wss

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
)

const (
	GateFuturesU     = "gate"
	UsdtWssUrl       = "wss://fx-ws.gateio.ws/v4/ws/usdt"
	UsdtDeliveWssUrl = "wss://fx-ws.gateio.ws/v4/ws/delivery/usdt"
	BtcWssUrl        = "wss://fx-ws.gateio.ws/v4/ws/btc"
	UsdtHttpUrl      = "https://api.gateio.ws/api/v4/futures/usdt/"
)

const (
	Subscribe          = "subscribe"
	UnSubscribe        = "unsubscribe"
	TypeDepthAll       = "all"
	MsgTypeUpdate      = "update"
	ChannelDepth       = "futures.order_book"
	ChannelTickers     = "futures.tickers"
	ChannelDepthUpdate = "futures.order_book_update"
	ChannelTrade       = "futures.trades"
	ChannelUserTrade   = "futures.usertrades"
	ChannelOrders      = "futures.orders"
	ChannelPositions   = "futures.positions"
	ChannelBalances    = "futures.balances"

	OrderBookNum              = "50"
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
	Channel string            `json:"channel"`
	Event   string            `json:"event"`
	Time    int64             `json:"time"`
	Result  DepthUpdateResult `json:"result"`
}

type DepthUpdateResult struct {
	Time    int64  `json:"t"`
	Symbol  string `json:"s"`
	FirstId int64  `json:"U"`
	LastId  int64  `json:"u"`
	Bids    []Bid  `json:"b"`
	Asks    []Ask  `json:"a"`
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

/**
{
    "id": null,
    "time": 1637119645,
    "channel": "futures.tickers",
    "event": "update",
    "error": null,
    "result": [
        {
            "contract": "SHIB_USDT",
            "last": "0.0000475",
            "change_percentage": "-6.18",
            "funding_rate": "0.0001",
            "mark_price": "0.000047474",
            "index_price": "0.000047471",
            "total_size": "97955109",
            "volume_24h": "239162673",
            "quanto_base_rate": "",
            "funding_rate_indicative": "0.0001",
            "volume_24h_quote": "113602269",
            "volume_24h_settle": "113602269",
            "volume_24h_base": "2391626730000"
        }
    ]
}
*/

type TickersEvent struct {
	Channel string    `json:"channel"`
	Event   string    `json:"event"`
	Time    int64     `json:"time"`
	Result  []Tickers `json:"result"`
}

type Tickers struct {
	Symbol                string  `json:"contract,omitempty" `
	Last                  float64 `json:"last,omitempty,string"  `                   //最后价格
	ChangePercentage      float64 `json:"change_percentage,omitempty,string" `       //变化百分比
	FundingRate           float64 `json:"funding_rate,omitempty,string" `            //资金汇率
	MarkPrice             float64 `json:"mark_price,omitempty,string" `              //标记价
	IndexPrice            float64 `json:"index_price,omitempty,string"  `            // 指数价
	TotalSize             float64 `json:"total_size,omitempty,string" `              //重开仓
	Volume24h             float64 `json:"volume_24h,omitempty,string"  `             //24小时成交量
	QuantoBaseRate        string  `json:"quanto_base_rate,omitempty" `               //量化基准利率
	FundingRateIndicative float64 `json:"funding_rate_indicative,omitempty,string" ` //下阶段资金汇率
	Volume24hQuote        float64 `json:"volume_24h_quote,omitempty,string" `        //24小时报价货币计
	Volume24hSettle       float64 `json:"volume_24h_settle,omitempty,string" `       //24小时结算货币计
	Volume24hBase         float64 `json:"volume_24h_base,omitempty,string" `         //24小时基础货币计
}

/**
{
    "id": null,
    "time": 1637052099,
    "channel": "futures.usertrades",
    "event": "update",
    "error": null,
    "result": [
        {
            "id": "51450052",
            "create_time": 1637052099,
            "create_time_ms": 1637052099044,
            "contract": "BTC_USDT",
            "order_id": "93682328194",
            "size": -1,
            "price": "61022.2",
            "role": "taker"
        }
    ]
}
*/
type UserTradeEvent struct {
	Channel string      `json:"channel"`
	Event   string      `json:"event"`
	Time    int         `json:"time"`
	Result  []UserTrade `json:"result"`
}

type UserTrade struct {
	Symbol       string `json:"contract"`
	Size         int64  `json:"size"`
	Id           string `json:"id"`
	OrderId      string `json:"order_id"`
	Role         string `json:"role"`
	CreateTime   int64  `json:"create_time"`
	CreateTimeMs int64  `json:"create_time_ms"`
	Price        string `json:"price"`
}

/**
{
    "id": null,
    "time": 1637052099,
    "channel": "futures.orders",
    "event": "update",
    "error": null,
    "result": [
        {
            "contract": "BTC_USDT",
            "create_time": 1637052099,
            "create_time_ms": 1637052099044,
            "fill_price": 61022.2,
            "finish_as": "filled",
            "finish_time": 1637052099,
            "finish_time_ms": 1637052099044,
            "iceberg": 0,
            "id": 93682328194,
            "is_close": false,
            "is_liq": false,
            "is_reduce_only": false,
            "left": 0,
            "mkfr": -0.00005,
            "price": 60983.2,
            "refr": 0,
            "refu": 0,
            "size": -1,
            "status": "finished",
            "text": "web",
            "tif": "gtc",
            "tkfr": 0.00048,
            "user": "3567250"
        }
    ]
}
*/
type OrdersEvent struct {
	Channel string   `json:"channel"`
	Event   string   `json:"event"`
	Time    int      `json:"time"`
	Result  []Orders `json:"result"`
}

type Orders struct {
	Symbol       string  `json:"contract"`
	CreateTime   int64   `json:"create_time"`
	CreateTimeMs int64   `json:"create_time_ms"`
	FillPrice    float64 `json:"fill_price"`
	FinishAs     string  `json:"finish_as"`
	FinishTime   int64   `json:"finish_time"`
	FinishTimeMs int64   `json:"finish_time_ms"`
	Iceberg      int64   `json:"iceberg"`
	Id           int     `json:"id"`
	IsLiq        bool    `json:"is_liq"`
	IsClose      bool    `json:"is_close"`
	IsReduceOnly bool    `json:"is_reduce_only"`
	Left         int64   `json:"left"`
	Mkfr         float64 `json:"mkfr"`
	Price        float64 `json:"price"`
	Refr         float64 `json:"refr"`
	Refu         int     `json:"refu"`
	Size         int64   `json:"size"`
	Status       string  `json:"status"`
	Text         string  `json:"text"`
	Tif          string  `json:"tif"`
	Tkfr         float64 `json:"tkfr"`
	User         string  `json:"user"`
}

/**
{
    "id": null,
    "time": 1637052099,
    "channel": "futures.balances",
    "event": "update",
    "error": null,
    "result": [
        {
            "balance": 1000.465894783673,
            "change": -0.0029290656,
            "text": "BTC_USDT:93682328194",
            "time": 1637052099,
            "time_ms": 1637052099044,
            "type": "fee",
            "user": "3567250"
        }
    ]
}
*/

type BalancesEvent struct {
	Channel string     `json:"channel"`
	Event   string     `json:"event"`
	Time    int64      `json:"time"`
	Result  []Balances `json:"result"`
}

type Balances struct {
	Balance float64 `json:"balance"`
	Change  float64 `json:"change"`
	Text    string  `json:"text"`
	Time    int64   `json:"time"`
	TimeMs  int64   `json:"time_ms"`
	Type    string  `json:"type"`
	User    string  `json:"user"`
}

/**
{
    "id": null,
    "time": 1637052099,
    "channel": "futures.positions",
    "event": "update",
    "error": null,
    "result": [
        {
            "contract": "BTC_USDT",
            "cross_leverage_limit": 0,
            "entry_price": 63147.125925925924,
            "historyPnl": -0.1210513226,
            "history_point": 0,
            "last_close_pnl": -0.0131945696,
            "leverage": 5,
            "leverage_max": 100,
            "liq_price": 50936.21,
            "maintenance_rate": 0.005,
            "margin": 28.7587562551,
            "mode": "single",
            "realised_pnl": -8.089672016622,
            "realised_point": 0,
            "risk_limit": 1000000,
            "size": 23,
            "time": 1637052099,
            "time_ms": 1637052099044,
            "user": "3567250"
        }
    ]
}
*/

type PositionsEvent struct {
	Channel string      `json:"channel"`
	Event   string      `json:"event"`
	Time    int64       `json:"time"`
	Result  []Positions `json:"result"`
}

type Positions struct {
	Symbol             string  `json:"contract"`
	CrossLeverageLimit float64 `json:"cross_leverage_limit"`
	EntryPrice         float64 `json:"entry_price"`
	HistoryPnl         float64 `json:"historyPnl"`
	HistoryPoint       float64 `json:"history_point"`
	LastClosePnl       float64 `json:"last_close_pnl"`
	Leverage           int     `json:"leverage"`
	LeverageMax        float64 `json:"leverage_max"`
	LiqPrice           float64 `json:"liq_price"`
	MaintenanceRate    float64 `json:"maintenance_rate"`
	Margin             float64 `json:"margin"`
	Mode               string  `json:"mode"`
	RealisedPnl        float64 `json:"realised_pnl"`
	RealisedPoint      float64 `json:"realised_point"`
	RiskLimit          int64   `json:"risk_limit"`
	Size               int64   `json:"size"`
	Time               int64   `json:"time"`
	TimeMs             int64   `json:"time_ms"`
	User               string  `json:"user"`
}

type TradeEvent struct {
	Channel string  `json:"channel"`
	Event   string  `json:"event"`
	Time    int     `json:"time"`
	Result  []Trade `json:"result"`
}

type Trade struct {
	Id           int64   `json:"id,omitempty" structs:"id,omitempty"`                         //成交记录 ID
	CreateTime   int64   `json:"create_time,omitempty" structs:"create_time,omitempty"`       //成交时间
	CreateTimeMs int64   `json:"create_time_ms,omitempty" structs:"create_time_ms,omitempty"` //成交时间
	Symbol       string  `json:"contract,omitempty" structs:"contract,omitempty"`             //合约标识
	Size         float64 `json:"size,omitempty" structs:"size,omitempty"`                     //成交数量
	Price        string  `json:"price,omitempty" structs:"price,omitempty"`                   //成交价格
}
