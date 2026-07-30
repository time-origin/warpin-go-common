package session

import (
	"github.com/gorilla/sessions"
)

// NewStore 根据配置创建一个完全配置好的 gorilla/sessions.Store
// Store 接口是 gorilla/sessions 的核心接口
func NewStore(cfg Config) sessions.Store {
	// 1. 使用 Secret 初始化 CookieStore
	authKey := []byte(cfg.Secret)
	store := sessions.NewCookieStore(authKey)

	// 2. 配置 store.Options (封装了您在 Service 层想要做的配置)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   cfg.MaxAge,
		HttpOnly: cfg.HttpOnly,
		Secure:   cfg.Secure,
		SameSite: cfg.MapSameSiteString(), // 使用解析函数
	}

	return store
}
