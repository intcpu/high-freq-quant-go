package futures_api

import (
	"high-freq-quant-go/core/exch"
	cmap "github.com/orcaman/concurrent-map"
	"sync"
)

var (
	//apisign
	InitApier = &InitApiersType{
		D: cmap.New(),
	}

	//symbol
	InitInfo = &InitInfosType{
		D: cmap.New(),
	}

	//symbol
	InitBalance = &InitBalanceType{
		D: map[string]*exch.Balance{},
	}

	//symbol
	InitPosition = &InitPositionType{
		D: map[string]*exch.Position{},
	}
)

type InitInfosType struct {
	D  cmap.ConcurrentMap
	sc sync.Mutex
}

type InitApiersType struct {
	D  cmap.ConcurrentMap
	sc sync.Mutex
}

type InitBalanceType struct {
	D  map[string]*exch.Balance
	T  int64 //最后更新时间
	sc sync.Mutex
}

type InitPositionType struct {
	D  map[string]*exch.Position
	T  int64 //最后更新时间
	sc sync.Mutex
}
