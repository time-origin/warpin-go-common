package ysepay

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

const defaultHTTPTimeout = 20 * time.Second

type Config struct {
	Environment                Environment
	InitiatorMerchantID        string
	SigningPKCS12              []byte
	SigningCertificatePassword string
	YsePublicCertificate       []byte
	HTTPClient                 *http.Client
}

func NewClient(config Config) (*Client, error) {
	if config.Environment != EnvironmentTest && config.Environment != EnvironmentProduction {
		return nil, fmt.Errorf("%w: unsupported environment", ErrInvalidConfig)
	}
	merchantID := strings.TrimSpace(config.InitiatorMerchantID)
	if merchantID == "" {
		return nil, fmt.Errorf("%w: initiator merchant ID is required", ErrInvalidConfig)
	}
	if len(config.SigningPKCS12) == 0 {
		return nil, fmt.Errorf("%w: signing PKCS#12 is required", ErrInvalidConfig)
	}
	if config.SigningCertificatePassword == "" {
		return nil, fmt.Errorf("%w: signing certificate password is required", ErrInvalidConfig)
	}
	if len(config.YsePublicCertificate) == 0 {
		return nil, fmt.Errorf("%w: Ysepay public certificate is required", ErrInvalidConfig)
	}

	key, certificate, _, err := pkcs12.DecodeChain(config.SigningPKCS12, config.SigningCertificatePassword)
	if err != nil {
		if normalized, normalizeErr := normalizePKCS12BER(config.SigningPKCS12, config.SigningCertificatePassword); normalizeErr == nil {
			key, certificate, _, err = pkcs12.DecodeChain(normalized, config.SigningCertificatePassword)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("%w: cannot decode signing PKCS#12", ErrInvalidConfig)
	}
	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: signing key is not RSA", ErrInvalidConfig)
	}
	certificatePublicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
	if !ok || privateKey.PublicKey.N.Cmp(certificatePublicKey.N) != 0 || privateKey.PublicKey.E != certificatePublicKey.E {
		return nil, fmt.Errorf("%w: signing key and certificate do not match", ErrInvalidConfig)
	}
	if err := validateCertificateTime(certificate, time.Now()); err != nil {
		return nil, fmt.Errorf("%w: signing certificate is not currently valid", ErrInvalidConfig)
	}

	yseCertificate, err := parseRSAPublicCertificate(config.YsePublicCertificate)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot parse Ysepay public certificate", ErrInvalidConfig)
	}
	if err := validateRSAPublicCertificateTime(yseCertificate, time.Now()); err != nil {
		return nil, fmt.Errorf("%w: Ysepay public certificate is not currently valid", ErrInvalidConfig)
	}
	ysePublicKey := yseCertificate.publicKey

	httpClient := &http.Client{Timeout: defaultHTTPTimeout}
	if config.HTTPClient != nil {
		copy := *config.HTTPClient
		httpClient = &copy
		if httpClient.Timeout <= 0 {
			httpClient.Timeout = defaultHTTPTimeout
		}
	}
	httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &Client{
		environment:         config.Environment,
		initiatorMerchantID: merchantID,
		privateKey:          privateKey,
		ysePublicKey:        ysePublicKey,
		httpClient:          httpClient,
		now:                 time.Now,
	}, nil
}

type rsaPublicCertificate struct {
	publicKey *rsa.PublicKey
	notBefore time.Time
	notAfter  time.Time
}

func parseRSAPublicCertificate(data []byte) (*rsaPublicCertificate, error) {
	if len(data) == 0 || len(data) > maxResponseBytes {
		return nil, errors.New("certificate size is invalid")
	}
	der := data
	if block, _ := pem.Decode(data); block != nil {
		if block.Type != "CERTIFICATE" {
			return nil, errors.New("unexpected PEM block")
		}
		der = block.Bytes
	}
	certificate, err := x509.ParseCertificate(der)
	if err == nil {
		publicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("certificate key is not RSA")
		}
		return &rsaPublicCertificate{publicKey: publicKey, notBefore: certificate.NotBefore, notAfter: certificate.NotAfter}, nil
	}
	// Older Ysepay certificates may use a negative serial number, which modern
	// crypto/x509 correctly rejects. Parse only the public-key and validity
	// fields through RawValue as a narrowly scoped compatibility fallback.
	var legacy struct {
		TBSCertificate struct {
			Raw                asn1.RawContent
			Version            int `asn1:"optional,explicit,default:0,tag:0"`
			SerialNumber       asn1.RawValue
			SignatureAlgorithm pkix.AlgorithmIdentifier
			Issuer             asn1.RawValue
			Validity           struct {
				NotBefore, NotAfter time.Time
			}
			Subject       asn1.RawValue
			SubjectPublic asn1.RawValue
		}
		SignatureAlgorithm pkix.AlgorithmIdentifier
		SignatureValue     asn1.BitString
	}
	if trailing, legacyErr := asn1.Unmarshal(der, &legacy); legacyErr != nil || len(trailing) != 0 || len(legacy.TBSCertificate.SubjectPublic.FullBytes) == 0 {
		return nil, err
	}
	parsedPublic, legacyErr := x509.ParsePKIXPublicKey(legacy.TBSCertificate.SubjectPublic.FullBytes)
	if legacyErr != nil {
		return nil, err
	}
	publicKey, ok := parsedPublic.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("certificate key is not RSA")
	}
	return &rsaPublicCertificate{
		publicKey: publicKey,
		notBefore: legacy.TBSCertificate.Validity.NotBefore,
		notAfter:  legacy.TBSCertificate.Validity.NotAfter,
	}, nil
}

func validateRSAPublicCertificateTime(certificate *rsaPublicCertificate, now time.Time) error {
	if now.Before(certificate.notBefore) || now.After(certificate.notAfter) {
		return errors.New("certificate is outside its validity interval")
	}
	return nil
}

func validateCertificateTime(certificate *x509.Certificate, now time.Time) error {
	if now.Before(certificate.NotBefore) || now.After(certificate.NotAfter) {
		return errors.New("certificate is outside its validity interval")
	}
	return nil
}
