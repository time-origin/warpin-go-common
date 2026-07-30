package jwt

import (
	jwt_go "github.com/golang-jwt/jwt/v5"
)

// CustomClaims 结构体定义了 JWT token 中包含的自定义信息。
type CustomClaims struct {
	UserID      string `json:"user_id"`
	AccountType int16  `json:"account_type"` // <--- 修改点
	jwt_go.RegisteredClaims
}
