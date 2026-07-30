package session

import (
	"context"
	"fmt"
)

// Session 接口定义，封装了 session 的常用方法
type Session interface {
	Get(key interface{}) interface{}
	Set(key interface{}, val interface{})
	Delete(key interface{})
	Save() error
}

// GetSessionFromContext 从 context 中获取 Session 实例
// 它会尝试从 context.Value 中获取名为 "session" 的值，并断言为 Session 接口类型。
func GetSessionFromContext(ctx context.Context) (Session, error) {
	// 尝试从 context.Value 中获取 Session 接口实现
	if s, ok := ctx.Value("session").(Session); ok {
		return s, nil
	}
	return nil, fmt.Errorf("session not found in context")
}
