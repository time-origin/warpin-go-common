// File: pkg/accessControl/session/config.go

package session

import (
	"net/http"
)

// Config 结构体用于存储会话相关的配置信息。
type Config struct {
	// 映射 [security.session] 中的 secret 字段。
	Secret string `mapstructure:"secret"`
	// Cookie 选项
	MaxAge   int    `mapstructure:"maxAge"`   // Cookie 有效期（秒）
	HttpOnly bool   `mapstructure:"httpOnly"` // 防止客户端 JS 访问
	Secure   bool   `mapstructure:"secure"`   // 仅在 HTTPS 下发送
	Domain   string `mapstructure:"domain"`   // Cookie 的作用域
	// SameSite 字段使用字符串映射，稍后在 NewStore 中解析
	SameSite string `mapstructure:"sameSite"`
}

// MapSameSiteString converts string to http.SameSite enum
func (c *Config) MapSameSiteString() http.SameSite {
	switch c.SameSite {
	case "Strict":
		return http.SameSiteStrictMode
	case "Lax":
		return http.SameSiteLaxMode
	case "None":
		// 需要 Secure=true 才能使用 SameSite=None
		return http.SameSiteNoneMode
	default:
		// 默认使用 LaxMode (兼容性最好的选择)
		return http.SameSiteLaxMode
	}
}
