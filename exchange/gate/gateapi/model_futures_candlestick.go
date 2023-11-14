/*
 * Gate API v4
 *
 * Welcome to Gate.io API  APIv4 provides spot, margin and futures trading operations. There are public APIs to retrieve the real-time market statistics, and private APIs which needs authentication to trade on user's behalf.
 *
 * Contact: support@mail.gate.io
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package gateapi

// data point in every timestamp
type FuturesCandlestick struct {
	// Unix timestamp in seconds
	T float64 `json:"t,omitempty"`
	// size volume. Only returned if `contract` is not prefixed
	V int64 `json:"v,omitempty"`
	// Close price
	C string `json:"c,omitempty"`
	// Highest price
	H string `json:"h,omitempty"`
	// Lowest price
	L string `json:"l,omitempty"`
	// Open price
	O string `json:"o,omitempty"`
}
