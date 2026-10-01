package ysepay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// ParsePaymentNotification verifies and parses one Ysepay notification. It is
// deterministic and has no persistence or acknowledgement side effects.
func (c *Client) ParsePaymentNotification(payload []byte) (PaymentNotification, error) {
	if len(payload) == 0 || len(payload) > maxResponseBytes {
		return PaymentNotification{}, fmt.Errorf("%w: notification size is invalid", ErrInvalidResponse)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	var fields map[string]string
	if err := decoder.Decode(&fields); err != nil || fields == nil {
		return PaymentNotification{}, fmt.Errorf("%w: notification is not a JSON object", ErrInvalidResponse)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return PaymentNotification{}, fmt.Errorf("%w: notification has trailing data", ErrInvalidResponse)
	}
	if err := verifyFields(fields, c.ysePublicKey); err != nil {
		return PaymentNotification{}, err
	}
	if fields["src"] != "pregate" || fields["bizContent"] == "" {
		return PaymentNotification{}, fmt.Errorf("%w: notification envelope is incomplete", ErrInvalidResponse)
	}
	values, err := decodeJSONObject([]byte(fields["bizContent"]))
	if err != nil {
		return PaymentNotification{}, fmt.Errorf("%w: notification business data is invalid", ErrInvalidResponse)
	}
	result := PaymentNotification{
		RequestID: fields["reqId"], PayeeMerchantID: stringValue(values, "mercId"),
		OrderID: stringValue(values, "orderId"), TradeSerialNumber: stringValue(values, "tradeSn"),
		Currency: stringValue(values, "currencyCode"), PaidAt: stringValue(values, "payTime"),
		ChannelSendSerialNumber:    stringValue(values, "channelSendSn"),
		ChannelReceiveSerialNumber: stringValue(values, "channelRecvSn"),
		BusinessCode:               stringValue(values, "busiCode"),
	}
	if result.PayeeMerchantID == "" || result.OrderID == "" || result.TradeSerialNumber == "" || result.Currency != "CNY" || result.PaidAt == "" || result.ChannelSendSerialNumber == "" || result.ChannelReceiveSerialNumber == "" {
		return PaymentNotification{}, fmt.Errorf("%w: notification fields are incomplete", ErrInvalidResponse)
	}
	result.AmountFen, err = YuanToFen(stringValue(values, "amount"))
	if err != nil {
		return PaymentNotification{}, err
	}
	result.SettlementAmountFen, err = YuanToFen(stringValue(values, "settlementAmt"))
	if err != nil {
		return PaymentNotification{}, err
	}
	result.PaidAmountFen, err = YuanToFen(stringValue(values, "payAmt"))
	if err != nil {
		return PaymentNotification{}, err
	}
	return result, nil
}
