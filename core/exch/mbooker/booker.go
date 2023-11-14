package mbooker

import (
	"context"
	"strings"
	"sync"

	"high-freq-quant-go/core/exch"

	"high-freq-quant-go/adapter/convert"

	"high-freq-quant-go/adapter/text"

	"high-freq-quant-go/adapter/sort"
)

type Booker struct {
	Name, Symbol     string
	Exname, Extype   string
	asks, bids       []float64
	askBook, bidBook map[float64]float64
	IsReady          bool
	UpdateID         int64
	ResponTime       int64
	UpdateTime       int64
	rw               sync.RWMutex
}

func NewBooker(ctx context.Context) *Booker {
	bk := Booker{
		Symbol:  strings.ToUpper(text.GetString(ctx, exch.CtxSymbol)),
		Exname:  strings.ToLower(text.GetString(ctx, exch.CtxExname)),
		Extype:  strings.ToLower(text.GetString(ctx, exch.CtxExtype)),
		IsReady: false,
	}
	bk.Name = bk.Exname + "_" + bk.Extype + "_" + bk.Symbol
	return &bk
}

func (bk *Booker) SetBook(AskData, BidData map[string]float64) {
	var asks, bids []float64
	askBook, bidBook := map[float64]float64{}, map[float64]float64{}

	var wg sync.WaitGroup
	wg.Add(2)
	go func(asks *[]float64, askBook map[float64]float64, AskData map[string]float64, wg *sync.WaitGroup) {
		defer wg.Done()
		for p, s := range AskData {
			fp := convert.GetFloat64(p)
			if fp == 0 || s == 0 {
				continue
			}
			*asks = append(*asks, fp)
			askBook[fp] = s
		}
	}(&asks, askBook, AskData, &wg)
	go func(bids *[]float64, bidBook map[float64]float64, BidData map[string]float64, wg *sync.WaitGroup) {
		defer wg.Done()
		for p, s := range BidData {
			fp := convert.GetFloat64(p)
			if fp == 0 || s == 0 {
				continue
			}
			*bids = append(*bids, fp)
			bidBook[fp] = s
		}
	}(&bids, bidBook, BidData, &wg)
	wg.Wait()

	sort.SortSlice(asks, 0, len(asks)-1)
	sort.RSortSlice(bids, 0, len(bids)-1)
	bk.rw.Lock()
	defer bk.rw.Unlock()
	bk.asks, bk.bids, bk.askBook, bk.bidBook = asks, bids, askBook, bidBook
}

func (bk *Booker) UpdateAsk(p string, size float64) {
	bk.rw.Lock()
	defer bk.rw.Unlock()
	price := convert.GetFloat64(p)
	isIn := false
	if _, ok := bk.askBook[price]; ok {
		isIn = true
	}
	if size == 0 && !isIn {
		return
	}
	if size == 0 && isIn {
		bk.asks = sort.DelSliceVal(bk.asks, price)
		delete(bk.askBook, price)
		return
	}
	if !isIn {
		bk.asks = sort.AddSortSliceVal(bk.asks, price)
	}
	bk.askBook[price] = size
}

func (bk *Booker) UpdateBid(p string, size float64) {
	bk.rw.Lock()
	defer bk.rw.Unlock()
	price := convert.GetFloat64(p)
	isIn := false
	if _, ok := bk.bidBook[price]; ok {
		isIn = true
	}
	if size == 0 && !isIn {
		return
	}
	if size == 0 && isIn {
		bk.bids = sort.DelSliceVal(bk.bids, price)
		delete(bk.bidBook, price)
		return
	}
	if !isIn {
		bk.bids = sort.AddRSortSliceVal(bk.bids, price)
	}
	bk.bidBook[price] = size
}

func (bk *Booker) GetAsks() []float64 {
	bk.rw.RLock()
	defer bk.rw.RUnlock()
	asks := make([]float64, len(bk.asks))
	copy(asks, bk.asks)
	return bk.asks
}

func (bk *Booker) GetBids() []float64 {
	bk.rw.RLock()
	defer bk.rw.RUnlock()
	bids := make([]float64, len(bk.bids))
	copy(bids, bk.bids)
	return bids
}

func (bk *Booker) GetAskBook() map[float64]float64 {
	bk.rw.RLock()
	defer bk.rw.RUnlock()
	askBook := map[float64]float64{}
	for p, s := range bk.askBook {
		askBook[p] = s
	}
	return askBook
}

func (bk *Booker) GetBidBook() map[float64]float64 {
	bk.rw.RLock()
	defer bk.rw.RUnlock()
	bidBook := map[float64]float64{}
	for p, s := range bk.bidBook {
		bidBook[p] = s
	}
	return bidBook
}

func (bk *Booker) GetAskMap() map[string]float64 {
	bk.rw.RLock()
	defer bk.rw.RUnlock()
	askMap := map[string]float64{}
	for p, s := range bk.askBook {
		sp := convert.GetString(p)
		askMap[sp] = s
	}
	return askMap
}

func (bk *Booker) GetBidMap() map[string]float64 {
	bk.rw.RLock()
	defer bk.rw.RUnlock()
	bidMap := map[string]float64{}
	for p, s := range bk.bidBook {
		sp := convert.GetString(p)
		bidMap[sp] = s
	}
	return bidMap
}

func (bk *Booker) GetBook() ([]float64, []float64, map[float64]float64, map[float64]float64) {
	bk.rw.RLock()
	defer bk.rw.RUnlock()
	askBook, bidBook := map[float64]float64{}, map[float64]float64{}
	var wg sync.WaitGroup
	wg.Add(2)
	go func(askBook map[float64]float64, wg *sync.WaitGroup) {
		defer wg.Done()
		for p, s := range bk.askBook {
			askBook[p] = s
		}
	}(askBook, &wg)
	go func(bidMap map[float64]float64, wg *sync.WaitGroup) {
		defer wg.Done()
		for p, s := range bk.bidBook {
			bidBook[p] = s
		}
	}(bidBook, &wg)
	wg.Wait()
	asks := make([]float64, len(bk.asks))
	copy(asks, bk.asks)
	bids := make([]float64, len(bk.bids))
	copy(bids, bk.bids)
	return asks, bids, askBook, bidBook
}
