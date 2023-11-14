package timer

import (
	"fmt"
	"time"
)

func MicNow() int64 {
	return time.Now().UnixNano() / 1000000
}

func Now() int64 {
	return time.Now().Unix()
}

func FormatStr(second int64) string {
	var millSecond int64
	base_format := "2006-01-02 15:04:05"
	if second > 1e10 {
		base_format = "2006-01-02 15:04:05.000"
		millSecond = second * 1e6
		second = 0
	}
	strT := time.Unix(second, millSecond).Format(base_format)
	return strT
}

func NowStr(dsep, sep, hsep string) string {
	base_format := fmt.Sprintf("2006%s01%s02%s15%s04%s05", dsep, dsep, sep, hsep, hsep)
	time_str := time.Now().Format(base_format)
	return time_str
}
