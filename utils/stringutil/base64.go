package stringutil

import (
	"encoding/base64"
)

// Base64Encode 将字符串进行 Base64 加密
func Base64Encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// Base64Decode 将 Base64 字符串进行解密
func Base64Decode(str string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}
	return string(decodedBytes), nil
}
