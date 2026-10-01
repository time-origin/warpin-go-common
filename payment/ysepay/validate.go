package ysepay

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const requestIDAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_-"

func generateRequestID(random io.Reader, now time.Time) (string, error) {
	prefix := make([]byte, 20)
	buffer := make([]byte, 20)
	if _, err := io.ReadFull(random, buffer); err != nil {
		return "", fmt.Errorf("%w: cannot generate request ID", ErrCrypto)
	}
	for index, value := range buffer {
		prefix[index] = requestIDAlphabet[int(value)%len(requestIDAlphabet)]
	}
	return string(prefix) + now.Format("060102150405"), nil
}

func validRequestID(value string) bool {
	if len(value) < 14 || len(value) > 32 {
		return false
	}
	for _, character := range value {
		if !strings.ContainsRune(requestIDAlphabet, character) {
			return false
		}
	}
	_, err := time.Parse("060102150405", value[len(value)-12:])
	return err == nil
}

// YuanToFen converts a non-negative plain decimal yuan value to integer fen.
// It intentionally rejects scientific notation and rounding.
func YuanToFen(value string) (int64, error) {
	if value == "" || strings.TrimSpace(value) != value || strings.HasPrefix(value, "+") || strings.HasPrefix(value, "-") || strings.ContainsAny(value, "eE") {
		return 0, fmt.Errorf("%w: invalid decimal amount", ErrInvalidResponse)
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || (len(parts) == 2 && (parts[1] == "" || len(parts[1]) > 2)) {
		return 0, fmt.Errorf("%w: invalid decimal amount", ErrInvalidResponse)
	}
	for _, part := range parts {
		for _, character := range part {
			if character < '0' || character > '9' {
				return 0, fmt.Errorf("%w: invalid decimal amount", ErrInvalidResponse)
			}
		}
	}
	yuan, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || yuan > math.MaxInt64/100 {
		return 0, fmt.Errorf("%w: decimal amount overflows", ErrInvalidResponse)
	}
	fen := int64(0)
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
		fen, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: invalid decimal amount", ErrInvalidResponse)
		}
	}
	if yuan > (math.MaxInt64-fen)/100 {
		return 0, fmt.Errorf("%w: decimal amount overflows", ErrInvalidResponse)
	}
	return yuan*100 + fen, nil
}

func validShopDate(value string) bool {
	if len(value) != 8 {
		return false
	}
	parsed, err := time.Parse("20060102", value)
	return err == nil && parsed.Format("20060102") == value
}

func validLength(value string, minimum, maximum int) bool {
	length := utf8.RuneCountInString(value)
	return length >= minimum && length <= maximum
}

func validOptionalRoutingValue(value string) bool {
	return value == "" || (strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n"))
}
