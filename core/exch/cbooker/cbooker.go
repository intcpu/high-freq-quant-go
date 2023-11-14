package cbooker

import (
	"context"
	"strings"
	"sync"

	"high-freq-quant-go/core/exch"

	"high-freq-quant-go/adapter/convert"

	"high-freq-quant-go/adapter/text"

	"high-freq-quant-go/adapter/sort"
	cmap "github.com/orcaman/concurrent-map"
)

type Booker struct {
	Name, Symbol   string
	Exname, Extype string
	asks, bids     []float64
	askMap, bidMap *cmap.ConcurrentMap
	IsReady        bool
	UpdateID       int64
	ResponTime     int64
	UpdateTime     int64
	rw             sync.RWMutex
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
	askMap, bidMap := cmap.New(), cmap.New()

	var wg sync.WaitGroup
	wg.Add(2)
	go func(asks *[]float64, askMap *cmap.ConcurrentMap, AskData map[string]float64, wg *sync.WaitGroup) {
		defer wg.Done()
		for p, s := range AskData {
			fp := convert.GetFloat64(p)
			if fp == 0 || s == 0 {
				continue
			}
			*asks = append(*asks, fp)
			askMap.Set(p, s)
		}
	}(&asks, &askMap, AskData, &wg)
	go func(bids *[]float64, bidMap *cmap.ConcurrentMap, BidData map[string]float64, wg *sync.WaitGroup) {
		defer wg.Done()
		for p, s := range BidData {
			fp := convert.GetFloat64(p)
			if fp == 0 || s == 0 {
				continue
			}
			*bids = append(*bids, fp)
			bidMap.Set(p, s)
		}
	}(&bids, &bidMap, BidData, &wg)
	wg.Wait()

	sort.SortSlice(asks, 0, len(asks)-1)
	sort.RSortSlice(bids, 0, len(bids)-1)
	bk.rw.Lock()
	defer bk.rw.Unlock()
	bk.asks, bk.bids, bk.askMap, bk.bidMap = asks, bids, &askMap, &bidMap
}

func (bk *Booker) UpdateAsk(price string, size float64) {
	p := convert.GetFloat64(price)
	bk.rw.Lock()
	defer bk.rw.Unlock()
	isIn := false
	if _, ok := bk.askMap.Get(price); ok {
		isIn = true
	}
	if size == 0 && !isIn {
		return
	}
	if size == 0 && isIn {
		bk.asks = sort.DelSliceVal(bk.asks, p)
		bk.askMap.Remove(price)
		return
	}
	if !isIn {
		bk.asks = sort.AddSortSliceVal(bk.asks, p)
	}
	bk.askMap.Set(price, size)
}

func (bk *Booker) UpdateBid(price string, size float64) {
	p := convert.GetFloat64(price)
	bk.rw.Lock()
	defer bk.rw.Unlock()
	isIn := false
	if _, ok := bk.bidMap.Get(price); ok {
		isIn = true
	}
	if size == 0 && !isIn {
		return
	}
	if size == 0 && isIn {
		bk.bids = sort.DelSliceVal(bk.bids, p)
		bk.bidMap.Remove(price)
		return
	}
	if !isIn {
		bk.bids = sort.AddRSortSliceVal(bk.bids, p)
	}
	bk.bidMap.Set(price, size)
}

func (bk *Booker) GetAsks() []float64 {
	bk.rw.RLock()
	defer bk.rw.RUnlock()
	asks := make([]float64, len(bk.asks))
	copy(asks, bk.asks)
	return asks
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
	for p, s := range bk.askMap.Items() {
		sp := convert.GetFloat64(p)
		askBook[sp] = s.(float64)
	}
	return askBook
}

func (bk *Booker) GetBidBook() map[float64]float64 {
	bk.rw.RLock()
	defer bk.rw.RUnlock()
	bidBook := map[float64]float64{}
	for p, s := range bk.bidMap.Items() {
		sp := convert.GetFloat64(p)
		bidBook[sp] = s.(float64)
	}
	return bidBook
}

func (bk *Booker) GetAskMap() map[string]float64 {
	bk.rw.RLock()
	defer bk.rw.RUnlock()
	askMap := map[string]float64{}
	for p, s := range bk.askMap.Items() {
		sp := convert.GetString(p)
		askMap[sp] = s.(float64)
	}
	return askMap
}

func (bk *Booker) GetBidMap() map[string]float64 {
	bk.rw.RLock()
	defer bk.rw.RUnlock()
	bidMap := map[string]float64{}
	for p, s := range bk.bidMap.Items() {
		sp := convert.GetString(p)
		bidMap[sp] = s.(float64)
	}
	return bidMap
}

func (bk *Booker) GetBook() ([]float64, []float64, map[float64]float64, map[float64]float64) {
	bk.rw.RLock()
	defer bk.rw.RUnlock()

	askBook, bidBook := map[float64]float64{}, map[float64]float64{}
	var wg sync.WaitGroup
	wg.Add(2)
	go func(askBook map[float64]float64, askMap *cmap.ConcurrentMap, wg *sync.WaitGroup) {
		defer wg.Done()
		for p, s := range askMap.Items() {
			sp := convert.GetFloat64(p)
			askBook[sp] = s.(float64)
		}
	}(askBook, bk.askMap, &wg)
	go func(bidBook map[float64]float64, bidMap *cmap.ConcurrentMap, wg *sync.WaitGroup) {
		defer wg.Done()
		for p, s := range bidMap.Items() {
			sp := convert.GetFloat64(p)
			bidBook[sp] = s.(float64)
		}
	}(bidBook, bk.bidMap, &wg)
	wg.Wait()
	asks := make([]float64, len(bk.asks))
	copy(asks, bk.asks)
	bids := make([]float64, len(bk.bids))
	copy(bids, bk.bids)
	return asks, bids, askBook, bidBook
}
