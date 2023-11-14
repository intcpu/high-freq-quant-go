package mather

import (
	"fmt"
	"math"
	"strconv"
)

//浮点精度
func FloatRound(f float64, n int) float64 {
	format := "%." + strconv.Itoa(n) + "f"
	res, err := strconv.ParseFloat(fmt.Sprintf(format, f), 64)
	if err != nil {
		res = math.Trunc(f*math.Pow10(n)) * math.Pow10(-1*n)
	}
	return res
}

//浮点向下精度
func FloatFloor(f float64, n int) float64 {
	res := math.Trunc(f*math.Pow10(n)) * math.Pow10(-1*n)
	return res
}

//最小公倍数
func LcmNormal(x, y int) int {
	var top int = x * y
	var i = x
	if x < y {
		i = y
	}
	for ; i <= top; i++ {
		if i%x == 0 && i%y == 0 {
			return i
		}
	}
	return top
}
