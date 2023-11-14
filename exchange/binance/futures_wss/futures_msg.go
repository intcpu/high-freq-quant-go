package futures_wss

import (
	"encoding/json"
	"fmt"
	"regexp"

	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/log"
)

// MsgHandler implement
func ReadPublicMessage(message *[]byte, WsQueue *exch.MsgQueue) {
	msgType := getMsgType(*message)
	symbol := getMsgSymbol(*message)
	switch msgType {
	case TypeDepthUpdate:
		var event DepthMsg
		err := json.Unmarshal(*message, &event)
		if err != nil {
			log.Errorln(log.Global, symbol, " binance TypeDepthUpdate wss msg json decode error ", err)
			return
		}
		*WsQueue <- &event.Data
	case TypeMarkPrice:
		var event MarkPriceMsg
		err := json.Unmarshal(*message, &event)
		if err != nil {
			log.Errorln(log.Global, symbol, " binance TypeMarkPrice wss msg json decode error ", err)
			return
		}
		*WsQueue <- &event.Data
	case TypeTicker:
		var event TickerMsg
		err := json.Unmarshal(*message, &event)
		if err != nil {
			log.Errorln(log.Global, symbol, " binance TypeMarkPrice wss msg json decode error ", err)
			return
		}
		*WsQueue <- &event.Data
	case TypeKline:
		var event KlineMsg
		err := json.Unmarshal(*message, &event)
		if err != nil {
			log.Errorln(log.Global, symbol, " binance TypeKline wss msg json decode error ", err)
			return
		}
		*WsQueue <- &event.Data.Kline
	case TypeAggTrade:
		var event AggTradeMsg
		err := json.Unmarshal(*message, &event)
		if err != nil {
			log.Errorln(log.Global, symbol, " binance TypeAggTrade wss msg json decode error ", err)
			return
		}
		*WsQueue <- &event.Data
	default:
		log.Warnln(log.Global, symbol, "binance public wss Unknown type", msgType, string(*message))
		return
	}
}

func getMsgType(message []byte) string {
	var (
		regexPrefix   = `"data":{"e":"`
		regexSuffix   = `",`
		findEventType = regexp.MustCompile(fmt.Sprintf(`%s([a-zA-Z0-9]*)%s`, regexPrefix, regexSuffix))
	)
	matches := findEventType.FindSubmatch(message)
	if matches == nil || len(matches) != 2 {
		return ""
	}
	return string(matches[1])
}

func getMsgSymbol(message []byte) string {
	var (
		regexPrefix   = `,"s":"`
		regexSuffix   = `",`
		findEventType = regexp.MustCompile(fmt.Sprintf(`%s([a-zA-Z0-9]*)%s`, regexPrefix, regexSuffix))
	)
	matches := findEventType.FindSubmatch(message)
	if matches == nil || len(matches) != 2 {
		return ""
	}
	return string(matches[1])
}
