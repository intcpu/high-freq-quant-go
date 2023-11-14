package analyze

import (
	"math"
)

//对比差
func Diff(v []float64, n int) []float64 {
	var diffs []float64
	for i, _ := range v {
		if i < n {
			continue
		}
		diff := math.Abs(v[i] - v[i-n])
		diffs = append(diffs, diff)
	}
	return diffs
}

//移动比
func Shift(v, p []float64) []float64 {
	var shifts []float64
	for i, n := range v {
		shifts = append(shifts, n/p[i])
	}
	return shifts
}

//移动标准差
func RollingStd(p []float64, n int) []float64 {
	var stds []float64
	for i := n; i < len(p); i++ {
		stds = append(stds, Std(p[i-n:i]))
	}
	return stds
}

//移动和
func RollingSum(p []float64, n int) []float64 {
	var stds []float64
	for i := n; i < len(p); i++ {
		stds = append(stds, Sum(p[i-n:i]))
	}
	return stds
}

//平均
func Mean(v []float64) float64 {
	var res float64 = 0
	var n int = len(v)
	for i := 0; i < n; i++ {
		res += v[i]
	}
	return res / float64(n)
}

//方差
func Variance(v []float64) float64 {
	var res float64 = 0
	var m = Mean(v)
	var n int = len(v)
	for i := 0; i < n; i++ {
		res += (v[i] - m) * (v[i] - m)
	}
	return res / float64(n)
}

//标准差
func Std(v []float64) float64 {
	return math.Sqrt(Variance(v))
}

//数值几率
func Scale(v []float64, t float64) float64 {
	var num int
	for _, n := range v {
		if n >= t {
			num++
		}
	}
	l := float64(len(v))
	if l == 0 {
		return -1
	}
	return float64(num) / float64(len(v))
}

//对数
func Logs(v []float64) []float64 {
	var ls []float64
	for _, n := range v {
		ls = append(ls, math.Log(n))
	}
	return ls
}

//平方
func Square(v []float64) []float64 {
	var vs []float64
	for _, n := range v {
		vs = append(vs, n*n)
	}
	return vs
}

//和
func Sum(v []float64) float64 {
	var vs float64
	for _, n := range v {
		vs += n
	}
	return vs
}
