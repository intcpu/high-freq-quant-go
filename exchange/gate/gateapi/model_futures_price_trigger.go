/*
 * Gate API v4
 *
 * Welcome to Gate.io API  APIv4 provides spot, margin and futures trading operations. There are public APIs to retrieve the real-time market statistics, and private APIs which needs authentication to trade on user's behalf.
 *
 * Contact: support@mail.gate.io
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package gateapi

type FuturesPriceTrigger struct {
	// How the order will be triggered   - `0`: by price, which means the order will be triggered if price condition is satisfied  - `1`: by price gap, which means the order will be triggered if gap of recent two prices of specified `price_type` are satisfied.  Only `0` is supported currently
	StrategyType int32 `json:"strategy_type,omitempty"`
	// Price type. 0 - latest deal price, 1 - mark price, 2 - index price
	PriceType int32 `json:"price_type,omitempty"`
	// Value of price on price triggered, or price gap on price gap triggered
	Price string `json:"price,omitempty"`
	// Trigger condition type  - `1`: calculated price based on `strategy_type` and `price_type` >= `price` - `2`: calculated price based on `strategy_type` and `price_type` <= `price`
	Rule int32 `json:"rule,omitempty"`
	// How long (in seconds) to wait for the condition to be triggered before cancelling the order.
	Expiration int32 `json:"expiration,omitempty"`
}
