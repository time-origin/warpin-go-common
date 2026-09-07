package ysepay

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestCFG001_NewClientRejectsMissingInitiatorMerchantID(t *testing.T) {
	t.Parallel()

	_, err := NewClient(Config{})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("NewClient() error = %v, want ErrInvalidConfig", err)
	}
}

func TestCFG006_OfficialLegacyRSACertificateIsAccepted(t *testing.T) {
	t.Parallel()
	// Public certificate from Ysepay's official Java demo. It has a legacy
	// negative serial number and is safe to keep as a non-secret test vector.
	const officialServiceCertificate = "MIICljCCAf+gAwIBAgIEmaPvDjANBgkqhkiG9w0BAQQFADBcMQ8wDQYDVQQDDAZ5c2VwYXkxDzANBgNVBAsMBnlzZXBheTERMA8GA1UECgwIb3JnYW5pemUxCzAJBgNVBAcMAnN6MQswCQYDVQQGEwJjbjELMAkGA1UECAwCZ2QwIBcNMjExMDA5MDY0OTExWhgPMjA1MTEwMDIwNjQ5MTFaMGUxGDAWBgNVBAMMD29wZW5hcGktc2VydmljZTEPMA0GA1UECwwGeXNlcGF5MREwDwYDVQQKDAhvcmdhbml6ZTELMAkGA1UEBwwCc3oxCzAJBgNVBAYTAmNuMQswCQYDVQQIDAJnZDCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAzWHp5Ld7RgX98hiwYDSl6Y++oCNyhYXBOjijvuDb20E8e9JYNvbOKH4pD+X1Ux4uIv/xk0zxEskM9RfPm9lmrDgWVBtwkvOO6NSoskYllEcSuC4GoJwap73vLUCfqh0/x/Ii+shXYqY6tWZMrtvaZoTILGyvvsx14/kpK1nV1pMCAwEAAaNaMFgwHQYDVR0OBBYEFJM1PLx7wOuDFDW93X21/6svKr70MB8GA1UdIwQYMBaAFM1FzVxKgu/KdBqtv+kRNuBYKXKYMAkGA1UdEwQCMAAwCwYDVR0PBAQDAgSQMA0GCSqGSIb3DQEBBAUAA4GBAH29QsVwZh6LfkSAb01GbdAwNrWzgs+R4LnC2uET5MYXkwxssiJU6tFH/VZLTKeZ29NmJVhMBeCjJlX7q8RWFPgu0q2zmVJu6delid+mL67nPGsp9KFTVoDk/d+LmtupMFys/yAcAh9tYyUYzbPAUUk+K5t69XGrmhAXvavmd59Z"
	der, err := base64.StdEncoding.DecodeString(officialServiceCertificate)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := parseRSAPublicCertificate(der)
	if err != nil {
		t.Fatalf("parseRSAPublicCertificate() error = %v", err)
	}
	if certificate.publicKey == nil || certificate.publicKey.N.BitLen() != 1024 {
		t.Fatalf("unexpected official public key: %#v", certificate.publicKey)
	}
}

func TestCFG002AndCFG003_NewClientRejectsBadCertificateMaterialWithoutLeakingIt(t *testing.T) {
	t.Parallel()
	config, _ := newTestConfig(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("unused")
	}))
	tests := []struct {
		name   string
		mutate func(*Config)
		secret string
	}{
		{name: "wrong pfx password", mutate: func(c *Config) { c.SigningCertificatePassword = "do-not-leak" }, secret: "do-not-leak"},
		{name: "damaged cer", mutate: func(c *Config) { c.YsePublicCertificate = []byte("do-not-leak-certificate") }, secret: "do-not-leak-certificate"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			configCopy := config
			test.mutate(&configCopy)
			_, err := NewClient(configCopy)
			if !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("NewClient() error = %v, want ErrInvalidConfig", err)
			}
			if strings.Contains(err.Error(), test.secret) {
				t.Fatalf("error leaked secret: %v", err)
			}
		})
	}
}

func TestCFG004_NewClientRejectsUnknownEnvironment(t *testing.T) {
	t.Parallel()
	config, _ := newTestConfig(t, nil)
	config.Environment = Environment("https://attacker.invalid")
	_, err := NewClient(config)
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("NewClient() error = %v, want ErrInvalidConfig", err)
	}
}

func TestCFG005_NewClientAcceptsVendorStyleBERIndefinitePFX(t *testing.T) {
	t.Parallel()
	config, _ := newTestConfig(t, nil)
	content, ok := stripDERSequenceHeader(config.SigningPKCS12)
	if !ok {
		t.Fatal("test PFX is not a DER sequence")
	}
	config.SigningPKCS12 = append([]byte{0x30, 0x80}, content...)
	config.SigningPKCS12 = append(config.SigningPKCS12, 0, 0)
	if _, err := NewClient(config); err != nil {
		t.Fatalf("NewClient() BER PFX error = %v", err)
	}
}

func stripDERSequenceHeader(data []byte) ([]byte, bool) {
	if len(data) < 2 || data[0] != 0x30 {
		return nil, false
	}
	if data[1] < 0x80 {
		length := int(data[1])
		if length != len(data)-2 {
			return nil, false
		}
		return data[2:], true
	}
	count := int(data[1] & 0x7f)
	if count == 0 || count > 4 || len(data) < 2+count {
		return nil, false
	}
	length := 0
	for _, value := range data[2 : 2+count] {
		length = length<<8 | int(value)
	}
	if length != len(data)-(2+count) {
		return nil, false
	}
	return data[2+count:], true
}
