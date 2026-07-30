package jwt

import (
	"fmt"
	"time"

	jwt_go "github.com/golang-jwt/jwt/v5"
)

// GenerateToken 使用给定的 Claims 和配置生成一个 JWT 字符串。
// 它会根据配置中的过期时间设置 Claims 的 ExpiresAt 字段。
func GenerateToken(claims CustomClaims, cfg Config) (string, error) {
	// 如果 Claims 中没有手动设置过期时间，则根据配置设置
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.IsZero() {
		if cfg.Expire > 0 {
			duration := time.Duration(cfg.Expire) * time.Hour
			claims.ExpiresAt = jwt_go.NewNumericDate(time.Now().Add(duration))
		} // 如果 cfg.Expire 为 0，则不设置 ExpiresAt，表示永不过期
	}

	// 设置 Issuer
	if claims.Issuer == "" {
		claims.Issuer = cfg.Issuer
	}

	token := jwt_go.NewWithClaims(jwt_go.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

// ParseAndValidateToken 解析并验证给定的 JWT 字符串，如果有效则返回 CustomClaims。
func ParseAndValidateToken(tokenString string, cfg Config) (*CustomClaims, error) {
	claims := &CustomClaims{}
	token, err := jwt_go.ParseWithClaims(tokenString, claims, func(token *jwt_go.Token) (interface{}, error) { // Corrected type here
		if _, ok := token.Method.(*jwt_go.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse or validate token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
