package main

import (
	"context"
	"flag"
	"fmt"
	"high-freq-quant-go/adapter/convert"
	"high-freq-quant-go/adapter/timer"
	"high-freq-quant-go/core/config"
	"high-freq-quant-go/core/exch"
	"high-freq-quant-go/core/log"
	_ "high-freq-quant-go/exchange"
	"high-freq-quant-go/strategy/confer"
	"math"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

var (
	marketRate = 0.01
	orderCannelRate = 0.0006
	orderUsdt float64 = 100
)

const (
	HedgeTakerRate = 0.001
	HedgeCloseFeeRate = 0.001
	HedgeCloseRate = 0.001
	HedgeCloseCannelRate = 0.0006

	minHedgeUsdt = 10
	initLv = 5

	orderSleepTime = 100
	hedgeSleepTime = 1
	hedgeCloseTime = 100
	doSleepTime = 1000
	bookTickerTime = 100
	bookTickerLen = 300
	hedgeDiffTime = 150
	selfDiffTime = 120
)

const (
	makerCloseRate = 0.006
	takerCloseRate = 0.004
	closeRate = 0.002
	diffRate = 0.0005
	closeSleepTime = 1000
)

// Bt bookticker
type Bt struct {
	ask, bid float64

	t int64
}

type diff struct {
	ask, bid float64
}

type hedgeBookDiffs struct {
	gbAsk, bgAsk []float64
	gbBid, bgBid []float64

	t int64
}

type selfBookDiffs struct {
	//ask 自己交易所diffTime价差
	gAsk, bAsk []float64
	//bid 自己交易所diffTime价差
	gBid, bBid []float64

	t int64
}

type Exs struct {
	symbol string
	gstatus, bstatus int

	gask, gbid float64
	bask, bbid float64

	bnex *exch.Exchanger
	gtex *exch.Exchanger

	gbooks, bbooks []*Bt

	hds *hedgeBookDiffs
	sds *selfBookDiffs

	gd, bd *diff

	cancelTimes int
	ctx    context.Context
	cancel context.CancelFunc
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var symbol string
	var mrate, crate, ousdt float64

	flag.StringVar(&symbol, "s", "", "symbol")
	flag.Float64Var(&mrate, "r", 0, "marketRate")
	flag.Float64Var(&crate, "c", 0, "orderCannelRate")
	flag.Float64Var(&ousdt, "u", 0, "orderUsdt")
	flag.Parse()

	logcfg := log.Log{
		LogFilePath:  "./logfile/" + symbol + "_" + timer.NowStr("", "", "") + "/",
		Level:        "INFO|WARN|ERROR",
		Output:       "console|file",
		MaxFileSize:  10,
		MaxFileCount: 7,
	}
	log.ConfigLog(logcfg)

	if symbol == "" {
		fmt.Println(fmt.Sprintf("Usage: %s -s <symbol>", os.Args[0]))
		flag.PrintDefaults()
		os.Exit(1)
	}

	if mrate != 0 {
		marketRate = mrate
	}
	if crate != 0 {
		orderCannelRate = crate
	}
	if ousdt != 0 {
		orderUsdt = ousdt
	}

	keys := config.GetKeysConfig()

	ctx, cancel := context.WithCancel(context.Background())
	exs := &Exs{
		symbol: symbol,
		ctx:    ctx,
		cancel: cancel,
	}

	go setGtex(exs, keys)
	go setBnex(exs, keys)
	go setGtBook(exs)
	go setBnBook(exs)
	go setHedgeBookDiffs(exs)
	go setSelfBookDiffs(exs)
	go setDiff(exs)
	go gtexOrder(exs)
	go bnexOrder(exs)
	go HedgePos(exs)
	go HedgeClose(exs)

	// 关闭撤单
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGKILL, syscall.SIGALRM, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGTERM, syscall.SIGINT)

	sgName := <-ch
	log.Errorln(log.Stt, fmt.Sprintf("======kill by [%v]======", sgName))
	exs.cancel()

	for {
		time.Sleep(orderSleepTime * time.Millisecond)
		if exs.cancelTimes < 6 {
			continue
		}
		cannelOrder(exs)
		break
	}
}

func setDiff(exs *Exs) {
	var maxh, maxs float64
	for {
		time.Sleep(10 * time.Millisecond)
		if exs.sds == nil || exs.hds == nil {
			continue
		}
		if exs.gstatus != 1 || exs.bstatus != 1 {
			continue
		}

		var gd, bd diff

		maxh, maxs = 0, 0
		for _, v := range exs.sds.gAsk {
			if v > maxs {
				maxs = v
			}
		}
		for _, v := range exs.hds.gbAsk {
			if v > maxh {
				maxh = v
			}
		}
		gd.ask = maxh + maxs

		maxh, maxs = 0, 0
		for _, v := range exs.sds.gBid {
			if v > maxs {
				maxs = v
			}
		}
		for _, v := range exs.hds.gbBid {
			if v > maxh {
				maxh = v
			}
		}
		gd.bid = maxh + maxs

		//-----------
		maxh, maxs = 0, 0
		for _, v := range exs.sds.bAsk {
			if v > maxs {
				maxs = v
			}
		}
		for _, v := range exs.hds.bgAsk {
			if v > maxh {
				maxh = v
			}
		}
		bd.ask = maxh + maxs
		maxh, maxs = 0, 0
		for _, v := range exs.sds.bBid {
			if v > maxs {
				maxs = v
			}
		}
		for _, v := range exs.hds.bgBid {
			if v > maxh {
				maxh = v
			}
		}
		bd.bid = maxh + maxs

		exs.gd = &gd
		exs.bd = &bd
	}
}

func setHedgeBookDiffs(exs *Exs) {
	var lastgt, lastbt, stime, etime int64
	for {
		time.Sleep(25 * time.Millisecond)
		if exs.gbooks == nil || exs.bbooks == nil {
			continue
		}
		gl := len(exs.gbooks)
		bl := len(exs.bbooks)
		if gl < bookTickerLen || bl < bookTickerLen {
			continue
		}
		if exs.gstatus != 1 || exs.bstatus != 1 {
			continue
		}
		if exs.gbooks[gl-1].t == lastgt && exs.bbooks[bl-1].t == lastbt {
			continue
		}
		lastgt = exs.gbooks[gl-1].t
		lastbt = exs.bbooks[bl-1].t

		stime = exs.gbooks[0].t
		if exs.gbooks[0].t < exs.bbooks[0].t {
			stime = exs.bbooks[0].t
		}
		etime = exs.gbooks[gl-1].t
		if exs.gbooks[gl-1].t < exs.bbooks[bl-1].t {
			etime = exs.bbooks[bl-1].t
		}

		hds := &hedgeBookDiffs{
			t: timer.MicNow(),
		}
		gi, bi := 0, 0
		for i := stime; i <= etime; i += hedgeDiffTime {
			for j := gi; j < gl; j++ {
				if exs.gbooks[j].t >= i {
					break
				}
				gi = j
			}
			for k := bi; k < bl; k++ {
				if exs.bbooks[k].t >= i {
					break
				}
				bi = k
			}

			askDiff := exs.gbooks[gi].ask - exs.bbooks[bi].ask
			if askDiff > 0 {
				hds.gbAsk = append(hds.gbAsk, askDiff)
			} else if askDiff < 0 {
				hds.bgAsk = append(hds.bgAsk, math.Abs(askDiff))
			}

			bidDiff := exs.gbooks[gi].bid - exs.bbooks[bi].bid
			if bidDiff < 0 {
				hds.gbBid = append(hds.gbBid, math.Abs(bidDiff))
			} else if bidDiff > 0 {
				hds.bgBid = append(hds.bgBid, bidDiff)
			}
		}
		exs.hds = hds
	}
}

func setSelfBookDiffs(exs *Exs) {
	var lastBt *Bt
	for {
		time.Sleep(selfDiffTime * time.Millisecond)
		if exs.gbooks == nil || exs.bbooks == nil {
			continue
		}
		gl := len(exs.gbooks)
		bl := len(exs.bbooks)
		if gl < bookTickerLen || bl < bookTickerLen {
			continue
		}
		if exs.gstatus != 1 || exs.bstatus != 1 {
			continue
		}
		sds := &selfBookDiffs{
			t: timer.MicNow(),
		}
		lastBt = &Bt{}
		for _, v := range exs.gbooks {
			if lastBt.t == 0 {
				lastBt = v
				continue
			}
			if v.t-lastBt.t < selfDiffTime {
				continue
			}
			ask := math.Abs(v.ask - lastBt.ask)
			bid := math.Abs(v.bid - lastBt.bid)
			if ask > 0 {
				sds.gAsk = append(sds.gAsk, ask)
			}
			if bid > 0 {
				sds.gBid = append(sds.gBid, bid)
			}
			lastBt = v
		}

		lastBt = &Bt{}
		for _, v := range exs.bbooks {
			if lastBt.t == 0 {
				lastBt = v
				continue
			}
			if v.t-lastBt.t < selfDiffTime {
				continue
			}
			ask := math.Abs(v.ask - lastBt.ask)
			bid := math.Abs(v.bid - lastBt.bid)
			if ask > 0 {
				sds.bAsk = append(sds.bAsk, ask)
			}
			if bid > 0 {
				sds.bBid = append(sds.bBid, bid)
			}
			lastBt = v
		}
		exs.sds = sds
	}

}

func setGtBook(exs *Exs) {
	symbol := exs.symbol
	for {
		time.Sleep(1 * time.Millisecond)
		gtex := exs.gtex
		if gtex == nil {
			continue
		}
		select {
		case <-exs.ctx.Done():
			gtex.Cancel()
			exs.cancelTimes++
			log.Errorln(log.Stt, "setGtBook", symbol, "---Done stop---", exs.cancelTimes)
			return
		default:
		}
		if gtex.Ex.GetStatus() != 1 {
			log.Infoln(log.Stt, "setGtBook", symbol, "gtex status", gtex.Ex.GetStatus())
			exs.gstatus = 0
			time.Sleep(10 * time.Second)
			continue
		}
		exs.gstatus = 1
		gctx := context.WithValue(context.Background(), exch.CtxSymbol, symbol)
		gbook := gtex.Ex.GetOrderBook(gctx)
		if gbook == nil {
			continue
		}
		gasks, gbids, _, _ := gbook.GetBook()
		if len(gasks) <= 0 || len(gbids) <= 0 {
			continue
		}
		exs.gask = gasks[0]
		exs.gbid = gbids[0]

		gbt := &Bt{
			ask: exs.gask,
			bid: exs.gbid,
			t:   gbook.UpdateTime,
		}
		gl := len(exs.gbooks)
		if gl == 0 || gbt.t-exs.gbooks[gl-1].t > bookTickerTime {
			if gl >= bookTickerLen {
				exs.gbooks = exs.gbooks[1:]
			}
			exs.gbooks = append(exs.gbooks, gbt)
		}
		//todo 推送时间间隔判断wss异常
	}
}

func setBnBook(exs *Exs) {
	symbol := exs.symbol
	for {
		time.Sleep(1 * time.Millisecond)
		bnex := exs.bnex
		if bnex == nil {
			continue
		}
		select {
		case <-exs.ctx.Done():
			bnex.Cancel()
			exs.cancelTimes++
			log.Errorln(log.Stt, "setBnBook", symbol, "---Done stop---", exs.cancelTimes)
			return
		default:
		}
		if bnex.Ex.GetStatus() != 1 {
			log.Infoln(log.Stt, "setBnBook", symbol, "bnex status", bnex.Ex.GetStatus())
			exs.bstatus = 0
			time.Sleep(10 * time.Second)
			continue
		}
		exs.bstatus = 1

		bctx := context.WithValue(context.Background(), exch.CtxSymbol, symbol)
		bbook := bnex.Ex.GetOrderBook(bctx)
		if bbook == nil {
			continue
		}
		basks, bbids, _, _ := bbook.GetBook()
		if len(basks) <= 0 || len(bbids) <= 0 {
			continue
		}
		exs.bask = basks[0]
		exs.bbid = bbids[0]

		bbt := &Bt{
			ask: exs.gask,
			bid: exs.gbid,
			t:   bbook.UpdateTime,
		}
		bl := len(exs.bbooks)
		if bl == 0 || bbt.t-exs.bbooks[bl-1].t > bookTickerTime {
			if bl >= bookTickerLen {
				exs.bbooks = exs.bbooks[1:]
			}
			exs.bbooks = append(exs.bbooks, bbt)
		}
	}
}

func setGtex(exs *Exs, keys map[string]config.ApiUser) {
	symbol := exs.symbol
	gtapi := "gate-cd"
	gtkey := keys[gtapi]
	gtkey.ApiSign = gtapi
	_, gtid := confer.GetPubCid(gtkey.ApiSign, symbol)
	gtctx := exch.ApiCtx(&gtkey)
	ex := exch.NewExchanger(gtctx, gtid)
	gtctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
	gtctx = context.WithValue(gtctx, exch.CtxLv, convert.GetString(initLv))
	_ = ex.Ex.SubUserTrade(gtctx)
	_ = ex.Ex.SubOrder(gtctx)
	_ = ex.Ex.SubPosition(gtctx)
	_ = ex.Ex.SubBalance(gtctx)
	_ = ex.Ex.SubOrderBook(gtctx)

	for {
		time.Sleep(1 * time.Second)

		gtpos := ex.Ex.GetPosition(gtctx)
		if gtpos != nil && gtpos.Lv == float64(initLv) {
			log.Errorln(log.Stt, gtapi, symbol, "setGtex GetPosition success")
			exs.gtex = ex
			break
		}

		_, err := ex.Ex.UpdateLeverage(gtctx)
		if err != nil {
			log.Errorln(log.Stt, gtapi, symbol, "setGtex UpdateLeverage set error", initLv)
		}
	}
}

func setBnex(exs *Exs, keys map[string]config.ApiUser) {
	symbol := exs.symbol
	bnapi := "binance-cd"
	bnkey := keys[bnapi]
	bnkey.ApiSign = bnapi
	_, bnid := confer.GetPubCid(bnkey.ApiSign, symbol)
	bnctx := exch.ApiCtx(&bnkey)
	ex := exch.NewExchanger(bnctx, bnid)
	bnctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
	bnctx = context.WithValue(bnctx, exch.CtxLv, convert.GetString(initLv))
	_ = ex.Ex.SubUserTrade(bnctx)
	_ = ex.Ex.SubOrder(bnctx)
	_ = ex.Ex.SubPosition(bnctx)
	_ = ex.Ex.SubBalance(bnctx)
	_ = ex.Ex.SubOrderBook(bnctx)
	for {
		time.Sleep(1 * time.Second)
		bnpos := ex.Ex.GetPosition(bnctx)
		if bnpos != nil && bnpos.Lv == float64(initLv) {
			log.Errorln(log.Stt, bnapi, symbol, "setBnex GetPosition success", bnpos)
			exs.bnex = ex
			break
		}
		_, err := ex.Ex.UpdateLeverage(bnctx)
		if err == nil {
			log.Errorln(log.Stt, bnapi, symbol, "setBnex UpdateLeverage success", bnpos)
			exs.bnex = ex
			break
		} else {
			log.Errorln(log.Stt, bnapi, symbol, "UpdateLeverage set error", initLv)
		}
	}
}

func cannelOrder(exs *Exs) {
	symbol := exs.symbol
	var gdone, bdone, gdoneErr, bdoneErr int
	for {
		gtex := exs.gtex
		bnex := exs.bnex
		if gtex == nil || bnex == nil {
			break
		}
		if gdone == 1 || bdone == 1 {
			break
		}
		if gdone == 1 && bdoneErr > 10 {
			break
		}
		if bdone == 1 && gdoneErr > 10 {
			break
		}
		if bdoneErr > 10 && gdoneErr > 10 {
			break
		}
		ctx := context.WithValue(context.Background(), exch.CtxSymbol, symbol)
		if gdone == 0 {
			log.Infoln(log.Stt, "gtex CannelAllOrder start")
			go func() {
				_, gerr := gtex.Ex.CannelAllOrder(ctx)
				if gerr == nil {
					log.Infoln(log.Stt, "======gtex CannelAllOrder success======")
					gdone = 1
				} else {
					gdoneErr++
					log.Errorln(log.Stt, "gtex CannelAllOrder error", gerr)
				}
			}()
		}
		if bdone == 0 {
			log.Infoln(log.Stt, "bnex CannelAllOrder start")
			go func() {
				_, berr := bnex.Ex.CannelAllOrder(ctx)
				if berr == nil {
					log.Infoln(log.Stt, "======bnex CannelAllOrder success======")
					bdone = 1
				} else {
					bdoneErr++
					log.Errorln(log.Stt, "bnex CannelAllOrder error", berr)
				}
			}()
		}
		time.Sleep(3 * time.Second)
	}
}

// gt 铺单
func gtexOrder(exs *Exs) {
	symbol := exs.symbol
	for {
		time.Sleep(orderSleepTime * time.Millisecond)
		if exs == nil {
			continue
		}
		select {
		case <-exs.ctx.Done():
			exs.cancelTimes++
			log.Errorln(log.Stt, "gtexOrder", symbol, "---Done stop---", exs.cancelTimes)
			return
		default:
		}
		if exs.gtex == nil || exs.bnex == nil {
			continue
		}
		if exs.gd == nil || exs.bd == nil {
			continue
		}
		//log.Infoln(log.Stt, "gtexOrder", symbol)
		gtex := exs.gtex
		bnex := exs.bnex
		if exs.gstatus != 1 || exs.bstatus != 1 {
			time.Sleep(10 * time.Second)
			log.Infoln(log.Stt, "gtexOrder", symbol, "status error: gstatus, bstatus", exs.gstatus, exs.bstatus)
			continue
		}

		//撤单
		ctx := context.WithValue(context.Background(), exch.CtxSymbol, symbol)
		orders := gtex.Ex.GetOrder(ctx)
		pos := gtex.Ex.GetPosition(ctx)
		bpos := bnex.Ex.GetPosition(ctx)

		if orders == nil || pos == nil || bpos == nil {
			continue
		}

		//有仓位
		if math.Abs(pos.Size*pos.Price) >= minHedgeUsdt || math.Abs(bpos.Size*bpos.Price) >= minHedgeUsdt {
			continue
		}

		if len(orders) > 0 {
			if len(orders) != 2 {
				log.Infoln(log.Stt, "gtexOrder", symbol, "bnex CannelAllOrder orders len, pos price size", len(orders), pos.Price, pos.Size)
				ctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
				_, _ = gtex.Ex.CannelAllOrder(ctx)
				time.Sleep(doSleepTime * time.Millisecond)
				continue
			} else {
				aprice := exs.bask*(1+marketRate) + exs.gd.ask
				bprice := exs.bbid*(1-marketRate) - exs.gd.bid
				nochange := true
				for _, o := range orders {
					if o.Size > 0 && (o.Price/bprice-1) > orderCannelRate {
						nochange = false
						log.Infoln(log.Stt, "gtexOrder", symbol, "gtex bid CannelAllOrder o.Price,o.Size,bprice,bask,bbid,exs.gd-bid", o.Price, o.Size, bprice, exs.bask, exs.bbid, exs.gd.bid)
					}
					if o.Size < 0 && (aprice/o.Price-1) > orderCannelRate {
						nochange = false
						log.Infoln(log.Stt, "gtexOrder", symbol, "gtex ask CannelAllOrder o.Price,o.Size,aprice,bask,bbid,exs.gd-ask", o.Price, o.Size, aprice, exs.bask, exs.bbid, exs.gd.ask)
					}
				}
				if nochange {
					continue
				}
				ctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
				_, _ = gtex.Ex.CannelAllOrder(ctx)
				time.Sleep(doSleepTime * time.Millisecond)
				continue
			}
		}

		log.Infoln(log.Stt, "gtexOrder", symbol, "bnex ask:bid", exs.bask, exs.bbid)
		log.Infoln(log.Stt, "gtexOrder", symbol, "gtex ask:bid", exs.gask, exs.gbid)

		//检查资金
		ctx = context.WithValue(context.Background(), exch.CtxBase, "USDT")
		balance := gtex.Ex.GetBalance(ctx)
		log.Infoln(log.Stt, "gtexOrder", symbol, "gtex balance", balance)
		if balance.Avative*initLv < orderUsdt {
			log.Errorln(log.Stt, "gtexOrder", symbol, "gtex balance error:balance.Avative, initLv, orderUsdt", balance.Avative, initLv, orderUsdt)
			continue
		}

		//下单
		aprice := exs.bask*(1+marketRate) + exs.gd.ask
		bprice := exs.bbid*(1-marketRate) - exs.gd.bid
		size := orderUsdt * 2 / (aprice + bprice)
		log.Infoln(log.Stt, "gtexOrder", symbol, "gtex create order aprice,bprice,asize,bsize,gd-ask,gd-bid", aprice, bprice, -size, size, exs.gd.ask, exs.gd.bid)

		var aos []*exch.Order
		aos = append(aos, &exch.Order{Symbol: symbol, Price: aprice, Size: -size, Text: "market"})
		aos = append(aos, &exch.Order{Symbol: symbol, Price: bprice, Size: size, Text: "market"})
		ctx = context.WithValue(context.Background(), exch.CtxOrders, aos)
		_, _ = gtex.Ex.CreateBatchOrder(ctx)
		time.Sleep(doSleepTime * time.Millisecond)
	}
}

// bn铺单
func bnexOrder(exs *Exs) {
	symbol := exs.symbol
	for {
		time.Sleep(orderSleepTime * time.Millisecond)
		if exs == nil {
			continue
		}
		select {
		case <-exs.ctx.Done():
			exs.cancelTimes++
			log.Errorln(log.Stt, "bnexOrder", symbol, "---Done stop---", exs.cancelTimes)
			return
		default:
		}
		if exs.gtex == nil || exs.bnex == nil {
			continue
		}
		if exs.gd == nil || exs.bd == nil {
			continue
		}
		//log.Infoln(log.Stt, "bnexOrder", symbol)
		bnex := exs.bnex
		gtex := exs.gtex
		if exs.gstatus != 1 || exs.bstatus != 1 {
			time.Sleep(10 * time.Second)
			log.Infoln(log.Stt, "bnexOrder", symbol, "status error: gstatus, bstatus", exs.gstatus, exs.bstatus)
			continue
		}

		ctx := context.WithValue(context.Background(), exch.CtxSymbol, symbol)
		orders := bnex.Ex.GetOrder(ctx)
		pos := bnex.Ex.GetPosition(ctx)
		gpos := gtex.Ex.GetPosition(ctx)

		if orders == nil || gpos == nil || pos == nil {
			continue
		}
		if math.Abs(pos.Size*pos.Price) >= minHedgeUsdt || math.Abs(gpos.Size*gpos.Price) >= minHedgeUsdt {
			continue
		}

		if len(orders) > 0 {
			if len(orders) != 2 {
				log.Infoln(log.Stt, "bnexOrder", symbol, "bnex CannelAllOrder orders len, pos price size", len(orders), pos.Price, pos.Size)
				ctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
				_, _ = bnex.Ex.CannelAllOrder(ctx)
				time.Sleep(doSleepTime * time.Millisecond)
				continue
			} else {
				aprice := exs.gask*(1+marketRate) + exs.bd.ask
				bprice := exs.gbid*(1-marketRate) - exs.bd.bid
				nochange := true
				for _, o := range orders {
					if o.Size > 0 && (o.Price/bprice-1) > orderCannelRate {
						log.Infoln(log.Stt, "bnexOrder", symbol, "bnex bid CannelAllOrder o.Price,o.Size,bprice,gask,gbid,bd-bid", o.Price, o.Size, bprice, exs.gask, exs.gbid, exs.bd.bid)
						nochange = false
					}
					if o.Size < 0 && (aprice/o.Price-1) > orderCannelRate {
						log.Infoln(log.Stt, "bnexOrder", symbol, "bnex ask CannelAllOrder o.Price,o.Size,aprice,gask,gbid,bd-ask", o.Price, o.Size, aprice, exs.gask, exs.gbid, exs.bd.ask)
						nochange = false
					}
				}
				if nochange {
					continue
				}
				ctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
				_, _ = bnex.Ex.CannelAllOrder(ctx)
				time.Sleep(doSleepTime * time.Millisecond)
				continue
			}
		}

		log.Infoln(log.Stt, "bnexOrder", symbol, "gtex ask:bid", exs.gask, exs.gbid)
		log.Infoln(log.Stt, "bnexOrder", symbol, "bnex ask:bid", exs.bask, exs.bbid)

		//检查资金
		ctx = context.WithValue(context.Background(), exch.CtxBase, "USDT")
		balance := bnex.Ex.GetBalance(ctx)
		log.Infoln(log.Stt, "bnexOrder", symbol, "bnex balance", balance)
		if balance.Avative*initLv < orderUsdt {
			log.Errorln(log.Stt, "bnexOrder", symbol, "bnex balance error:balance.Avative, initLv, orderUsdt", balance.Avative, initLv, orderUsdt)
			continue
		}

		//下单
		aprice := exs.gask*(1+marketRate) + exs.bd.ask
		bprice := exs.gbid*(1-marketRate) - exs.bd.bid
		size := orderUsdt * 2 / (aprice + bprice)
		log.Infoln(log.Stt, "bnexOrder", symbol, "bnex create order aprice,bprice,asize,bsize,gd-ask,gd-bid ", aprice, bprice, -size, size, exs.bd.ask, exs.bd.bid)

		var aos []*exch.Order
		aos = append(aos, &exch.Order{Symbol: symbol, Price: aprice, Size: -size, Text: "market"})
		aos = append(aos, &exch.Order{Symbol: symbol, Price: bprice, Size: size, Text: "market"})
		ctx = context.WithValue(context.Background(), exch.CtxOrders, aos)
		_, _ = bnex.Ex.CreateBatchOrder(ctx)
		time.Sleep(doSleepTime * time.Millisecond)

		//ctx = context.WithValue(context.Background(), exch.CtxOrder, &exch.Order{Symbol: "BTC_USDT", Id: k})
		//co, _ := bnex.Ex.CannelOrder(ctx)
	}
}

// HedgePos 对冲
// todo 对冲单，单边挂单平仓后，又被对冲吃出对冲单
func HedgePos(exs *Exs) {
	symbol := exs.symbol
	var lastht, gpostime, bpostime int64
	var lastsize float64
	for {
		time.Sleep(hedgeSleepTime * time.Millisecond)
		if exs == nil {
			continue
		}
		select {
		case <-exs.ctx.Done():
			exs.cancelTimes++
			log.Errorln(log.Stt, "HedgePos", symbol, "---Done stop---", exs.cancelTimes)
			return
		default:
		}
		if exs.gtex == nil || exs.bnex == nil {
			continue
		}
		if exs.gd == nil || exs.bd == nil {
			continue
		}
		gtex := exs.gtex
		bnex := exs.bnex
		if exs.gstatus != 1 || exs.bstatus != 1 {
			time.Sleep(10 * time.Second)
			log.Infoln(log.Stt, "HedgePos", symbol, "status error: gstatus, bstatus", exs.gstatus, exs.bstatus)
			continue
		}

		//是否对冲
		ctx := context.WithValue(context.Background(), exch.CtxSymbol, symbol)
		gpos := gtex.Ex.GetPosition(ctx)
		bpos := bnex.Ex.GetPosition(ctx)

		if gpos == nil || bpos == nil {
			continue
		}

		//是否对冲
		naked := bpos.Size + gpos.Size
		if math.Abs(naked*bpos.Price) < minHedgeUsdt {
			continue
		}
		size := -naked

		if lastsize == size && bpos.LastUpdateTime == bpostime && gpos.LastUpdateTime == gpostime {
			gos := gtex.Ex.GetOrder(ctx)
			bos := bnex.Ex.GetOrder(ctx)
			if gos == nil || bos == nil {
				continue
			}
			log.Infoln(log.Stt, "HedgePos", symbol, "gate orders", len(gos), gos, "binance orders", len(bos), bos)
			if len(gos) == 0 && len(bos) == 0 && timer.Now()-lastht < 3 {
				continue
			}
		}

		log.Infoln(log.Stt, "HedgePos", symbol, "start gpos", gpos.Size, "bpos", bpos.Size)

		//撤单
		ctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
		go func() {
			//wg.Add(1)
			_, _ = gtex.Ex.CannelAllOrder(ctx)
		}()
		go func() {
			//wg.Add(1)
			_, _ = bnex.Ex.CannelAllOrder(ctx)
		}()

		log.Infoln(log.Stt, "HedgePos", symbol, "CannelAllOrder done")
		mmt, hmt := gtex, bnex
		mask, mbid := exs.gask, exs.gbid
		hask, hbid := exs.bask, exs.bbid
		if math.Abs(bpos.Size) > math.Abs(gpos.Size) {
			mmt, hmt = bnex, gtex
			mask, mbid = exs.bask, exs.bbid
			hask, hbid = exs.gask, exs.gbid
		}

		log.Infoln(log.Stt, "HedgePos", symbol, "bnex ask:bid", exs.bask, exs.bbid)
		log.Infoln(log.Stt, "HedgePos", symbol, "gtex ask:bid", exs.gask, exs.gbid)
		log.Infoln(log.Stt, "HedgePos", symbol, "gpos,bpos", gpos, bpos)

		var hex *exch.Exchanger
		var hprice float64
		if naked > 0 {
			hex = hmt
			hprice = hbid
			if mbid > hbid*(1-HedgeTakerRate-HedgeCloseFeeRate) {
				hex = mmt
				hprice = mbid
			}
			hprice = hprice*(1-HedgeTakerRate) - math.Max(exs.gd.bid, exs.bd.bid)
		} else {
			hex = hmt
			hprice = hask
			if mask < hask*(1+HedgeTakerRate+HedgeCloseFeeRate) {
				hex = mmt
				hprice = mask
			}
			hprice = hprice*(1+HedgeTakerRate) + math.Max(exs.gd.ask, exs.bd.ask)
		}

		//下单
		log.Infoln(log.Stt, "HedgePos", symbol, "HedgePos before create order lastht,lastsize,hprice,size,gd-ask,gd-bid", lastht, lastsize, hprice, size, exs.gd.ask, exs.gd.bid)
		var aos []*exch.Order
		aos = append(aos, &exch.Order{Symbol: symbol, Price: hprice, Size: size, Text: "hedge"})
		ctx = context.WithValue(context.Background(), exch.CtxOrders, aos)
		_, err := hex.Ex.CreateBatchOrder(ctx)
		if err == nil {
			gpostime = gpos.LastUpdateTime
			bpostime = bpos.LastUpdateTime
			lastsize = size
			lastht = timer.Now()
			log.Infoln(log.Stt, "HedgePos", symbol, "HedgePos success hprice,size,gpostime,bpostime: ", hprice, size, gpostime, bpostime)
			log.Infoln(log.Stt, "HedgePos", symbol, "HedgePos done create order lastht,lastsize,hprice,size,gd-ask,gd-bid", lastht, lastsize, hprice, size, exs.gd.ask, exs.gd.bid)
		}
		time.Sleep(doSleepTime * time.Millisecond)
	}
}

// HedgeClose 对冲平仓
func HedgeClose(exs *Exs) {
	symbol := exs.symbol
	for {
		time.Sleep(hedgeCloseTime * time.Millisecond)
		if exs == nil {
			continue
		}
		select {
		case <-exs.ctx.Done():
			exs.cancelTimes++
			log.Errorln(log.Stt, "HedgeClose", symbol, "---Done stop---", exs.cancelTimes)
			return
		default:
		}
		if exs.gtex == nil || exs.bnex == nil {
			continue
		}
		if exs.gd == nil || exs.bd == nil {
			continue
		}
		//log.Infoln(log.Stt, "gtexOrder", symbol)
		gtex := exs.gtex
		bnex := exs.bnex
		if exs.gstatus != 1 || exs.bstatus != 1 {
			time.Sleep(10 * time.Second)
			log.Infoln(log.Stt, "HedgeClose", symbol, "status error: gstatus, bstatus", exs.gstatus, exs.bstatus)
			continue
		}

		ctx := context.WithValue(context.Background(), exch.CtxSymbol, symbol)
		gpos := gtex.Ex.GetPosition(ctx)
		bpos := bnex.Ex.GetPosition(ctx)

		if gpos == nil || bpos == nil {
			continue
		}

		naked := bpos.Size + gpos.Size
		if math.Abs(naked*bpos.Price) >= minHedgeUsdt {
			continue
		}

		if math.Abs(gpos.Size*gpos.Price) < minHedgeUsdt && math.Abs(bpos.Size*bpos.Price) < minHedgeUsdt {
			continue
		}

		bos := bnex.Ex.GetOrder(ctx)
		gos := gtex.Ex.GetOrder(ctx)

		if bos == nil || gos == nil {
			continue
		}

		if len(bos) > 1 || len(gos) > 1 {
			var wg sync.WaitGroup
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				log.Infoln(log.Stt, "HedgeClose", symbol, "1 bnex CannelAllOrder orders len, pos price size", len(bos), bpos.Price, bpos.Size)
				ctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
				_, _ = bnex.Ex.CannelAllOrder(ctx)
				wg.Done()
			}(&wg)
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				log.Infoln(log.Stt, "HedgeClose", symbol, "2 gtex CannelAllOrder orders len, pos price size", len(gos), gpos.Price, gpos.Size)
				ctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
				_, _ = gtex.Ex.CannelAllOrder(ctx)
				wg.Done()
			}(&wg)
			wg.Wait()
			time.Sleep(doSleepTime * time.Millisecond)
			continue
		}

		log.Infoln(log.Stt, "HedgeClose", symbol, "3 bnex pos price,size,bask,bbid", bpos.Price, bpos.Size, exs.bask, exs.bbid)
		log.Infoln(log.Stt, "HedgeClose", symbol, "4 gtex pos price,size,gask,gbid", gpos.Price, gpos.Size, exs.gask, exs.gbid)

		var gprice, bprice, gsize, bsize float64
		if gpos.Size > 0 {
			posPnl := bpos.Price - gpos.Price
			bookPnl := exs.gbid - exs.bask
			closePnl := gpos.Price * (HedgeCloseFeeRate + HedgeCloseRate)
			if posPnl+bookPnl > closePnl || math.Abs(posPnl+bookPnl) > exs.gd.bid*3 {
				gprice = exs.gbid * (1 - HedgeCloseRate)
				bprice = exs.bask * (1 + HedgeCloseRate)
				log.Infoln(log.Stt, "HedgeClose", symbol, "5 gprice,bprice", gprice, bprice)
			} else {
				bpprice := exs.bbid - (gpos.Price * HedgeCloseFeeRate)
				gpprice := exs.gask + (gpos.Price * HedgeCloseFeeRate)
				if posPnl+bookPnl < 0 {
					bprice = bpprice + (posPnl + bookPnl)
					gprice = gpprice - (posPnl + bookPnl)
					log.Infoln(log.Stt, "HedgeClose", symbol, "6 gprice,bprice", gprice, bprice)
				}
			}
		} else {
			posPnl := gpos.Price - bpos.Price
			bookPnl := exs.bbid - exs.gask
			closePnl := bpos.Price * (HedgeCloseFeeRate + HedgeCloseRate)
			if posPnl+bookPnl > closePnl || math.Abs(posPnl+bookPnl) > exs.bd.bid*3 {
				bprice = exs.bbid * (1 - HedgeCloseRate)
				gprice = exs.gask * (1 + HedgeCloseRate)
				log.Infoln(log.Stt, "HedgeClose", symbol, "7 gprice,bprice", gprice, bprice)
			} else {
				bpprice := exs.bask + (bpos.Price * HedgeCloseFeeRate)
				gpprice := exs.gbid - (bpos.Price * HedgeCloseFeeRate)
				if posPnl+bookPnl < 0 {
					bprice = bpprice - (posPnl + bookPnl)
					gprice = gpprice + (posPnl + bookPnl)
					log.Infoln(log.Stt, "HedgeClose", symbol, "8 gprice,bprice", gprice, bprice)
				}
			}
		}
		gsize = -gpos.Size
		bsize = -bpos.Size

		log.Infoln(log.Stt, "HedgeClose", symbol, "9 gsize,bsize", gsize, bsize)

		var wg sync.WaitGroup
		if len(bos) > 0 {
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				nochange := true
				for _, o := range bos {
					if math.Abs(o.Price/bprice-1) > HedgeCloseCannelRate {
						nochange = false
						log.Infoln(log.Stt, "HedgeClose", symbol, "10 bnex CannelAllOrder o.Price,o.Size,bprice,bask,bbid", o.Price, o.Size, bprice, exs.bask, exs.bbid)
					}
				}
				if nochange {
					wg.Done()
					return
				}
				ctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
				_, _ = bnex.Ex.CannelAllOrder(ctx)
				wg.Done()
			}(&wg)
		} else {
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				log.Infoln(log.Stt, "HedgeClose", symbol, "11 bnex create order  bprice, bsize", bprice, bsize)
				ctx = context.WithValue(context.Background(), exch.CtxOrder, &exch.Order{Symbol: symbol, Price: bprice, Size: bsize})
				_, _ = bnex.Ex.CreateOrder(ctx)
				wg.Done()
			}(&wg)
		}

		if len(gos) > 0 {
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				nochange := true
				for _, o := range gos {
					if math.Abs(o.Price/gprice-1) > HedgeCloseCannelRate {
						nochange = false
						log.Infoln(log.Stt, "HedgeClose", symbol, "12 gtex CannelAllOrder o.Price,o.Size,gprice,gask,gbid", o.Price, o.Size, gprice, exs.gask, exs.gbid)
					}
				}
				if nochange {
					wg.Done()
					return
				}
				ctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
				_, _ = gtex.Ex.CannelAllOrder(ctx)
				wg.Done()
			}(&wg)
		} else {
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				log.Infoln(log.Stt, "HedgeClose", symbol, "13 gtex create order  gprice, gsize", gprice, gsize)
				ctx = context.WithValue(context.Background(), exch.CtxOrder, &exch.Order{Symbol: symbol, Price: gprice, Size: gsize})
				_, _ = gtex.Ex.CreateOrder(ctx)
				wg.Done()
			}(&wg)
		}
		wg.Wait()
		time.Sleep(doSleepTime * time.Millisecond)
	}
}

func gtexClose(exs *Exs) {
	symbol := exs.symbol
	for {
		time.Sleep(closeSleepTime * time.Millisecond)
		if exs == nil || exs.gtex == nil || exs.bnex == nil {
			continue
		}
		gtex := exs.gtex
		if exs.gstatus != 1 || exs.bstatus != 1 {
			log.Infoln(log.Stt, "gtexClose", symbol, "status error: gstatus, bstatus", exs.gstatus, exs.bstatus)
			continue
		}

		ctx := context.WithValue(context.Background(), exch.CtxSymbol, symbol)
		orders := gtex.Ex.GetOrder(ctx)
		pos := gtex.Ex.GetPosition(ctx)

		if pos == nil || orders == nil {
			continue
		}

		if pos.Size == 0 {
			continue
		}
		log.Infoln(log.Stt, "gtexClose", symbol, "gtex pos", pos)
		log.Infoln(log.Stt, "gtexClose", symbol, "bnex ask:bid", exs.bask, exs.bbid)
		log.Infoln(log.Stt, "gtexClose", symbol, "gtex ask:bid", exs.gask, exs.gbid)
		log.Infoln(log.Stt, "gtexClose", symbol, "gtex orders", orders)

		csize := -pos.Size
		var oprice, mprice, tprice, cprice float64
		if csize < 0 {
			mprice = exs.bbid * (1 - makerCloseRate)
			tprice = exs.bbid * (1 - takerCloseRate)
			cprice = exs.bbid * (1 - closeRate)
			if cprice < pos.Price {
				oprice = exs.gbid * (1 - closeRate)
			} else if tprice < pos.Price {
				oprice = exs.gbid
			} else if mprice < pos.Price {
				oprice = pos.Price * (1 + closeRate)
			} else {
				oprice = cprice
			}
		} else {
			mprice = exs.bask * (1 + makerCloseRate)
			tprice = exs.bask * (1 + takerCloseRate)
			cprice = exs.bask * (1 + closeRate)
			if cprice > pos.Price {
				oprice = exs.gask * (1 + closeRate)
			} else if tprice > pos.Price {
				oprice = exs.gbid
			} else if mprice > pos.Price {
				oprice = pos.Price * (1 - closeRate)
			} else {
				oprice = cprice
			}
		}
		log.Infoln(log.Stt, "gtexClose", symbol, "gtex closePos oprice, mprice, tprice, cprice, len(orders)", oprice, mprice, tprice, cprice, len(orders))
		if len(orders) > 1 {
			ctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
			_, _ = gtex.Ex.CannelAllOrder(ctx)
			time.Sleep(doSleepTime * time.Millisecond)
			continue
		} else if len(orders) > 0 {
			nochange := true
			for _, o := range orders {
				pdiff := math.Abs((oprice / o.Price) - 1)
				if o.Size*pos.Size > 0 || pdiff > diffRate {
					log.Infoln(log.Stt, "gtexClose", symbol, "gtex closePos pdiff,posSize,oSize", pdiff, pos.Size, o.Size)
					nochange = false
				}
			}
			if nochange {
				continue
			}
		}
		log.Infoln(log.Stt, "gtexClose", symbol, "gtex closePos order Price Size,pos Price Size ", oprice, csize, pos.Price, pos.Size)
		ctx = context.WithValue(context.Background(), exch.CtxOrder, &exch.Order{Symbol: symbol, Price: oprice, Size: csize, Text: "close"})
		_, _ = gtex.Ex.CreateOrder(ctx)
		time.Sleep(doSleepTime * time.Millisecond)
	}
}

func bnexClose(exs *Exs) {
	symbol := exs.symbol
	for {
		time.Sleep(closeSleepTime * time.Millisecond)
		if exs == nil || exs.gtex == nil || exs.bnex == nil {
			continue
		}
		bnex := exs.bnex
		if exs.gstatus != 1 || exs.bstatus != 1 {
			log.Infoln(log.Stt, "bnexClose", symbol, "status error: gstatus, bstatus", exs.gstatus, exs.bstatus)
			continue
		}

		ctx := context.WithValue(context.Background(), exch.CtxSymbol, symbol)
		orders := bnex.Ex.GetOrder(ctx)
		pos := bnex.Ex.GetPosition(ctx)

		if pos == nil || orders == nil {
			continue
		}


		if pos.Size == 0 {
			continue
		}
		log.Infoln(log.Stt, "bnexClose", symbol, "bnex pos", pos)
		log.Infoln(log.Stt, "bnexClose", symbol, "gtex ask:bid", exs.gask, exs.gbid)
		log.Infoln(log.Stt, "bnexClose", symbol, "bnex ask:bid", exs.bask, exs.bbid)
		log.Infoln(log.Stt, "bnexClose", symbol, "bnex orders", orders)

		csize := -pos.Size
		var oprice, mprice, tprice, cprice float64
		if csize < 0 {
			mprice = exs.gbid * (1 - makerCloseRate)
			tprice = exs.gbid * (1 - takerCloseRate)
			cprice = exs.gbid * (1 - closeRate)
			if cprice < pos.Price {
				oprice = exs.bbid * (1 - closeRate)
			} else if tprice < pos.Price {
				oprice = exs.bask
			} else if mprice < pos.Price {
				oprice = pos.Price * (1 + closeRate)
			} else {
				oprice = cprice
			}
		} else {
			mprice = exs.gask * (1 + makerCloseRate)
			tprice = exs.gask * (1 + takerCloseRate)
			cprice = exs.gask * (1 + closeRate)
			if cprice > pos.Price {
				oprice = exs.bask * (1 + closeRate)
			} else if tprice > pos.Price {
				oprice = exs.bbid
			} else if mprice > pos.Price {
				oprice = pos.Price * (1 - closeRate)
			} else {
				oprice = cprice
			}
		}
		log.Infoln(log.Stt, "bnexClose", symbol, "bnex closePos oprice, mprice, tprice, cprice, len(orders)", oprice, mprice, tprice, cprice, len(orders))
		if len(orders) > 1 {
			ctx = context.WithValue(context.Background(), exch.CtxSymbol, symbol)
			_, _ = bnex.Ex.CannelAllOrder(ctx)
			time.Sleep(doSleepTime * time.Millisecond)
			continue
		} else if len(orders) > 0 {
			nochange := true
			for _, o := range orders {
				pdiff := math.Abs((oprice / o.Price) - 1)
				if o.Size*pos.Size > 0 || pdiff > diffRate {
					log.Infoln(log.Stt, "bnexClose", symbol, "bnex closePos diff,posSize,oSize", pdiff, pos.Size, o.Size)
					nochange = false
				}
			}
			if nochange {
				continue
			}
		}

		log.Infoln(log.Stt, "bnexClose", symbol, "bnex closePos order Price Size, pos Price Size ", oprice, csize, pos.Price, pos.Size)
		ctx = context.WithValue(context.Background(), exch.CtxOrder, &exch.Order{Symbol: symbol, Price: oprice, Size: csize, Text: "close"})
		_, _ = bnex.Ex.CreateOrder(ctx)
		time.Sleep(doSleepTime * time.Millisecond)
	}
}
