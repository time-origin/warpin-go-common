package ysepay

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func signedNotification(t *testing.T, privateKey any, mutateBusiness func(map[string]any)) []byte {
	t.Helper()
	business := map[string]any{
		"mercId":        "PAYEE_TEST",
		"orderId":       "ORDER_20260903_001",
		"amount":        "1.01",
		"currencyCode":  "CNY",
		"tradeSn":       "TRADE_001",
		"settlementAmt": "1.00",
		"payAmt":        "1.01",
		"payTime":       "2026-09-03 12:00:00",
		"channelSendSn": "CHANNEL_SEND",
		"channelRecvSn": "CHANNEL_RECV",
	}
	if mutateBusiness != nil {
		mutateBusiness(business)
	}
	bizContent, err := json.Marshal(business)
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]string{
		"timeStamp":  "2026-09-03 12:00:00",
		"src":        "pregate",
		"reqId":      "notification-request",
		"charset":    "UTF-8",
		"bizContent": string(bizContent),
	}
	key, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		t.Fatal("test private key is not RSA")
	}
	fields["sign"], err = signFields(fields, key)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestBDD_NOT_001AndNOT022_ValidNotificationIsExactAndDeterministic(t *testing.T) {
	t.Parallel()
	config, yse := newTestConfig(t, nil)
	client, err := NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	payload := signedNotification(t, yse.private, nil)
	first, err := client.ParsePaymentNotification(payload)
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.ParsePaymentNotification(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("duplicate parse differs: %#v != %#v", first, second)
	}
	if first.PayeeMerchantID != "PAYEE_TEST" || first.OrderID != "ORDER_20260903_001" || first.AmountFen != 101 || first.SettlementAmountFen != 100 {
		t.Fatalf("notification = %+v", first)
	}
}

func TestBDD_SEC_001_TamperedNotificationIsRejected(t *testing.T) {
	t.Parallel()
	config, yse := newTestConfig(t, nil)
	client, err := NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	payload := signedNotification(t, yse.private, nil)
	var fields map[string]string
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	fields["reqId"] = "tampered"
	payload, _ = json.Marshal(fields)
	if _, err := client.ParsePaymentNotification(payload); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("error = %v, want ErrInvalidSignature", err)
	}
}

func TestQA001_NotificationTamperMatrix(t *testing.T) {
	t.Parallel()
	for _, field := range []string{"reqId", "bizContent", "sign"} {
		field := field
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			config, yse := newTestConfig(t, nil)
			client, err := NewClient(config)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]string
			if err := json.Unmarshal(signedNotification(t, yse.private, nil), &fields); err != nil {
				t.Fatal(err)
			}
			fields[field] += "tampered"
			payload, _ := json.Marshal(fields)
			if _, err := client.ParsePaymentNotification(payload); !errors.Is(err, ErrInvalidSignature) {
				t.Fatalf("tampered %s error = %v, want ErrInvalidSignature", field, err)
			}
		})
	}
}

func TestNOT003AndAMT003_InvalidNotificationFieldsAreRejected(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "missing merchant", mutate: func(v map[string]any) { delete(v, "mercId") }},
		{name: "missing order", mutate: func(v map[string]any) { delete(v, "orderId") }},
		{name: "fraction precision", mutate: func(v map[string]any) { v["amount"] = "1.001" }},
		{name: "scientific amount", mutate: func(v map[string]any) { v["amount"] = "1e3" }},
		{name: "wrong currency", mutate: func(v map[string]any) { v["currencyCode"] = "USD" }},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			config, yse := newTestConfig(t, nil)
			client, err := NewClient(config)
			if err != nil {
				t.Fatal(err)
			}
			payload := signedNotification(t, yse.private, test.mutate)
			if _, err := client.ParsePaymentNotification(payload); !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf("error = %v, want ErrInvalidResponse", err)
			}
		})
	}
}

func FuzzParsePaymentNotification(f *testing.F) {
	config, _ := newTestConfig(f, nil)
	client, err := NewClient(config)
	if err != nil {
		f.Fatal(err)
	}
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"sign":"invalid","bizContent":"{}"}`))
	f.Fuzz(func(t *testing.T, payload []byte) {
		_, _ = client.ParsePaymentNotification(payload)
	})
}

func FuzzYuanToFen(f *testing.F) {
	f.Add("0.01")
	f.Add("1e999999")
	f.Fuzz(func(t *testing.T, value string) {
		_, _ = YuanToFen(value)
	})
}

func FuzzParseYseCertificate(f *testing.F) {
	certificate := newTestCertificate(f, "yse.test")
	f.Add(certificate.der)
	random := make([]byte, 64)
	_, _ = rand.Read(random)
	f.Add(random)
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = parseRSAPublicCertificate(data)
	})
}

func FuzzDecodeResponseFields(f *testing.F) {
	f.Add([]byte(`{"code":"10000"}`))
	f.Add([]byte("bm90LWpzb24="))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = decodeResponseFields(data)
	})
}
