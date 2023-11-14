package exch

import (
	"context"
	"strings"
	"sync"

	"high-freq-quant-go/adapter/convert"
	"high-freq-quant-go/adapter/sort"
	"high-freq-quant-go/adapter/text"
)

type Booker struct {
	Name, Symbol   string
	Exname, Extype string

	asks, bids       []float64
	askBook, bidBook map[float64]float64
	cacheTime        int64

	askMap, bidMap map[string]float64
	IsReady        bool
	UpdateID       int64
	ResponTime     int64
	UpdateTime     int64
	Rw             sync.RWMutex
}

func NewBooker(ctx context.Context) *Booker {
	bk := Booker{
		Symbol:  strings.ToUpper(text.GetString(ctx, CtxSymbol)),
		Exname:  strings.ToLower(text.GetString(ctx, CtxExname)),
		Extype:  strings.ToLower(text.GetString(ctx, CtxExtype)),
		IsReady: false,
	}
	bk.Name = bk.Exname + "_" + bk.Extype + "_" + bk.Symbol
	return &bk
}

func (bk *Booker) SetBook(AskMap, BidMap map[string]float64) {
	bk.Rw.Lock()
	defer bk.Rw.Unlock()
	bk.askMap, bk.bidMap = AskMap, BidMap
}

func (bk *Booker) UpdateAsk(price string, size float64) {
	isIn := false
	if _, ok := bk.askMap[price]; ok {
		isIn = true
	}
	if size == 0 && !isIn {
		return
	}
	if size == 0 && isIn {
		delete(bk.askMap, price)
		return
	}
	bk.askMap[price] = size
}

func (bk *Booker) UpdateBid(price string, size float64) {
	isIn := false
	if _, ok := bk.bidMap[price]; ok {
		isIn = true
	}
	if size == 0 && !isIn {
		return
	}
	if size == 0 && isIn {
		delete(bk.bidMap, price)
		return
	}
	bk.bidMap[price] = size
}

func (bk *Booker) GetAskLen() int {
	bk.Rw.RLock()
	defer bk.Rw.RUnlock()
	if bk.askMap == nil {
		return 0
	}
	return len(bk.askMap)
}

func (bk *Booker) GetBidLen() int {
	bk.Rw.RLock()
	defer bk.Rw.RUnlock()
	if bk.bidMap == nil {
		return 0
	}
	return len(bk.bidMap)
}

func (bk *Booker) GetAskMap() map[string]float64 {
	bk.Rw.RLock()
	defer bk.Rw.RUnlock()
	askMap := map[string]float64{}
	if bk.askMap != nil {
		for p, s := range bk.askMap {
			askMap[p] = s
		}
	}
	return askMap
}

func (bk *Booker) GetBidMap() map[string]float64 {
	bk.Rw.RLock()
	defer bk.Rw.RUnlock()
	bidMap := map[string]float64{}
	if bk.bidMap != nil {
		for p, s := range bk.bidMap {
			bidMap[p] = s
		}
	}
	return bidMap
}

//asks, bids, askBook, bidBook
func (bk *Booker) GetBook() ([]float64, []float64, map[float64]float64, map[float64]float64) {
	asks, bids := []float64{}, []float64{}
	askBook, bidBook := map[float64]float64{}, map[float64]float64{}
	bk.Rw.RLock()
	defer bk.Rw.RUnlock()
	if bk.askMap == nil || bk.bidMap == nil {
		return asks, bids, askBook, bidBook
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func(asks *[]float64, askBook map[float64]float64, wg *sync.WaitGroup) {
		defer wg.Done()
		for p, s := range bk.askMap {
			fp := convert.GetFloat64(p)
			*asks = append(*asks, fp)
			askBook[fp] = s
		}
	}(&asks, askBook, &wg)
	go func(bids *[]float64, bidBook map[float64]float64, wg *sync.WaitGroup) {
		defer wg.Done()
		for p, s := range bk.bidMap {
			fp := convert.GetFloat64(p)
			*bids = append(*bids, fp)
			bidBook[fp] = s
		}
	}(&bids, bidBook, &wg)
	wg.Wait()
	if bk.cacheTime != bk.UpdateTime {
		sort.SortSlice(asks, 0, len(asks)-1)
		sort.RSortSlice(bids, 0, len(bids)-1)
		bk.asks, bk.bids, bk.cacheTime = asks, bids, bk.UpdateTime
	} else {
		asks, bids = bk.asks, bk.bids
	}
	return asks, bids, askBook, bidBook
}
