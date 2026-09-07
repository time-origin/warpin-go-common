package ysepay

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCRY001_CanonicalSigningDataMatchesOfficialRule(t *testing.T) {
	t.Parallel()
	fields := map[string]string{
		"version":     "1.4",
		"sign":        "excluded",
		"bizContent":  "ciphertext",
		"certId":      "initiator",
		"$jacocoData": "excluded",
		"charset":     "utf-8",
	}
	got := canonicalSigningData(fields)
	want := "bizContent=ciphertext&certId=initiator&charset=utf-8&version=1.4"
	if got != want {
		t.Fatalf("canonicalSigningData() = %q, want %q", got, want)
	}
}

func TestCRY002_SHA256WithRSASignAndTamperDetection(t *testing.T) {
	t.Parallel()
	private, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]string{"a": "1", "b": "2"}
	signature, err := signFields(fields, private)
	if err != nil {
		t.Fatal(err)
	}
	fields["sign"] = signature
	if err := verifyFields(fields, &private.PublicKey); err != nil {
		t.Fatalf("verifyFields() error = %v", err)
	}
	fields["b"] = "tampered"
	if err := verifyFields(fields, &private.PublicKey); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("tampered verify error = %v, want ErrInvalidSignature", err)
	}
}

func TestCRY003_AESOfficialCompatibilityVector(t *testing.T) {
	t.Parallel()
	key := []byte("0123456789ABCDEF")
	plaintext := []byte(`{"a":"b"}`)
	want := "4qHcdRUQgtTJWcEQL3qPPA=="
	ciphertext, err := encryptAES(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}
	if got := base64.StdEncoding.EncodeToString(ciphertext); got != want {
		t.Fatalf("ciphertext = %s, want %s", got, want)
	}
	decrypted, err := decryptAES(ciphertext, key)
	if err != nil {
		t.Fatal(err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted = %q", decrypted)
	}
}

func TestCRY005_AESRejectsInvalidCiphertextAndPadding(t *testing.T) {
	t.Parallel()
	for _, ciphertext := range [][]byte{{1}, make([]byte, 16)} {
		if _, err := decryptAES(ciphertext, []byte("0123456789ABCDEF")); !errors.Is(err, ErrCrypto) {
			t.Fatalf("decryptAES(%d bytes) error = %v, want ErrCrypto", len(ciphertext), err)
		}
	}
}

func TestAMT001AndAMT002_ExactYuanToFen(t *testing.T) {
	t.Parallel()
	tests := map[string]int64{
		"0.01":        1,
		"1":           100,
		"1.2":         120,
		"99999999.99": 9_999_999_999,
	}
	for input, want := range tests {
		input, want := input, want
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			got, err := YuanToFen(input)
			if err != nil || got != want {
				t.Fatalf("YuanToFen(%q) = %d, %v; want %d", input, got, err, want)
			}
		})
	}
}

func TestAMT003_InvalidYuanAmountsAreRejected(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"", "-1", "+1", "1.001", "1e3", ".1", "1.", "92233720368547759"} {
		if _, err := YuanToFen(input); !errors.Is(err, ErrInvalidResponse) {
			t.Errorf("YuanToFen(%q) error = %v, want ErrInvalidResponse", input, err)
		}
	}
}

func TestREQ001_RequestIDContract(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 3, 12, 34, 56, 0, time.UTC)
	id, err := generateRequestID(rand.Reader, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 32 || !strings.HasSuffix(id, "260903123456") || !validRequestID(id) {
		t.Fatalf("generated invalid reqId %q", id)
	}
}

func TestREQ002_ConcurrentRequestIDsAreUnique(t *testing.T) {
	const count = 200
	now := time.Date(2026, 9, 3, 12, 34, 56, 0, time.UTC)
	ids := make(chan string, count)
	var wait sync.WaitGroup
	for range count {
		wait.Add(1)
		go func() {
			defer wait.Done()
			id, err := generateRequestID(rand.Reader, now)
			if err != nil {
				t.Errorf("generateRequestID() error = %v", err)
				return
			}
			ids <- id
		}()
	}
	wait.Wait()
	close(ids)
	seen := make(map[string]struct{}, count)
	for id := range ids {
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate reqId %q", id)
		}
		seen[id] = struct{}{}
	}
}
