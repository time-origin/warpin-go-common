package ysepay

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxResponseBytes = 1 << 20

type Client struct {
	environment         Environment
	initiatorMerchantID string
	privateKey          *rsa.PrivateKey
	ysePublicKey        *rsa.PublicKey
	httpClient          *http.Client
	now                 func() time.Time
}

type operationSpec struct {
	name          string
	method        string
	version       string
	testURL       string
	productionURL string
	successCode   string
}

var (
	createOrderOperation = operationSpec{
		name: "create cashier order", method: "order.createOrder", version: "1.4", successCode: "0",
		testURL: "https://appdev.ysepay.com/openapi/order/createOrder", productionURL: "https://ysgate.ysepay.com/openapi/order/createOrder",
	}
	queryTradeOperation = operationSpec{
		name: "query trade", method: "unify.online.trade.order.query", version: "1.1", successCode: "0000",
		testURL: "https://appdev.ysepay-test.com/openapi/unify/online/trade/order/query", productionURL: "https://ysgate.ysepay.com/openapi/unify/online/trade/order/query",
	}
	refundOperation = operationSpec{
		name: "refund", method: "unify.trade.refund", version: "1.0", successCode: "0000",
		testURL: "https://appdev.ysepay.com/openapi/unify/trade/refund", productionURL: "https://ysgate.ysepay.com/openapi/unify/trade/refund",
	}
	queryRefundOperation = operationSpec{
		name: "query refund", method: "unify.trade.refund.query", version: "1.1", successCode: "0000",
		testURL: "https://appdev.ysepay-test.com/openapi/unify/trade/refund/query", productionURL: "https://ysgate.ysepay.com/openapi/unify/trade/refund/query",
	}
)

func (spec operationSpec) endpoint(environment Environment) string {
	if environment == EnvironmentProduction {
		return spec.productionURL
	}
	return spec.testURL
}

type createOrderBusiness struct {
	OrderID             string          `json:"orderId"`
	MessageCode         string          `json:"msgCode"`
	PayeeMerchantID     string          `json:"mercId"`
	BusinessCode        string          `json:"busiCode"`
	ShopDate            string          `json:"shopDate"`
	Amount              string          `json:"amount"`
	PaymentValidMinutes string          `json:"paymentValidTime"`
	Currency            string          `json:"currency"`
	Note                string          `json:"note,omitempty"`
	BackURL             string          `json:"backUrl"`
	LimitPay            string          `json:"limitPay,omitempty"`
	PayMode             string          `json:"payMode"`
	Detail              json.RawMessage `json:"detail,omitempty"`
	StoreID             string          `json:"storeId,omitempty"`
	BuyerRealName       string          `json:"buyerRealname,omitempty"`
	RepeatPayment       string          `json:"isRepeatPay,omitempty"`
	FastPay             string          `json:"isFastPay,omitempty"`
	MerchantHomeURL     string          `json:"mercHomeUrl,omitempty"`
}

func (c *Client) CreateCashierOrder(ctx context.Context, request CreateCashierOrderRequest) (CreateCashierOrderResult, error) {
	if err := validateCreateRequest(request); err != nil {
		return CreateCashierOrderResult{}, err
	}
	amount := strconv.FormatInt(request.AmountFen, 10)
	messageCode := request.MessageCode
	if messageCode == "" {
		messageCode = "S3001"
	}
	business := createOrderBusiness{
		OrderID: request.OrderID, MessageCode: messageCode, PayeeMerchantID: request.PayeeMerchantID,
		BusinessCode: string(request.BusinessCode), ShopDate: request.ShopDate, Amount: amount,
		PaymentValidMinutes: strconv.Itoa(request.PaymentValidMinutes), Currency: "CNY", Note: request.Note,
		BackURL: request.BackURL, LimitPay: string(request.LimitPay), PayMode: string(request.PayMode),
		Detail: request.Detail, StoreID: request.StoreID, BuyerRealName: request.BuyerRealName,
		MerchantHomeURL: request.MerchantHomeURL,
	}
	if request.AllowRepeatPayment != nil {
		if *request.AllowRepeatPayment {
			business.RepeatPayment = "1"
		} else {
			business.RepeatPayment = "0"
		}
	}
	if request.FastPay {
		business.FastPay = "01"
	}
	raw, values, err := c.execute(ctx, createOrderOperation, business)
	if err != nil {
		return CreateCashierOrderResult{}, err
	}
	data := mapValue(values, "data")
	if data == nil {
		data = values
	}
	result := CreateCashierOrderResult{
		RequestID: stringValue(data, "reqMsgId"), OrderID: stringValue(data, "orderId"),
		PayeeMerchantID: stringValue(data, "mercId"), PayMode: request.PayMode,
		PayURL: stringValue(data, "payUrl"), CashierAppID: stringValue(data, "appId"),
		EncryptedData: stringValue(data, "encryData"), OrderCreatedAt: stringValue(data, "orderCreateTime"),
		OrderExpiresAt: stringValue(data, "orderEfficientTime"),
	}
	if request.PayMode == PaymentModeWeChatMiniProgram {
		result.BusinessData = raw
	}
	result.AmountFen, err = integerFen(data, "amount")
	if err != nil || result.OrderID != request.OrderID || result.PayeeMerchantID != request.PayeeMerchantID || result.AmountFen != request.AmountFen {
		return CreateCashierOrderResult{}, operationError(createOrderOperation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	if request.PayMode == PaymentModeAlipay && result.PayURL == "" && result.EncryptedData == "" {
		return CreateCashierOrderResult{}, operationError(createOrderOperation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	if request.PayMode == PaymentModeWeChatMiniProgram && (result.EncryptedData == "" || len(result.BusinessData) == 0) {
		return CreateCashierOrderResult{}, operationError(createOrderOperation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	return result, nil
}

func validateCreateRequest(request CreateCashierOrderRequest) error {
	if !validLength(request.OrderID, 1, 32) || strings.TrimSpace(request.PayeeMerchantID) == "" {
		return fmt.Errorf("%w: order ID and payee merchant ID are required", ErrInvalidRequest)
	}
	if request.BusinessCode != BusinessCodeD0 && request.BusinessCode != BusinessCodeStandard {
		return fmt.Errorf("%w: unsupported business code", ErrInvalidRequest)
	}
	if request.PayMode != PaymentModeAlipay && request.PayMode != PaymentModeWeChatMiniProgram {
		return fmt.Errorf("%w: unsupported payment mode", ErrInvalidRequest)
	}
	if request.AmountFen <= 0 || request.AmountFen > 9_999_999_999 || request.PaymentValidMinutes < 1 || request.PaymentValidMinutes > 30 || !validShopDate(request.ShopDate) {
		return fmt.Errorf("%w: invalid amount, validity, or shop date", ErrInvalidRequest)
	}
	if !validCallbackURLs(request.BackURL) || (request.MerchantHomeURL != "" && !validHTTPURL(request.MerchantHomeURL)) {
		return fmt.Errorf("%w: invalid callback or merchant URL", ErrInvalidRequest)
	}
	if request.MessageCode != "" && request.MessageCode != "S3001" && request.MessageCode != "S3002" {
		return fmt.Errorf("%w: unsupported message code", ErrInvalidRequest)
	}
	if request.LimitPay != "" && request.LimitPay != LimitPayNone && request.LimitPay != LimitPayCreditCard && request.LimitPay != LimitPayHuabei && request.LimitPay != LimitPayHuabeiInstallments && request.LimitPay != LimitPayAllCredit {
		return fmt.Errorf("%w: unsupported limitPay", ErrInvalidRequest)
	}
	if request.PayMode == PaymentModeWeChatMiniProgram && request.LimitPay != "" && request.LimitPay != LimitPayNone && request.LimitPay != LimitPayCreditCard {
		return fmt.Errorf("%w: limitPay is unsupported for WeChat", ErrInvalidRequest)
	}
	if len(request.Detail) > maxResponseBytes || (len(request.Detail) > 0 && (!json.Valid(request.Detail) || bytes.TrimSpace(request.Detail)[0] != '{')) {
		return fmt.Errorf("%w: detail must be valid JSON", ErrInvalidRequest)
	}
	return nil
}

func (c *Client) QueryTrade(ctx context.Context, request QueryTradeRequest) (TradeResult, error) {
	if request.OrderID == "" && request.TradeSerialNumber == "" {
		return TradeResult{}, fmt.Errorf("%w: order ID or trade serial number is required", ErrInvalidRequest)
	}
	if request.ShopDate != "" && !validShopDate(request.ShopDate) {
		return TradeResult{}, fmt.Errorf("%w: invalid shop date", ErrInvalidRequest)
	}
	business := map[string]string{}
	addNonEmpty(business, "srcMercId", request.SourceMerchantID)
	addNonEmpty(business, "payeeMercId", request.PayeeMerchantID)
	addNonEmpty(business, "orderId", request.OrderID)
	if request.OrderID == "" {
		addNonEmpty(business, "tradeSn", request.TradeSerialNumber)
	}
	addNonEmpty(business, "shopDate", request.ShopDate)
	_, outer, err := c.execute(ctx, queryTradeOperation, business)
	if err != nil {
		return TradeResult{}, err
	}
	data := mapValue(outer, "data")
	if data == nil {
		return TradeResult{}, operationError(queryTradeOperation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	result := TradeResult{
		RequestID: stringValue(outer, "reqMsgId"), SystemCode: stringValue(outer, "systemCode"),
		Status: stringValue(data, "tradeStatus"), OrderID: stringValue(data, "orderId"),
		TradeSerialNumber: stringValue(data, "tradeSn"), PayeeMerchantID: stringValue(data, "payeeMercId"),
		AccountDate: stringValue(data, "accountDate"), ResultNote: stringValue(data, "resultNote"),
		OpenID: stringValue(data, "openid"),
	}
	result.State = classifyTradeStatus(result.Status)
	result.TotalAmountFen, err = integerFen(data, "totalAmount")
	if err != nil {
		return TradeResult{}, operationError(queryTradeOperation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	if value := stringValue(data, "receiptAmount"); value != "" {
		result.ReceiptAmountFen, err = parsePositiveOrZeroFen(value)
		if err != nil {
			return TradeResult{}, operationError(queryTradeOperation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
		}
	}
	if result.Status == "" || result.OrderID == "" {
		return TradeResult{}, operationError(queryTradeOperation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	return result, nil
}

func (c *Client) Refund(ctx context.Context, request RefundRequest) (RefundResult, error) {
	if err := validateRefundRequest(request); err != nil {
		return RefundResult{}, err
	}
	business := map[string]string{
		"shopDate": request.ShopDate, "refundAmount": strconv.FormatInt(request.AmountFen, 10),
		"refundReason": request.Reason, "refundOrderId": request.RefundOrderID,
	}
	addNonEmpty(business, "origOrderId", request.OriginalOrderID)
	if request.OriginalOrderID == "" {
		addNonEmpty(business, "origTradeSn", request.OriginalTradeSerialNumber)
	}
	addNonEmpty(business, "notifyUrl", request.NotifyURL)
	_, outer, err := c.execute(ctx, refundOperation, business)
	if err != nil {
		return RefundResult{}, err
	}
	data := mapValue(outer, "businessData")
	if data == nil {
		data = outer
	}
	result := RefundResult{
		Accepted: true, RequestID: stringValue(outer, "reqMsgId"),
		OriginalOrderID: stringValue(data, "origOrderId"), OriginalTradeSerialNumber: stringValue(data, "origTradeSn"),
		RefundOrderID: stringValue(data, "refundOrderId"), RefundTradeSerialNumber: stringValue(data, "refundTradeSn"),
		AccountDate: stringValue(data, "accountDate"),
	}
	result.AmountFen, err = integerFen(data, "refundAmount")
	if err != nil || result.RefundOrderID != request.RefundOrderID || result.AmountFen != request.AmountFen {
		return RefundResult{}, operationError(refundOperation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	return result, nil
}

func validateRefundRequest(request RefundRequest) error {
	if request.OriginalOrderID == "" && request.OriginalTradeSerialNumber == "" {
		return fmt.Errorf("%w: original order ID or trade serial number is required", ErrInvalidRequest)
	}
	if !validShopDate(request.ShopDate) || request.AmountFen <= 0 || request.AmountFen > 9_999_999_999 || !validLength(request.Reason, 1, 50) || !validLength(request.RefundOrderID, 1, 32) {
		return fmt.Errorf("%w: invalid refund date, amount, reason, or order ID", ErrInvalidRequest)
	}
	if request.NotifyURL != "" && !validHTTPURL(request.NotifyURL) {
		return fmt.Errorf("%w: invalid refund notification URL", ErrInvalidRequest)
	}
	return nil
}

func (c *Client) QueryRefund(ctx context.Context, request QueryRefundRequest) (RefundQueryResult, error) {
	if !validLength(request.RefundOrderID, 1, 32) {
		return RefundQueryResult{}, fmt.Errorf("%w: refund order ID is required", ErrInvalidRequest)
	}
	business := map[string]string{"refundOrderId": request.RefundOrderID}
	addNonEmpty(business, "origOrderId", request.OriginalOrderID)
	if request.OriginalOrderID == "" {
		addNonEmpty(business, "origTradeSn", request.OriginalTradeSerialNumber)
	}
	_, outer, err := c.execute(ctx, queryRefundOperation, business)
	if err != nil {
		return RefundQueryResult{}, err
	}
	container := mapValue(outer, "businessData")
	if container == nil {
		container = outer
	}
	data := mapValue(container, "data")
	if data == nil {
		data = container
	}
	result := RefundQueryResult{
		RequestID: stringValue(outer, "reqMsgId"), OriginalOrderID: stringValue(data, "origOrderId"),
		OriginalTradeSerialNumber: stringValue(data, "origTradeSn"), RefundOrderID: stringValue(data, "refundOrderId"),
		RefundTradeSerialNumber: stringValue(data, "refundTradeSn"), Status: stringValue(data, "refundState"),
		FundsStatus: stringValue(data, "fundsState"), AccountDate: stringValue(data, "accountDate"),
		Reason: stringValue(data, "refundReason"),
	}
	result.State = classifyRefundStatus(result.Status)
	result.TotalAmountFen, err = integerFen(data, "totalAmount")
	if err != nil {
		return RefundQueryResult{}, operationError(queryRefundOperation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	result.RefundAmountFen, err = integerFen(data, "refundAmount")
	if err != nil || result.RefundOrderID != request.RefundOrderID || result.Status == "" {
		return RefundQueryResult{}, operationError(queryRefundOperation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	return result, nil
}

func (c *Client) execute(ctx context.Context, operation operationSpec, business any) (json.RawMessage, map[string]any, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("%w: context is required", ErrInvalidRequest)
	}
	businessJSON, err := json.Marshal(business)
	if err != nil {
		return nil, nil, operationError(operation.name, ErrInvalidRequest)
	}
	aesKey, err := generateAESKey(rand.Reader)
	if err != nil {
		return nil, nil, operationError(operation.name, err)
	}
	ciphertext, err := encryptAES(businessJSON, aesKey)
	if err != nil {
		return nil, nil, operationError(operation.name, err)
	}
	check, err := encryptAESKey(aesKey, c.ysePublicKey)
	if err != nil {
		return nil, nil, operationError(operation.name, err)
	}
	now := c.now()
	requestID, err := generateRequestID(rand.Reader, now)
	if err != nil {
		return nil, nil, operationError(operation.name, err)
	}
	fields := map[string]string{
		"timeStamp": now.Format("2006-01-02 15:04:05"), "method": operation.method, "charset": "utf-8",
		"check": check, "bizContent": base64.StdEncoding.EncodeToString(ciphertext), "reqId": requestID,
		"certId": c.initiatorMerchantID, "version": operation.version,
	}
	fields["sign"], err = signFields(fields, c.privateKey)
	if err != nil {
		return nil, nil, operationError(operation.name, err)
	}
	requestBody, err := json.Marshal(fields)
	if err != nil {
		return nil, nil, operationError(operation.name, ErrInvalidRequest)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, operation.endpoint(c.environment), bytes.NewReader(requestBody))
	if err != nil {
		return nil, nil, operationError(operation.name, ErrInvalidRequest)
	}
	httpRequest.Header.Set("Content-Type", "application/json; charset=utf-8")
	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, nil, operationError(operation.name, errors.Join(ErrUncertain, ErrTransport))
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, maxResponseBytes+1)
	responseBody, err := io.ReadAll(limited)
	if err != nil || len(responseBody) > maxResponseBytes || response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, nil, operationError(operation.name, errors.Join(ErrUncertain, ErrTransport))
	}
	responseFields, err := decodeResponseFields(responseBody)
	if err != nil {
		return nil, nil, operationError(operation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	if err := verifyFields(responseFields, c.ysePublicKey); err != nil {
		return nil, nil, operationError(operation.name, errors.Join(ErrUncertain, err))
	}
	if responseFields["subCode"] == "" {
		return nil, nil, operationError(operation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	if responseFields["code"] != "00000" || responseFields["subCode"] != operation.successCode {
		return nil, nil, rejectedError(operation.name, responseFields["code"], responseFields["subCode"])
	}
	encodedBusiness := responseFields["businessData"]
	if encodedBusiness == "" {
		return nil, nil, operationError(operation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	encryptedBusiness, err := decodeBase64Ciphertext(encodedBusiness)
	if err != nil {
		return nil, nil, operationError(operation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	plainBusiness, err := decryptAES(encryptedBusiness, aesKey)
	if err != nil {
		return nil, nil, operationError(operation.name, errors.Join(ErrUncertain, err))
	}
	values, err := decodeJSONObject(plainBusiness)
	if err != nil {
		return nil, nil, operationError(operation.name, errors.Join(ErrUncertain, ErrInvalidResponse))
	}
	return append(json.RawMessage(nil), plainBusiness...), values, nil
}

func decodeResponseFields(body []byte) (map[string]string, error) {
	trimmed := bytes.TrimSpace(body)
	if !json.Valid(trimmed) {
		decoded, err := base64.StdEncoding.DecodeString(string(trimmed))
		if err != nil || !json.Valid(decoded) {
			return nil, errors.New("invalid response encoding")
		}
		trimmed = decoded
	}
	var fields map[string]string
	if err := json.Unmarshal(trimmed, &fields); err != nil || fields == nil {
		return nil, errors.New("invalid response object")
	}
	return fields, nil
}

func decodeJSONObject(data []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var values map[string]any
	if err := decoder.Decode(&values); err != nil || values == nil {
		return nil, errors.New("invalid JSON object")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return nil, errors.New("trailing JSON data")
	}
	return values, nil
}

func stringValue(values map[string]any, key string) string {
	switch value := values[key].(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	default:
		return ""
	}
}

func mapValue(values map[string]any, key string) map[string]any {
	value, _ := values[key].(map[string]any)
	return value
}

func integerFen(values map[string]any, key string) (int64, error) {
	value := stringValue(values, key)
	if value == "" {
		return 0, errors.New("amount is missing")
	}
	return parsePositiveOrZeroFen(value)
}

func parsePositiveOrZeroFen(value string) (int64, error) {
	if value == "" || strings.HasPrefix(value, "+") || strings.HasPrefix(value, "-") {
		return 0, errors.New("invalid amount")
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0, errors.New("invalid amount")
	}
	return parsed, nil
}

func validHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" && parsed.User == nil
}

func validCallbackURLs(value string) bool {
	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if part == "" || strings.TrimSpace(part) != part || !validHTTPURL(part) {
			return false
		}
	}
	return true
}

func addNonEmpty(values map[string]string, key, value string) {
	if value != "" {
		values[key] = value
	}
}

func classifyTradeStatus(status string) ResultState {
	switch status {
	case "00":
		return ResultSucceeded
	case "11", "13", "14", "80", "81":
		return ResultPending
	case "50", "93", "95", "97", "98", "99":
		return ResultFailed
	default:
		return ResultUnknown
	}
}

func classifyRefundStatus(status string) ResultState {
	switch status {
	case "00":
		return ResultSucceeded
	case "10":
		return ResultPending
	case "96", "97", "98", "99":
		return ResultFailed
	default:
		return ResultUnknown
	}
}
