package futures_wss

import (
	"context"
	"high-freq-quant-go/adapter/timer"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map"

	"high-freq-quant-go/adapter/text"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/log"
	"high-freq-quant-go/exchange/binance/binanceapi"
	"high-freq-quant-go/exchange/binance/futures_api"
	"high-freq-quant-go/exchange/binance/unify"
)

type Futures struct {
	//sign
	Sign string
	//param
	Ctx context.Context
	//return data
	Bookers  cmap.ConcurrentMap
	BaseData cmap.ConcurrentMap

	//wss client
	Client *FuturesClient

	//struct msg chan
	OrderBookQueue *chan *DepthEvent
	BaseDataQueue  *chan interface{}

	//struct msg to public
	BookDataChannel *chan interface{}
	BookMsgChan     *chan interface{}
}

func NewFutures(ctx context.Context) *Futures {
	bq := make(chan *DepthEvent, exch.MsgChannelLen)
	uq := make(chan interface{}, exch.MsgChannelLen)
	sign := text.GetString(ctx, exch.ConnSign)
	cl := &Futures{
		Sign:           sign,
		Ctx:            ctx,
		Bookers:        cmap.New(),
		BaseData:       cmap.New(),
		OrderBookQueue: &bq,
		BaseDataQueue:  &uq,
	}

	cl.SetPubChannel()
	cl.Client = NewFuturesClient(ctx)
	go cl.MsgEvent()
	go cl.OrderBookEvent()
	go cl.BaseDataEvent()
	return cl
}

func (ws *Futures) GetStatus() int {
	return ws.Client.Status
}

func (ws *Futures) GetBaseinfo(ctx context.Context) *exch.BaseInfo {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if res, ok := ws.BaseData.Get(symbol); ok {
		return res.(*exch.BaseInfo)
	}
	return nil
}

func (ws *Futures) GetBook(ctx context.Context) *exch.Booker {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if res, ok := ws.Bookers.Get(symbol); ok {
		return res.(*exch.Booker)
	}
	return nil
}

func (ws *Futures) SetPubChannel() {
	if ret, ok := ws.Ctx.Value(exch.BookDataChannel).(*chan interface{}); ok {
		ws.BookDataChannel = ret
	}
	if ret, ok := ws.Ctx.Value(exch.BookMsgChan).(*chan interface{}); ok {
		ws.BookMsgChan = ret
	}
}

func (ws *Futures) SubOrderBook(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	if symbol == "" {
		return nil
	}
	if _, ok := ws.Bookers.Get(symbol); ok {
		return nil
	}
	books := exch.NewBooker(ws.Ctx)
	ws.Bookers.Set(symbol, books)
	err := ws.Client.OrderBook(ctx)
	return err
}

func (ws *Futures) SubTicker(ctx context.Context) error {
	err := ws.InitTicker(ctx)
	if err != nil {
		log.Errorln(log.Global, ws.Sign, "binance InitTicker error ", err)
		return err
	}
	err = ws.Client.MarkPrice(ctx)
	if err != nil {
		log.Errorln(log.Global, ws.Sign, "binance sub MarkPrice error ", err)
		return err
	}
	err = ws.Client.Ticker(ctx)
	return err
}

func (ws *Futures) InitTicker(ctx context.Context) error {
	symbol := text.GetString(ctx, exch.CtxSymbol)
	res, err := futures_api.NewBinanceApi(ws.Ctx).GetBaseInfo(symbol)
	if err != nil {
		return err
	}
	ws.BaseData.Set(symbol, res)
	return nil
}

func (ws *Futures) MsgEvent() {
	for {
		select {
		case <-ws.Ctx.Done():
			log.Warnln(log.Wss, ws.Sign, " Binance Futures MsgEvent return by done")
			return
		case msg := <-*ws.Client.MsgQueue:
			switch msg.(type) {
			case *DepthEvent: // handle order book update
				//m.DepthUpdateEventHandler(rtype)
				*ws.OrderBookQueue <- msg.(*DepthEvent)
			default:
				*ws.BaseDataQueue <- msg
			}
		}
	}
}

func (ws *Futures) OrderBookEvent() {
	for {
		select {
		case <-ws.Ctx.Done():
			log.Warnln(log.Wss, ws.Sign, " Binance Futures OrderBookEvent return by done")
			return
		case msg := <-*ws.OrderBookQueue:
			//st := time.Now().UnixNano() / 1000000
			symbol := unify.BToSymbol(msg.Symbol)
			var book *exch.Booker
			if booki, ok := ws.Bookers.Get(symbol); !ok {
				log.Warnln(log.Wss, ws.Sign, symbol, "no symbolbook")
				continue
			} else {
				book = booki.(*exch.Booker)
			}
			if book.UpdateID == 0 || (book.IsReady && msg.PU != book.UpdateID) {
				ws.InitOrderbook(symbol)
				if booki, ok := ws.Bookers.Get(symbol); !ok {
					continue
				} else {
					book = booki.(*exch.Booker)
				}
				if ws.BookMsgChan != nil {
					*ws.BookMsgChan <- book
				}
			}
			if book.GetAskLen() == 0 || book.GetBidLen() == 0 {
				log.Warnln(log.Wss, ws.Sign, symbol, "binance init Orderbook error asks bids:", book.GetAskLen(), book.GetBidLen())
				continue
			}
			if msg.UpdateID < book.UpdateID {
				continue
			}
			if !book.IsReady && msg.FirstUpdateID <= book.UpdateID && book.UpdateID <= msg.UpdateID {
				log.Debugf(log.Global, "%s %s binance init Orderbook wss msg is isready id: %d \r\n", ws.Sign, symbol, msg.UpdateID)
				book.IsReady = true
			}
			if !book.IsReady {
				continue
			}
			book.Rw.Lock()
			var wg sync.WaitGroup
			wg.Add(2)
			//todo SHIB price quantity
			go func(result []Bid, book *exch.Booker, wg *sync.WaitGroup) {
				defer wg.Done()
				for _, bid := range result {
					quantity := unify.QuantityToFloat(book.Symbol, bid.Quantity)
					price := unify.PriceToStr(book.Symbol, bid.Price)
					book.UpdateBid(price, quantity)
				}
			}(msg.Bids, book, &wg)
			go func(result []Ask, book *exch.Booker, wg *sync.WaitGroup) {
				defer wg.Done()
				for _, ask := range result {
					quantity := unify.QuantityToFloat(book.Symbol, ask.Quantity)
					price := unify.PriceToStr(book.Symbol, ask.Price)
					book.UpdateAsk(price, quantity)
				}
			}(msg.Asks, book, &wg)
			wg.Wait()
			book.UpdateID = msg.UpdateID
			book.ResponTime = msg.Time
			book.UpdateTime = time.Now().UnixNano() / 1000000
			book.Rw.Unlock()
			if ws.BookMsgChan != nil {
				*ws.BookMsgChan <- msg
			}
			if ws.BookDataChannel != nil {
				*ws.BookDataChannel <- book
			}
		}
	}
}

func (ws *Futures) InitOrderbook(symbol string) {
	book := exch.NewBooker(ws.Ctx)
	ctx := context.WithValue(context.Background(), exch.CtxSymbol, symbol)
	ctx = context.WithValue(ctx, futures_api.OrderBookLimit, InitOrderBookLimit)
	res, err := futures_api.NewBinanceApi(ws.Ctx).GetOrderBook(ctx)
	if err != nil {
		log.Errorln(log.Global, ws.Sign, symbol, "binance InitOrderbook error ", err)
		return
	}
	book.Symbol = symbol
	book.IsReady = false
	askMap, bidMap := map[string]float64{}, map[string]float64{}
	var wg sync.WaitGroup
	wg.Add(2)
	go func(bids []binanceapi.Bid, wg *sync.WaitGroup) {
		defer wg.Done()
		for _, bid := range bids {
			quantity := unify.QuantityToFloat(symbol, bid.Quantity)
			if quantity == 0 {
				continue
			}
			price := unify.PriceToStr(symbol, bid.Price)
			bidMap[price] = quantity
		}
	}(res.Bids, &wg)
	go func(asks []binanceapi.Ask, wg *sync.WaitGroup) {
		defer wg.Done()
		for _, ask := range asks {
			quantity := unify.QuantityToFloat(symbol, ask.Quantity)
			if quantity == 0 {
				continue
			}
			price := unify.PriceToStr(symbol, ask.Price)
			askMap[price] = quantity
		}
	}(res.Asks, &wg)
	wg.Wait()
	book.SetBook(askMap, bidMap)
	book.UpdateID = res.LastUpdateID
	book.ResponTime = res.Time
	book.UpdateTime = timer.MicNow()
	ws.Bookers.Set(symbol, book)
	log.Debugf(log.Wss, "%s %s binace init Orderbook success [ResponTime=%d] [UpdateTime=%d][bookName=%s][lastid=%d] \r\n", ws.Sign, symbol, book.ResponTime, book.UpdateTime, book.Name, book.UpdateID)
}

func (ws *Futures) BaseDataEvent() {
	for {
		select {
		case <-ws.Ctx.Done():
			log.Warnln(log.Wss, ws.Sign, " Binance Futures BaseDataQueue return by done")
			return
		case msg := <-*ws.BaseDataQueue:
			switch msg.(type) {
			case *MarkPriceEvent:
				d := msg.(*MarkPriceEvent)
				symbol := unify.BToSymbol(d.Symbol)
				mprice := unify.PriceToFloat(symbol, d.MarkPrice)
				iprice := unify.PriceToFloat(symbol, d.IndexPrice)
				if infoi, ok := ws.BaseData.Get(symbol); ok {
					info := infoi.(*exch.BaseInfo)
					info.Symbol = symbol
					info.MarkPrice = mprice
					info.IndexPrice = iprice
					info.FundingRate = d.LastFundingRate
					info.FundingNextApply = float64(d.NextFundingTime)
				}
			case *TickerEvent:
				d := msg.(*TickerEvent)
				symbol := unify.BToSymbol(d.Symbol)
				if infoi, ok := ws.BaseData.Get(symbol); ok {
					info := infoi.(*exch.BaseInfo)
					info.Symbol = symbol
					info.ChangeRate = d.PriceChangePercent
					info.DayVolume = d.QuoteVolume
				}
			default:
				log.Errorln(log.Wss, ws.Sign, "------unknow BaseDataQueue msg type----", msg)
			}
		}
	}
}
