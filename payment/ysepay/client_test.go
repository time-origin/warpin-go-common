package ysepay

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestBDD_PAY_001To003_CreateCashierOrderMapsBothIdentitiesAndChannels(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		mode PaymentMode
		want CreateCashierOrderResult
	}{
		{name: "PAY-001 alipay", mode: PaymentModeAlipay, want: CreateCashierOrderResult{PayURL: "alipays://cashier/test"}},
		{name: "PAY-002 wechat", mode: PaymentModeWeChatMiniProgram, want: CreateCashierOrderResult{CashierAppID: "wx-cashier-test"}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client, gateway := newGatewayClient(t, func(method string, business map[string]any) (string, any) {
				if method != "order.createOrder" {
					t.Fatalf("method = %q", method)
				}
				return "0", map[string]any{
					"reqMsgId":           "REQUEST_001",
					"amount":             "101",
					"mercId":             "PAYEE_TEST",
					"orderId":            "ORDER_20260903_001",
					"payUrl":             test.want.PayURL,
					"appId":              test.want.CashierAppID,
					"encryData":          "client-data",
					"orderCreateTime":    "2026-09-03 12:00:00",
					"orderEfficientTime": "2026-09-03 12:30:00",
				}
			})
			got, err := client.CreateCashierOrder(context.Background(), validCreateRequest(test.mode))
			if err != nil {
				t.Fatalf("CreateCashierOrder() error = %v", err)
			}
			if got.PayMode != test.mode || got.PayURL != test.want.PayURL || got.CashierAppID != test.want.CashierAppID {
				t.Fatalf("CreateCashierOrder() = %+v", got)
			}
			if got.AmountFen != 101 || got.OrderID != "ORDER_20260903_001" {
				t.Fatalf("result lost validated data: %+v", got)
			}
			if test.mode == PaymentModeAlipay && len(got.BusinessData) != 0 {
				t.Fatal("Alipay result exposed unnecessary full business data")
			}
			if test.mode == PaymentModeWeChatMiniProgram && len(got.BusinessData) == 0 {
				t.Fatal("WeChat result omitted required business data")
			}
			gateway.mu.Lock()
			wire := gateway.requests[0]
			business := gateway.businesses[0]
			gateway.mu.Unlock()
			if wire["certId"] != "INITIATOR_TEST" || business["mercId"] != "PAYEE_TEST" {
				t.Fatalf("identity mapping certId=%q mercId=%v", wire["certId"], business["mercId"])
			}
			if wire["version"] != "1.4" || business["payMode"] != string(test.mode) {
				t.Fatalf("version/payMode = %q/%v", wire["version"], business["payMode"])
			}
			if _, exists := business["appType"]; exists {
				t.Fatal("undocumented appType must not be sent")
			}
		})
	}
}

func TestTDD016R_CreateCashierOrderAcceptsProductionNestedEncryptedData(t *testing.T) {
	t.Parallel()
	client, gateway := newGatewayClient(t, func(method string, business map[string]any) (string, any) {
		if method != "order.createOrder" {
			t.Fatalf("method = %q", method)
		}
		return "0", map[string]any{
			"rpcError":   false,
			"suffixCode": 0,
			"data": map[string]any{
				"reqMsgId":           "REQUEST_001",
				"amount":             "101",
				"mercId":             "PAYEE_TEST",
				"orderId":            "ORDER_20260903_001",
				"encryData":          "client-data",
				"orderCreateTime":    "2026-09-03 12:00:00",
				"orderEfficientTime": "2026-09-03 12:30:00",
			},
		}
	})

	got, err := client.CreateCashierOrder(context.Background(), validCreateRequest(PaymentModeAlipay))
	if err != nil {
		t.Fatalf("CreateCashierOrder() error = %v", err)
	}
	if got.OrderID != "ORDER_20260903_001" || got.PayeeMerchantID != "PAYEE_TEST" || got.AmountFen != 101 || got.EncryptedData != "client-data" {
		t.Fatalf("CreateCashierOrder() = %+v", got)
	}
	if got.PayURL != "" || len(got.BusinessData) != 0 {
		t.Fatalf("Alipay result exposed unexpected launch data: %+v", got)
	}
	gateway.mu.Lock()
	version := gateway.requests[0]["version"]
	gateway.mu.Unlock()
	if version != "1.4" {
		t.Fatalf("version = %q, want 1.4", version)
	}
}

func TestTDD017R_WeChatAcceptsNestedBusinessDataWithoutOptionalAppID(t *testing.T) {
	t.Parallel()
	client, _ := newGatewayClient(t, func(string, map[string]any) (string, any) {
		return "0", map[string]any{
			"rpcError":   false,
			"suffixCode": 0,
			"data": map[string]any{
				"reqMsgId":  "REQUEST_001",
				"amount":    "101",
				"mercId":    "PAYEE_TEST",
				"orderId":   "ORDER_20260903_001",
				"encryData": "client-data",
			},
		}
	})

	got, err := client.CreateCashierOrder(context.Background(), validCreateRequest(PaymentModeWeChatMiniProgram))
	if err != nil {
		t.Fatalf("CreateCashierOrder() error = %v", err)
	}
	if got.CashierAppID != "" || got.EncryptedData != "client-data" || len(got.BusinessData) == 0 {
		t.Fatalf("CreateCashierOrder() = %+v", got)
	}
	var business map[string]any
	if err := json.Unmarshal(got.BusinessData, &business); err != nil || business["data"] == nil {
		t.Fatalf("BusinessData is not the verified root object: %s", got.BusinessData)
	}
}

func TestPAY003AndID002AndID003_CreateValidationHappensBeforeNetwork(t *testing.T) {
	t.Parallel()
	calls := 0
	client, _ := newGatewayClient(t, func(string, map[string]any) (string, any) {
		calls++
		return "0", map[string]any{}
	})
	tests := []struct {
		name   string
		mutate func(*CreateCashierOrderRequest)
	}{
		{name: "missing order", mutate: func(r *CreateCashierOrderRequest) { r.OrderID = "" }},
		{name: "missing payee", mutate: func(r *CreateCashierOrderRequest) { r.PayeeMerchantID = "" }},
		{name: "missing business code", mutate: func(r *CreateCashierOrderRequest) { r.BusinessCode = "" }},
		{name: "unsupported business code", mutate: func(r *CreateCashierOrderRequest) { r.BusinessCode = "00510199" }},
		{name: "unsupported pay mode", mutate: func(r *CreateCashierOrderRequest) { r.PayMode = "28" }},
		{name: "invalid amount", mutate: func(r *CreateCashierOrderRequest) { r.AmountFen = 0 }},
		{name: "oversized amount", mutate: func(r *CreateCashierOrderRequest) { r.AmountFen = 10_000_000_000 }},
		{name: "invalid duration", mutate: func(r *CreateCashierOrderRequest) { r.PaymentValidMinutes = 31 }},
		{name: "invalid date", mutate: func(r *CreateCashierOrderRequest) { r.ShopDate = "20260230" }},
		{name: "missing callback", mutate: func(r *CreateCashierOrderRequest) { r.BackURL = "" }},
	}
	for _, test := range tests {
		req := validCreateRequest(PaymentModeAlipay)
		test.mutate(&req)
		if _, err := client.CreateCashierOrder(context.Background(), req); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s error = %v, want ErrInvalidRequest", test.name, err)
		}
	}
	if calls != 0 {
		t.Fatalf("network calls = %d, want 0", calls)
	}
}

func TestTDD002_OfficialEndpointMatrix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		operation operationSpec
		testURL   string
		prodURL   string
	}{
		{createOrderOperation, "https://appdev.ysepay.com/openapi/order/createOrder", "https://ysgate.ysepay.com/openapi/order/createOrder"},
		{queryTradeOperation, "https://appdev.ysepay-test.com/openapi/unify/online/trade/order/query", "https://ysgate.ysepay.com/openapi/unify/online/trade/order/query"},
		{refundOperation, "https://appdev.ysepay.com/openapi/unify/trade/refund", "https://ysgate.ysepay.com/openapi/unify/trade/refund"},
		{queryRefundOperation, "https://appdev.ysepay-test.com/openapi/unify/trade/refund/query", "https://ysgate.ysepay.com/openapi/unify/trade/refund/query"},
	}
	for _, test := range tests {
		if got := test.operation.endpoint(EnvironmentTest); got != test.testURL {
			t.Errorf("%s test endpoint = %q", test.operation.name, got)
		}
		if got := test.operation.endpoint(EnvironmentProduction); got != test.prodURL {
			t.Errorf("%s production endpoint = %q", test.operation.name, got)
		}
	}
}

func TestCreateAllowsOfficialCommaSeparatedCallbackURLs(t *testing.T) {
	t.Parallel()
	client, _ := newGatewayClient(t, func(string, map[string]any) (string, any) {
		return "0", map[string]any{"amount": "101", "mercId": "PAYEE_TEST", "orderId": "ORDER_20260903_001", "payUrl": "alipays://cashier/test"}
	})
	request := validCreateRequest(PaymentModeAlipay)
	request.BackURL = "https://one.invalid/notify,https://two.invalid/notify"
	if _, err := client.CreateCashierOrder(context.Background(), request); err != nil {
		t.Fatalf("CreateCashierOrder() error = %v", err)
	}
}

func TestPAY004AndPAY005_CreateRejectsMissingChannelLaunchData(t *testing.T) {
	t.Parallel()
	for _, mode := range []PaymentMode{PaymentModeAlipay, PaymentModeWeChatMiniProgram} {
		mode := mode
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()
			client, _ := newGatewayClient(t, func(string, map[string]any) (string, any) {
				return "0", map[string]any{"amount": "101", "mercId": "PAYEE_TEST", "orderId": "ORDER_20260903_001"}
			})
			_, err := client.CreateCashierOrder(context.Background(), validCreateRequest(mode))
			if !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf("error = %v, want ErrInvalidResponse", err)
			}
		})
	}
}

func TestCRY012_ResponseIsVerifiedBeforeDecryption(t *testing.T) {
	t.Parallel()
	client, gateway := newGatewayClient(t, func(string, map[string]any) (string, any) {
		return "0", map[string]any{"payUrl": "alipays://cashier/test"}
	})
	gateway.tamperSign = true
	_, err := client.CreateCashierOrder(context.Background(), validCreateRequest(PaymentModeAlipay))
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("error = %v, want ErrInvalidSignature", err)
	}
}

func TestHTTP_ResponseSupportsOfficialRawAndBase64JSONForms(t *testing.T) {
	t.Parallel()
	client, gateway := newGatewayClient(t, func(string, map[string]any) (string, any) {
		return "0", map[string]any{
			"amount": "101", "mercId": "PAYEE_TEST", "orderId": "ORDER_20260903_001", "payUrl": "alipays://cashier/test",
		}
	})
	gateway.base64Result = true
	if _, err := client.CreateCashierOrder(context.Background(), validCreateRequest(PaymentModeAlipay)); err != nil {
		t.Fatalf("base64 response error = %v", err)
	}
}

func TestHTTP001To003_TransportFailuresRemainUncertainAndBounded(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		transport roundTripFunc
	}{
		{name: "network error", transport: func(*http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded }},
		{name: "non 2xx", transport: func(*http.Request) (*http.Response, error) { return jsonResponse(503, []byte("unavailable")), nil }},
		{name: "invalid json", transport: func(*http.Request) (*http.Response, error) { return jsonResponse(200, []byte("not-json")), nil }},
		{name: "oversize", transport: func(*http.Request) (*http.Response, error) {
			return jsonResponse(200, []byte(strings.Repeat("x", maxResponseBytes+1))), nil
		}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			config, _ := newTestConfig(t, test.transport)
			client, err := NewClient(config)
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.CreateCashierOrder(context.Background(), validCreateRequest(PaymentModeAlipay))
			if !errors.Is(err, ErrUncertain) {
				t.Fatalf("error = %v, want ErrUncertain", err)
			}
			if !errors.Is(err, ErrTransport) && test.name != "invalid json" {
				t.Fatalf("error = %v, want transport classification", err)
			}
		})
	}
}

func TestQRY001_QueryTradePreservesAndClassifiesStatuses(t *testing.T) {
	t.Parallel()
	tests := map[string]ResultState{
		"00": ResultSucceeded,
		"11": ResultPending,
		"13": ResultPending,
		"14": ResultPending,
		"80": ResultPending,
		"81": ResultPending,
		"50": ResultFailed,
		"93": ResultFailed,
		"95": ResultFailed,
		"97": ResultFailed,
		"98": ResultFailed,
		"99": ResultFailed,
		"XX": ResultUnknown,
	}
	for status, want := range tests {
		status, want := status, want
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			client, _ := newGatewayClient(t, func(method string, business map[string]any) (string, any) {
				if method != "unify.online.trade.order.query" || business["orderId"] != "ORDER_20260903_001" {
					t.Fatalf("query request method=%q business=%v", method, business)
				}
				return "0000", map[string]any{"reqMsgId": "REQ", "systemCode": "SYS", "data": map[string]any{
					"tradeStatus": status, "orderId": "ORDER_20260903_001", "tradeSn": "TRADE", "totalAmount": "101", "receiptAmount": "100",
				}}
			})
			got, err := client.QueryTrade(context.Background(), QueryTradeRequest{OrderID: "ORDER_20260903_001", ShopDate: "20260903"})
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != status || got.State != want || got.TotalAmountFen != 101 {
				t.Fatalf("QueryTrade() = %+v, want status/state %s/%s", got, status, want)
			}
		})
	}
}

func TestQRY_ValidationRequiresOrderOrTradeNumber(t *testing.T) {
	t.Parallel()
	client, _ := newGatewayClient(t, func(string, map[string]any) (string, any) { return "0000", nil })
	_, err := client.QueryTrade(context.Background(), QueryTradeRequest{ShopDate: "20260903"})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestBDD_REF_001_RefundIsAcceptedButNotReportedAsFinalSuccess(t *testing.T) {
	t.Parallel()
	client, _ := newGatewayClient(t, func(method string, business map[string]any) (string, any) {
		if method != "unify.trade.refund" || business["refundAmount"] != "1" {
			t.Fatalf("refund request method=%q business=%v", method, business)
		}
		return "0000", map[string]any{"reqMsgId": "REQ", "businessData": map[string]any{
			"refundAmount": "1", "origTradeSn": "TRADE", "origOrderId": "ORDER_20260903_001", "accountDate": "20260903", "refundOrderId": "REFUND_20260903_001", "refundTradeSn": "REFUND_TRADE",
		}}
	})
	got, err := client.Refund(context.Background(), validRefundRequest())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Accepted || got.RefundOrderID != "REFUND_20260903_001" || got.AmountFen != 1 {
		t.Fatalf("Refund() = %+v", got)
	}
}

func TestREF002_RefundValidationPreventsNetwork(t *testing.T) {
	t.Parallel()
	client, _ := newGatewayClient(t, func(string, map[string]any) (string, any) {
		t.Fatal("network must not be called")
		return "", nil
	})
	for _, mutate := range []func(*RefundRequest){
		func(r *RefundRequest) { r.OriginalOrderID = "" },
		func(r *RefundRequest) { r.AmountFen = 0 },
		func(r *RefundRequest) { r.RefundOrderID = "" },
		func(r *RefundRequest) { r.Reason = "" },
	} {
		req := validRefundRequest()
		mutate(&req)
		if _, err := client.Refund(context.Background(), req); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("Refund() error = %v, want ErrInvalidRequest", err)
		}
	}
}

func TestREF003_QueryRefundPreservesAndClassifiesStatuses(t *testing.T) {
	t.Parallel()
	for status, want := range map[string]ResultState{"00": ResultSucceeded, "10": ResultPending, "96": ResultFailed, "97": ResultFailed, "98": ResultFailed, "99": ResultFailed, "XX": ResultUnknown} {
		status, want := status, want
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			client, _ := newGatewayClient(t, func(method string, _ map[string]any) (string, any) {
				if method != "unify.trade.refund.query" {
					t.Fatalf("method = %q", method)
				}
				return "0000", map[string]any{"reqMsgId": "REQ", "businessData": map[string]any{"data": map[string]any{
					"origOrderId": "ORDER", "origTradeSn": "TRADE", "refundOrderId": "REFUND", "refundState": status, "totalAmount": "101", "refundAmount": "1", "fundsState": "00",
				}}}
			})
			got, err := client.QueryRefund(context.Background(), QueryRefundRequest{RefundOrderID: "REFUND"})
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != status || got.State != want || got.RefundAmountFen != 1 {
				t.Fatalf("QueryRefund() = %+v", got)
			}
		})
	}
}

func TestGatewayRejectIsTypedAndDoesNotLeakUpstreamMessage(t *testing.T) {
	t.Parallel()
	const upstreamSecret = "upstream-sensitive-details"
	client, _ := newGatewayClient(t, func(string, map[string]any) (string, any) {
		return "FAIL_CODE", map[string]any{"sensitive": upstreamSecret}
	})
	_, err := client.CreateCashierOrder(context.Background(), validCreateRequest(PaymentModeAlipay))
	if !errors.Is(err, ErrUpstreamRejected) {
		t.Fatalf("error = %v, want ErrUpstreamRejected", err)
	}
	var gatewayErr *GatewayError
	if !errors.As(err, &gatewayErr) || gatewayErr.SubCode != "FAIL_CODE" {
		t.Fatalf("gateway error = %#v", err)
	}
	if strings.Contains(err.Error(), upstreamSecret) {
		t.Fatalf("error leaked upstream content: %v", err)
	}
}

func TestGatewayLevelFailureIsRejectedBeforeBusinessSuccess(t *testing.T) {
	t.Parallel()
	client, gateway := newGatewayClient(t, func(string, map[string]any) (string, any) {
		return "0", map[string]any{"amount": "101", "mercId": "PAYEE_TEST", "orderId": "ORDER_20260903_001", "payUrl": "alipays://cashier/test"}
	})
	original := gateway.roundTrip
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response, err := original(request)
		if err != nil {
			return nil, err
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		var fields map[string]string
		if err := json.Unmarshal(body, &fields); err != nil {
			t.Fatal(err)
		}
		fields["code"] = "10000"
		fields["sign"], err = signFields(fields, gateway.ysePrivate)
		if err != nil {
			t.Fatal(err)
		}
		body, _ = json.Marshal(fields)
		return jsonResponse(http.StatusOK, body), nil
	})
	_, err := client.CreateCashierOrder(context.Background(), validCreateRequest(PaymentModeAlipay))
	if !errors.Is(err, ErrUpstreamRejected) {
		t.Fatalf("error = %v, want ErrUpstreamRejected", err)
	}
}

func TestGatewayErrorSanitizesUntrustedCodes(t *testing.T) {
	t.Parallel()
	err := rejectedError("operation", "CODE\nsecret", strings.Repeat("x", 100))
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), strings.Repeat("x", 100)) {
		t.Fatalf("GatewayError leaked untrusted code content: %v", err)
	}
}

func TestCON001_ClientIsSafeForConcurrentUse(t *testing.T) {
	client, gateway := newGatewayClient(t, func(string, map[string]any) (string, any) {
		return "0", map[string]any{"amount": "101", "mercId": "PAYEE_TEST", "orderId": "ORDER_20260903_001", "payUrl": "alipays://cashier/test"}
	})
	const count = 40
	var wait sync.WaitGroup
	for range count {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := client.CreateCashierOrder(context.Background(), validCreateRequest(PaymentModeAlipay)); err != nil {
				t.Errorf("CreateCashierOrder() error = %v", err)
			}
		}()
	}
	wait.Wait()
	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	if len(gateway.requests) != count {
		t.Fatalf("requests = %d, want %d", len(gateway.requests), count)
	}
	seen := make(map[string]struct{}, count)
	for _, request := range gateway.requests {
		if _, exists := seen[request["reqId"]]; exists {
			t.Fatalf("duplicate reqId %q", request["reqId"])
		}
		seen[request["reqId"]] = struct{}{}
	}
}

func TestWriteOperationsDoNotRetry(t *testing.T) {
	t.Parallel()
	calls := 0
	config, _ := newTestConfig(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, io.ErrUnexpectedEOF
	}))
	client, err := NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = client.Refund(context.Background(), validRefundRequest())
	if calls != 1 {
		t.Fatalf("HTTP calls = %d, want exactly 1", calls)
	}
}
