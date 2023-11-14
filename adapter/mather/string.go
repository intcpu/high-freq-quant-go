package mather

import (
	"strings"

	"high-freq-quant-go/adapter/convert"
)

func GetFloatNum(v interface{}) int {
	var s string
	switch res := v.(type) {
	case string:
		s = res
	default:
		s = convert.GetString(res)
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	sl := strings.Split(s, ".")
	if len(sl) == 1 {
		return 0
	}
	fStr := strings.Split(s, ".")[1]
	floatLen := len(fStr)
	if fStr[floatLen-1] != '1' {
		floatLen--
	}
	return floatLen
}
