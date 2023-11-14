package spot_wss

import (
	"encoding/json"
	"fmt"
	"regexp"

	"high-freq-quant-go/core/exch"

	"high-freq-quant-go/core/log"
)

type MsgHandler struct{}

func NewMsgHandler() *MsgHandler {
	msg := &MsgHandler{}
	return msg
}

func (mh *MsgHandler) ReadMessage(message *[]byte, WsQueue *exch.MsgQueue) {
	msgType := mh.getMsgType(*message)
	if msgType == Subscribe || msgType == UnSubscribe {
		log.Debugln(log.Wss, "gate subscribe msg: ", string(*message))
		return
	}
	channel := mh.getMsgChannel(*message)
	switch channel {
	case ChannelDepthUpdate:
		var event DepthUpdateAllEvent
		err := json.Unmarshal(*message, &event)
		if err != nil {
			log.Warnln(log.Wss, " json error", string(*message), err)
			return
		}
		*WsQueue <- &event
	case ChannelTickers:
		var event TickersEvent
		err := json.Unmarshal(*message, &event)
		if err != nil {
			log.Warnln(log.Wss, " json error", string(*message), err)
			return
		}
		*WsQueue <- &event
	case ChannelOrders:
		log.Debugln(log.Wss, "gate ChannelOrders msg", string(*message))
		var event OrdersEvent
		err := json.Unmarshal(*message, &event)
		if err != nil {
			log.Warnln(log.Wss, " json error", string(*message), err)
			return
		}
		*WsQueue <- &event
	case ChannelBalances:
		log.Debugln(log.Wss, "gate ChannelBalances msg", string(*message))
		var event BalancesEvent
		err := json.Unmarshal(*message, &event)
		if err != nil {
			log.Warnln(log.Wss, " json error", string(*message), err)
			return
		}
		*WsQueue <- &event
	case ChannelUserTrade:
		log.Debugln(log.Wss, "gate ChannelUserTrade msg", string(*message))
		var event UserTradeEvent
		err := json.Unmarshal(*message, &event)
		if err != nil {
			log.Warnln(log.Wss, " json error", string(*message), err)
			return
		}
		*WsQueue <- &event
	case ChannelTrade:
		var event TradeEvent
		err := json.Unmarshal(*message, &event)
		if err != nil {
			log.Warnln(log.Wss, " json error", string(*message), err)
			return
		}
		*WsQueue <- &event
	case ChannelDepth:
		var event OrderBookAll
		if msgType == "all" {
			err := json.Unmarshal(*message, &event)
			if err != nil {
				log.Warnln(log.Wss, " json error", string(*message), err)
				return
			}
		} else if msgType == "update" {
			mh.OrderBookUpdateHandler(message, &event)
		}
		*WsQueue <- &event
	default:
		log.Warnln(log.Wss, "Unknown channel", msgType)
		return
	}
}
func (mh *MsgHandler) getMsgChannel(message []byte) string {
	var (
		regexPrefix   = `"channel":"`
		regexSuffix   = `",`
		findEventType = regexp.MustCompile(fmt.Sprintf(`%s([0-9a-zA-Z\.\_]*)%s`, regexPrefix, regexSuffix))
	)
	matches := findEventType.FindSubmatch(message)
	if matches == nil || len(matches) != 2 {
		return ""
	}
	return string(matches[1])
}

func (mh *MsgHandler) getMsgType(message []byte) string {
	var regexPrefix = `"event":"`
	var regexSuffix = `",`
	var findEventType = regexp.MustCompile(fmt.Sprintf(`%s([0-9a-zA-Z]*)%s`, regexPrefix, regexSuffix))
	matches := findEventType.FindSubmatch(message)
	if matches == nil || len(matches) != 2 {
		return ""
	}
	return string(matches[1])
}

func (mh *MsgHandler) OrderBookUpdateHandler(message *[]byte, event *OrderBookAll) {
	var updateEvent OrderBookUpdate
	err := json.Unmarshal(*message, &updateEvent)
	if err != nil {
		return
	}
	for _, d := range updateEvent.Result {
		if d.Quantity > 0 {
			var bid Bid
			bid.Price = d.Price
			bid.Quantity = d.Quantity
			event.Result.Bids = append(event.Result.Bids, bid)
		} else if d.Quantity < 0 {
			var ask Ask
			ask.Price = d.Price
			ask.Quantity = -d.Quantity
			event.Result.Asks = append(event.Result.Asks, ask)
		} else {
			var ask Ask
			ask.Price = d.Price
			ask.Quantity = 0
			event.Result.Asks = append(event.Result.Asks, ask)
			var bid Bid
			bid.Price = d.Price
			bid.Quantity = 0
			event.Result.Bids = append(event.Result.Bids, bid)
		}
	}
	event.Event = updateEvent.Event
}

func getMsgSymbol(message []byte) string {
	var regexPrefix = `,"s":"`
	var regexSuffix = `",`
	var findEventType = regexp.MustCompile(fmt.Sprintf(`%s([0-9a-zA-Z\.\_]*)%s`, regexPrefix, regexSuffix))
	matches := findEventType.FindSubmatch(message)
	if matches == nil || len(matches) != 2 {
		return ""
	}
	return string(matches[1])
}
