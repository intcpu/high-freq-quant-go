package client

import "context"

const (
	Id        = "Id"
	WssUrl    = "WssUrl"
	ProxyUrl  = "ProxyUrl"
	Timeout   = "Timeout"
	Keepalive = "keepalive"
	MsgLen    = "MsgLen"

	SendMsg     = "SendMsg"
	IsReconnect = "IsReconnect"
)

type SubscribeData struct {
	Action *func(ctx context.Context) error
	Param  context.Context
}
