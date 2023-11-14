package request

import (
	"sync"
	"time"
)

type Times struct {
	SecTimes, MinTimes, HourTimes int64
	LastSec, LastMin, LastHour    int64

	lock sync.Mutex
}

func NewTimes() *Times {
	t := Times{}
	return &t
}

func (t *Times) Update() {
	t.lock.Lock()
	defer t.lock.Unlock()
	mec := time.Now().Unix()
	min := (mec / 60) * 60
	hour := (mec / 3600) * 3600
	if mec == t.LastSec {
		t.SecTimes++
	} else {
		t.SecTimes = 1
		t.LastSec = mec
	}
	if min == t.LastMin {
		t.MinTimes++
	} else {
		t.MinTimes = 1
		t.LastMin = min
	}
	if hour == t.LastHour {
		t.HourTimes++
	} else {
		t.HourTimes = 1
		t.LastHour = hour
	}
}
