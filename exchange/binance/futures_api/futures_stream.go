package futures_api

import (
	"context"
	"time"
)

const (
	KeyTimeOut = 3000
)

type UserStream struct {
	ListenKey string
	KeyTime   int64
	C         *BinaceFuturesRequest
}

func NewUserStream(ctx context.Context) *UserStream {
	user := &UserStream{
		C: NewBinaceFuturesRequest(ctx),
	}
	return user
}

func (us *UserStream) GetListenKey() string {
	ti := time.Now().Unix()
	if us.ListenKey != "" && ti-us.KeyTime < KeyTimeOut {
		return us.ListenKey
	}
	listenKey, err := us.C.GetClient().NewStartUserStreamService().Do(context.Background())
	if err != nil {
		return ""
	}
	us.KeyTime = ti
	us.ListenKey = listenKey
	return listenKey
}

func (us *UserStream) ResetListenKey() string {
	ti := time.Now().Unix()
	listenKey, err := us.C.GetClient().NewStartUserStreamService().Do(context.Background())
	if err != nil {
		return ""
	}
	us.KeyTime = ti
	us.ListenKey = listenKey
	return listenKey
}

func (us *UserStream) UpdateListenKey() error {
	return us.C.GetClient().NewKeepaliveUserStreamService().ListenKey(us.ListenKey).Do(context.Background())
}

func (us *UserStream) CloseListenKey() error {
	return us.C.GetClient().NewCloseUserStreamService().ListenKey(us.ListenKey).Do(context.Background())
}
