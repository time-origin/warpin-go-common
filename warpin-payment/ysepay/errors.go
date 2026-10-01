// Package ysepay implements the Ysepay Xiao-Y aggregated cashier protocol.
// Credentials are supplied by the calling service at runtime; this package
// never reads certificate files or environment variables.
package ysepay

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidConfig    = errors.New("ysepay: invalid configuration")
	ErrInvalidRequest   = errors.New("ysepay: invalid request")
	ErrCrypto           = errors.New("ysepay: cryptographic operation failed")
	ErrInvalidSignature = errors.New("ysepay: invalid signature")
	ErrTransport        = errors.New("ysepay: transport failed")
	ErrInvalidResponse  = errors.New("ysepay: invalid response")
	ErrUpstreamRejected = errors.New("ysepay: request rejected")
	ErrUncertain        = errors.New("ysepay: result is uncertain")
)

// GatewayError is a sanitized error returned by a gateway operation. It never
// includes upstream messages, payloads, signatures, certificate data, or keys.
type GatewayError struct {
	Operation string
	Code      string
	SubCode   string
	kind      error
}

func (e *GatewayError) Error() string {
	if e == nil {
		return "ysepay: gateway error"
	}
	if e.SubCode != "" {
		return fmt.Sprintf("ysepay: %s failed (code=%s, sub_code=%s)", e.Operation, e.Code, e.SubCode)
	}
	if e.Code != "" {
		return fmt.Sprintf("ysepay: %s failed (code=%s)", e.Operation, e.Code)
	}
	return fmt.Sprintf("ysepay: %s failed", e.Operation)
}

func (e *GatewayError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.kind
}

func operationError(operation string, kind error) error {
	return &GatewayError{Operation: operation, kind: kind}
}

func rejectedError(operation, code, subCode string) error {
	return &GatewayError{Operation: operation, Code: safeGatewayCode(code), SubCode: safeGatewayCode(subCode), kind: ErrUpstreamRejected}
}

func safeGatewayCode(value string) string {
	if value == "" {
		return ""
	}
	if len(value) > 64 || strings.IndexFunc(value, func(character rune) bool {
		return !(character >= 'a' && character <= 'z') && !(character >= 'A' && character <= 'Z') && !(character >= '0' && character <= '9') && character != '_' && character != '-' && character != '.'
	}) >= 0 {
		return "INVALID"
	}
	return value
}
