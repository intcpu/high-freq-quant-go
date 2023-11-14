/*
 * Gate API v4
 *
 * Welcome to Gate.io API  APIv4 provides spot, margin and futures trading operations. There are public APIs to retrieve the real-time market statistics, and private APIs which needs authentication to trade on user's behalf.
 *
 * Contact: support@mail.gate.io
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package gateapi

type CrossMarginCurrency struct {
	// Currency name
	Name string `json:"name,omitempty"`
	// Loan rate
	Rate string `json:"rate,omitempty"`
	// Currency precision
	Prec string `json:"prec,omitempty"`
	// Currency value discount, which is used in total value calculation
	Discount string `json:"discount,omitempty"`
	// Minimum currency borrow amount. Unit is currency itself
	MinBorrowAmount string `json:"min_borrow_amount,omitempty"`
	// Maximum borrow value allowed per user, in USDT
	UserMaxBorrowAmount string `json:"user_max_borrow_amount,omitempty"`
	// Maximum borrow value allowed for this currency, in USDT
	TotalMaxBorrowAmount string `json:"total_max_borrow_amount,omitempty"`
	// Price change between this currency and USDT
	Price string `json:"price,omitempty"`
}
