package ysepay

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"sync"
	"testing"
	"time"

	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

const testPFXPassword = "fixture-password"

type testCertificate struct {
	private *rsa.PrivateKey
	cert    *x509.Certificate
	der     []byte
}

func newTestCertificate(t testing.TB, commonName string) testCertificate {
	t.Helper()
	private, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano()),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &private.PublicKey, private)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return testCertificate{private: private, cert: cert, der: der}
}

func newTestConfig(t testing.TB, transport http.RoundTripper) (Config, testCertificate) {
	t.Helper()
	initiator := newTestCertificate(t, "initiator.test")
	yse := newTestCertificate(t, "yse.test")
	pfx, err := pkcs12.Encode(rand.Reader, initiator.private, initiator.cert, nil, testPFXPassword)
	if err != nil {
		t.Fatal(err)
	}
	return Config{
		Environment:                EnvironmentTest,
		InitiatorMerchantID:        "INITIATOR_TEST",
		SigningPKCS12:              pfx,
		SigningCertificatePassword: testPFXPassword,
		YsePublicCertificate:       yse.der,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   time.Second,
		},
	}, yse
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func jsonResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

type fakeGateway struct {
	t               *testing.T
	ysePrivate      *rsa.PrivateKey
	initiatorPublic *rsa.PublicKey
	mu              sync.Mutex
	requests        []map[string]string
	businesses      []map[string]any
	responseFor     func(method string, business map[string]any) (string, any)
	tamperSign      bool
	base64Result    bool
}

func (g *fakeGateway) roundTrip(request *http.Request) (*http.Response, error) {
	g.t.Helper()
	if request.Method != http.MethodPost {
		g.t.Errorf("method = %s, want POST", request.Method)
	}
	if got := request.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		g.t.Errorf("Content-Type = %q", got)
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	var wire map[string]string
	if err := json.Unmarshal(body, &wire); err != nil {
		g.t.Fatalf("decode request: %v", err)
	}
	if err := verifyFields(wire, g.initiatorPublic); err != nil {
		g.t.Fatalf("verify request signature: %v", err)
	}
	aesKey, err := rsa.DecryptPKCS1v15(rand.Reader, g.ysePrivate, mustBase64(g.t, wire["check"]))
	if err != nil {
		g.t.Fatalf("decrypt check: %v", err)
	}
	plain, err := decryptAES(mustBase64(g.t, wire["bizContent"]), aesKey)
	if err != nil {
		g.t.Fatalf("decrypt bizContent: %v", err)
	}
	var business map[string]any
	if err := json.Unmarshal(plain, &business); err != nil {
		g.t.Fatalf("decode business: %v", err)
	}
	g.mu.Lock()
	g.requests = append(g.requests, wire)
	g.businesses = append(g.businesses, business)
	g.mu.Unlock()

	subCode, responseBusiness := g.responseFor(wire["method"], business)
	businessJSON, err := json.Marshal(responseBusiness)
	if err != nil {
		g.t.Fatal(err)
	}
	ciphertext, err := encryptAES(businessJSON, aesKey)
	if err != nil {
		g.t.Fatal(err)
	}
	response := map[string]string{
		"code":         "00000",
		"msg":          "SUCCESS",
		"subCode":      subCode,
		"subMsg":       "SUCCESS",
		"timeStamp":    "2026-09-03 12:00:00",
		"norce":        "test",
		"businessData": base64.StdEncoding.EncodeToString(ciphertext),
	}
	response["sign"], err = signFields(response, g.ysePrivate)
	if err != nil {
		g.t.Fatal(err)
	}
	if g.tamperSign {
		response["subMsg"] = "TAMPERED"
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		g.t.Fatal(err)
	}
	if g.base64Result {
		encoded = []byte(base64.StdEncoding.EncodeToString(encoded))
	}
	return jsonResponse(http.StatusOK, encoded), nil
}

func mustBase64(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func newGatewayClient(t *testing.T, responseFor func(string, map[string]any) (string, any)) (*Client, *fakeGateway) {
	t.Helper()
	gateway := &fakeGateway{t: t, responseFor: responseFor}
	config, yse := newTestConfig(t, roundTripFunc(gateway.roundTrip))
	gateway.ysePrivate = yse.private
	client, err := NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	client.now = func() time.Time { return time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC) }
	gateway.initiatorPublic = &client.privateKey.PublicKey
	return client, gateway
}

func validCreateRequest(mode PaymentMode) CreateCashierOrderRequest {
	return CreateCashierOrderRequest{
		OrderID:             "ORDER_20260903_001",
		PayeeMerchantID:     "PAYEE_TEST",
		BusinessCode:        BusinessCodeStandard,
		ShopDate:            "20260903",
		AmountFen:           101,
		PaymentValidMinutes: 30,
		BackURL:             "https://merchant.invalid/payment/notify",
		PayMode:             mode,
	}
}

func validRefundRequest() RefundRequest {
	return RefundRequest{
		OriginalOrderID: "ORDER_20260903_001",
		ShopDate:        "20260903",
		AmountFen:       1,
		Reason:          "test refund",
		RefundOrderID:   "REFUND_20260903_001",
	}
}
