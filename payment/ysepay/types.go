package ysepay

import "encoding/json"

// Environment selects one of Ysepay's fixed official gateway environments.
type Environment string

const (
	EnvironmentTest       Environment = "test"
	EnvironmentProduction Environment = "production"
)

// PaymentMode is an aggregated cashier channel supported by this package.
type PaymentMode string

const (
	PaymentModeAlipay            PaymentMode = "26"
	PaymentModeWeChatMiniProgram PaymentMode = "29"
)

// BusinessCode must be the product code approved for the payee merchant.
type BusinessCode string

const (
	BusinessCodeD0       BusinessCode = "00510102"
	BusinessCodeStandard BusinessCode = "00510103"
)

// LimitPay controls credit-based payment methods in the cashier.
type LimitPay string

const (
	LimitPayNone               LimitPay = "0"
	LimitPayCreditCard         LimitPay = "1"
	LimitPayHuabei             LimitPay = "2"
	LimitPayHuabeiInstallments LimitPay = "3"
	LimitPayAllCredit          LimitPay = "4"
)

// ResultState is a stable classification while Status keeps Ysepay's raw value.
type ResultState string

const (
	ResultUnknown   ResultState = "unknown"
	ResultPending   ResultState = "pending"
	ResultSucceeded ResultState = "succeeded"
	ResultFailed    ResultState = "failed"
)

type CreateCashierOrderRequest struct {
	OrderID             string
	PayeeMerchantID     string
	BusinessCode        BusinessCode
	ShopDate            string
	AmountFen           int64
	PaymentValidMinutes int
	BackURL             string
	PayMode             PaymentMode
	MessageCode         string
	LimitPay            LimitPay
	Note                string
	Detail              json.RawMessage
	StoreID             string
	BuyerRealName       string
	AllowRepeatPayment  *bool
	FastPay             bool
	MerchantHomeURL     string
	// H5Join and AppType are optional routing values assigned by Ysepay for
	// the payee merchant. They are forwarded unchanged when configured.
	H5Join  string
	AppType string
}

type CreateCashierOrderResult struct {
	RequestID       string
	OrderID         string
	PayeeMerchantID string
	AmountFen       int64
	PayMode         PaymentMode
	PayURL          string
	CashierAppID    string
	EncryptedData   string
	OrderCreatedAt  string
	OrderExpiresAt  string
	// BusinessData is the complete verified and decrypted response object used
	// to launch the WeChat cashier Mini Program.
	BusinessData json.RawMessage
}

type QueryTradeRequest struct {
	SourceMerchantID  string
	PayeeMerchantID   string
	OrderID           string
	TradeSerialNumber string
	ShopDate          string
}

type TradeResult struct {
	RequestID         string
	SystemCode        string
	Status            string
	State             ResultState
	OrderID           string
	TradeSerialNumber string
	PayeeMerchantID   string
	TotalAmountFen    int64
	ReceiptAmountFen  int64
	AccountDate       string
	ResultNote        string
	OpenID            string
}

type RefundRequest struct {
	OriginalOrderID           string
	OriginalTradeSerialNumber string
	ShopDate                  string
	AmountFen                 int64
	Reason                    string
	RefundOrderID             string
	NotifyURL                 string
}

// RefundResult reports synchronous acceptance, not final refund success.
type RefundResult struct {
	Accepted                  bool
	RequestID                 string
	OriginalOrderID           string
	OriginalTradeSerialNumber string
	RefundOrderID             string
	RefundTradeSerialNumber   string
	AmountFen                 int64
	AccountDate               string
}

type QueryRefundRequest struct {
	OriginalOrderID           string
	OriginalTradeSerialNumber string
	RefundOrderID             string
}

type RefundQueryResult struct {
	RequestID                 string
	OriginalOrderID           string
	OriginalTradeSerialNumber string
	RefundOrderID             string
	RefundTradeSerialNumber   string
	Status                    string
	State                     ResultState
	FundsStatus               string
	TotalAmountFen            int64
	RefundAmountFen           int64
	AccountDate               string
	Reason                    string
}

type PaymentNotification struct {
	RequestID                  string
	PayeeMerchantID            string
	OrderID                    string
	TradeSerialNumber          string
	AmountFen                  int64
	SettlementAmountFen        int64
	PaidAmountFen              int64
	Currency                   string
	PaidAt                     string
	ChannelSendSerialNumber    string
	ChannelReceiveSerialNumber string
	BusinessCode               string
}
