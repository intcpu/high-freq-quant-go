package backtest

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"high-freq-quant-go/adapter/sort"

	"high-freq-quant-go/adapter/convert"
)

type OrderBook struct {
	Path      string
	Data      [][]string
	AskPrice  []float64
	BidPrice  []float64
	AskMap    map[float64]float64
	BidMap    map[float64]float64
	LastLine  []string
	LineQueue *chan []string
	RunMs     int64
	NowMs     int64
	IsReset   int
	ResetTime int64
	MaxGear   int
	End       *chan int
}

func NewOrderBook(path string, gear int, ms int64) *OrderBook {
	lq := make(chan []string, 1)
	eq := make(chan int, 1)
	ob := OrderBook{
		Path:      path,
		RunMs:     ms,
		MaxGear:   gear,
		IsReset:   1,
		LineQueue: &lq,
		End:       &eq,
	}
	ob.ResetData()
	go ob.ReadLine()
	return &ob
}

func (ob *OrderBook) ReadLine() {
	f, err := os.Open(ob.Path)
	if err != nil {
		fmt.Println("OrderBook ReadLine open error", err)
		return
	}
	defer f.Close()
	r := csv.NewReader(f)
	for {
		line, err := r.Read()
		if err == io.EOF {
			*ob.End <- 1
			return
		} else if err != nil {
			*ob.End <- 1
			fmt.Println("OrderBook ReadLine read error", err)
			return
		}
		*ob.LineQueue <- line
	}
}

func (ob *OrderBook) Next() {
	var ms int64
	ob.Data = [][]string{}
	if len(ob.LastLine) > 1 {
		ms = int64(convert.GetFloat64(ob.LastLine[1]))
		ob.Data = append(ob.Data, ob.LastLine)
	}
	ob.LastLine = []string{}
	ob.NowMs = ms
	ob.setNextData()
	ob.UpdateOrderBook()
}

func (ob *OrderBook) setNextData() {
	var nextMs int64
	for {
		select {
		case line := <-*ob.LineQueue:
			if len(line) != 6 {
				continue
			}
			ti := int64(convert.GetFloat64(line[1]))
			if ob.NowMs > 0 && ti-ob.NowMs >= ob.RunMs {
				ob.LastLine = line
				return
			}
			if nextMs == 0 {
				ob.NowMs = ti
				nextMs = ti
			}
			if line[5] == "1" && ob.ResetTime != ti {
				ob.ResetData()
				ob.ResetTime = ti
			}
			ob.Data = append(ob.Data, line)
		case <-*ob.End:
			*ob.End <- 1
			return
		default:

		}
	}
}

func (ob *OrderBook) UpdateOrderBook() {
	for _, line := range ob.Data {
		price := convert.GetFloat64(line[2])
		size := convert.GetFloat64(line[3])
		if line[5] == "1" {
			ob.Create(line[4], price, size)
			ob.IsReset = 1
		} else if line[5] == "0" {
			ob.ResetBook()
			ob.Update(line[4], price, size)
		}
	}
}

func (ob *OrderBook) ResetBook() {
	if ob.IsReset == 0 {
		return
	}
	ob.AskSort()
	ob.BidSort()
	if ob.MaxGear > 0 && len(ob.AskPrice) > ob.MaxGear {
		ob.CutAsks(ob.MaxGear)
	}
	if ob.MaxGear > 0 && len(ob.BidPrice) > ob.MaxGear {
		ob.CutBids(ob.MaxGear)
	}
	ob.IsReset = 0
}

func (ob *OrderBook) CutAsks(n int) {
	l := len(ob.AskPrice)
	for i := n; i < l; i++ {
		delete(ob.AskMap, ob.AskPrice[i])
	}
	ob.AskPrice = ob.AskPrice[0:n]
}

func (ob *OrderBook) CutBids(n int) {
	l := len(ob.BidPrice)
	for i := n; i < l; i++ {
		delete(ob.BidMap, ob.BidPrice[i])
	}
	ob.BidPrice = ob.BidPrice[0:n]
}

func (ob *OrderBook) AskSort() {
	sort.SortSlice(ob.AskPrice, 0, len(ob.AskPrice)-1)
}

func (ob *OrderBook) BidSort() {
	sort.RSortSlice(ob.BidPrice, 0, len(ob.BidPrice)-1)
}

func (ob *OrderBook) ResetData() {
	ob.Data = [][]string{}
	ob.AskPrice = []float64{}
	ob.BidPrice = []float64{}
	ob.AskMap = map[float64]float64{}
	ob.BidMap = map[float64]float64{}
}

func (ob *OrderBook) Create(side string, price, size float64) {
	if side == "1" {
		ob.BidPrice = append(ob.BidPrice, price)
		ob.BidMap[price] = size
	} else if side == "2" {
		ob.AskPrice = append(ob.AskPrice, price)
		ob.AskMap[price] = size
	}
}

func (ob *OrderBook) Update(side string, price, size float64) {
	if side == "1" {
		ob.UpdateBid(price, size)
	} else if side == "2" {
		ob.UpdateAsk(price, size)
	}
}

func (ob *OrderBook) UpdateAsk(price, size float64) {
	isIn := 0
	if _, ok := ob.AskMap[price]; ok {
		isIn = 1
	}
	if size == 0 && isIn == 0 {
		return
	}
	if size == 0 && isIn == 1 {
		ob.AskPrice = sort.DelSliceVal(ob.AskPrice, price)
		delete(ob.AskMap, price)
		return
	}
	if isIn == 1 {
		ob.AskMap[price] = size
		return
	}
	ob.AskMap[price] = size
	ob.AskPrice = sort.AddSortSliceVal(ob.AskPrice, price)
}

func (ob *OrderBook) UpdateBid(price, size float64) {
	isIn := 0
	if _, ok := ob.BidMap[price]; ok {
		isIn = 1
	}
	if size == 0 && isIn == 0 {
		return
	}
	if size == 0 && isIn == 1 {
		ob.BidPrice = sort.DelSliceVal(ob.BidPrice, price)
		delete(ob.BidMap, price)
		return
	}
	if isIn == 1 {
		ob.BidMap[price] = size
		return
	}
	ob.BidMap[price] = size
	ob.BidPrice = sort.AddRSortSliceVal(ob.BidPrice, price)
}
