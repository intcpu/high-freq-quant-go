package convert

import "math"

func Ceil(f float64, n int) float64 {
	pow10_n := math.Pow10(n)
	return math.Ceil(f*pow10_n) / pow10_n
}

func Floor(f float64, n int) float64 {
	pow10_n := math.Pow10(n)
	return math.Floor(f*pow10_n) / pow10_n
}

func Round(f float64, n int) float64 {
	pow10_n := math.Pow10(n)
	return math.Round(f*pow10_n) / pow10_n
}
