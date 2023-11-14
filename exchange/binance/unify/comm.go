package unify

import (
	"strconv"
	"strings"

	"high-freq-quant-go/exchange/binance/binanceapi/futures"

	"high-freq-quant-go/adapter/convert"
)

func SymbolToB(symbol string) string {
	baseQuote := strings.SplitN(symbol, "_", 3)
	if len(baseQuote) > 1 && !strings.Contains(baseQuote[0], "USDT") {
		symbol = strings.ToUpper(baseQuote[0] + baseQuote[1])
	}
	if symbol == "SHIBUSDT" {
		symbol = "1000SHIBUSDT"
	}
	if len(baseQuote) == 3 {
		symbol = symbol + "_" + baseQuote[2][2:]
	}
	return strings.ToUpper(symbol)
}

func BToSymbol(symbol string) string {
	symbol = strings.ToUpper(symbol)
	symbols := strings.Split(symbol, "USDT")
	if len(symbols) == 2 {
		symbol = symbols[0] + "_USDT"
	}
	if symbol == "1000SHIB_USDT" {
		symbol = "SHIB_USDT"
	}
	if len(symbols) == 2 {
		exDay := strings.Split(symbols[1], "_")
		if len(exDay) == 2 {
			symbol = symbol + "_20" + exDay[1]
		}
	}
	return symbol
}

func PriceToStr(symbol, price string) string {
	if symbol == "1000SHIBUSDT" || symbol == "SHIB_USDT" {
		p := convert.GetFloat64(price)
		price = strconv.FormatFloat(p/1000, 'f', 9, 64)
	}
	price = FloatZore(price)
	return price
}

func FloatZore(price string) string {
	if strings.Contains(price, ".") {
		price = strings.TrimRight(price, "0")
		price = strings.TrimRight(price, ".")
	}
	return price
}

func PriceToFloat(symbol string, price float64) float64 {
	if symbol == "1000SHIBUSDT" || symbol == "SHIB_USDT" {
		price = price / 1000
	}
	return price
}

func QuantityToFloat(symbol string, quantity string) float64 {
	num := convert.GetFloat64(quantity)
	if symbol == "1000SHIBUSDT" || symbol == "SHIB_USDT" {
		num = 1000 * num
	}
	return num
}

func IsDelive(symbol string) bool {
	baseQuote := strings.SplitN(symbol, "_", 3)
	idDelive := false
	if len(baseQuote) == 3 {
		idDelive = true
	}
	return idDelive
}

func GetOrderTif(str string) futures.TimeInForceType {
	for ft, t := range UnifyOrderType {
		if t == str {
			return ft
		}
	}
	return futures.TimeInForceTypeGTC
}
