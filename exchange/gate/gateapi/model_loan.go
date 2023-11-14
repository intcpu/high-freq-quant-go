/*
 * Gate API v4
 *
 * Welcome to Gate.io API  APIv4 provides spot, margin and futures trading operations. There are public APIs to retrieve the real-time market statistics, and private APIs which needs authentication to trade on user's behalf.
 *
 * Contact: support@mail.gate.io
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package gateapi

// Margin loan details
type Loan struct {
	// Loan ID
	Id string `json:"id,omitempty"`
	// Creation time
	CreateTime string `json:"create_time,omitempty"`
	// Repay time of the loan. No value will be returned for lending loan
	ExpireTime string `json:"expire_time,omitempty"`
	// Loan status  open - not fully loaned loaned - all loaned out for lending loan; loaned in for borrowing side finished - loan is finished, either being all repaid or cancelled by the lender auto_repaid - automatically repaid by the system
	Status string `json:"status,omitempty"`
	// Loan side
	Side string `json:"side"`
	// Loan currency
	Currency string `json:"currency"`
	// Loan rate. Only rates in [0.0002, 0.002] are supported.  Not required in lending. Market rate calculated from recent rates will be used if not set
	Rate string `json:"rate,omitempty"`
	// Loan amount
	Amount string `json:"amount"`
	// Loan days. Only 10 is supported for now
	Days int32 `json:"days,omitempty"`
	// Whether to auto renew the loan upon expiration
	AutoRenew bool `json:"auto_renew,omitempty"`
	// Currency pair. Required if borrowing
	CurrencyPair string `json:"currency_pair,omitempty"`
	// Amount not lent out yet
	Left string `json:"left,omitempty"`
	// Repaid amount
	Repaid string `json:"repaid,omitempty"`
	// Repaid interest
	PaidInterest string `json:"paid_interest,omitempty"`
	// Outstanding interest yet to be paid
	UnpaidInterest string `json:"unpaid_interest,omitempty"`
	// Loan fee rate
	FeeRate string `json:"fee_rate,omitempty"`
	// Original loan ID of the loan if auto-renewed, otherwise equals to id
	OrigId string `json:"orig_id,omitempty"`
	// User defined custom ID
	Text string `json:"text,omitempty"`
}
