package sort

func SortSlice(arr []float64, start, end int) {
	if start < end {
		i, j := start, end
		key := arr[(start+end)/2]
		for i <= j {
			for arr[i] < key {
				i++
			}
			for arr[j] > key {
				j--
			}
			if i <= j {
				arr[i], arr[j] = arr[j], arr[i]
				i++
				j--
			}
		}

		if start < j {
			SortSlice(arr, start, j)
		}
		if end > i {
			SortSlice(arr, i, end)
		}
	}
}

func RSortSlice(arr []float64, start, end int) {
	if start < end {
		i, j := start, end
		key := arr[(start+end)/2]
		for i <= j {
			for arr[i] > key {
				i++
			}
			for arr[j] < key {
				j--
			}
			if i <= j {
				arr[i], arr[j] = arr[j], arr[i]
				i++
				j--
			}
		}

		if start < j {
			RSortSlice(arr, start, j)
		}
		if end > i {
			RSortSlice(arr, i, end)
		}
	}
}

func AddSortSliceVal(prices []float64, price float64) []float64 {
	if len(prices) == 0 {
		prices = append(prices, price)
		return prices
	}
	if price <= prices[0] {
		prices = append([]float64{price}, prices...)
		return prices
	}
	l := len(prices)
	if price >= prices[l-1] {
		prices = append(prices, price)
		return prices
	}
	for i, p := range prices {
		if i+1 < l && p < price && price <= prices[i+1] {
			var ps []float64
			ps = append(ps, prices[:i+1]...)
			ps = append(ps, price)
			ps = append(ps, prices[i+1:]...)
			prices = ps
			break
		}
	}
	return prices
}

func AddRSortSliceVal(prices []float64, price float64) []float64 {
	if len(prices) == 0 {
		prices = append(prices, price)
		return prices
	}
	if price >= prices[0] {
		prices = append([]float64{price}, prices...)
		return prices
	}
	l := len(prices)
	if price <= prices[l-1] {
		prices = append(prices, price)
		return prices
	}
	for i, p := range prices {
		if i+1 < l && p > price && price > prices[i+1] {
			var ps []float64
			ps = append(ps, prices[:i+1]...)
			ps = append(ps, price)
			ps = append(ps, prices[i+1:]...)
			prices = ps
			break
		}
	}
	return prices
}

func DelSliceVal(prices []float64, price float64) []float64 {
	for i, p := range prices {
		if price == p {
			prices = append(prices[:i], prices[i+1:]...)
			break
		}
	}
	return prices
}
