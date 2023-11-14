package exch

import "sync"

var AllManage = NewAllManager()

type AllManager struct {
	ex       map[string]*Exchanger
	sc, lock sync.Mutex
}

func NewAllManager() *AllManager {
	ma := &AllManager{
		ex: map[string]*Exchanger{},
	}
	return ma
}

func (ma *AllManager) SetEx(s string, e *Exchanger) {
	ma.lock.Lock()
	defer ma.lock.Unlock()
	ma.ex[s] = e
}

func (ma *AllManager) DelEx(s string) {
	ex := ma.GetEx(s)
	if ex == nil {
		return
	}
	ex.Cancel()
	ma.lock.Lock()
	defer ma.lock.Unlock()
	delete(ma.ex, s)
}

func (ma *AllManager) GetEx(s string) *Exchanger {
	ma.lock.Lock()
	defer ma.lock.Unlock()
	if _, ok := ma.ex[s]; !ok {
		return nil
	}
	e := ma.ex[s]
	return e
}
