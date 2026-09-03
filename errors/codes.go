package errx

import (
	"fmt"
	"strings"
	"sync"
)

// ErrCode defines a type for custom error codes.
type ErrCode int

const (
	// Common error codes
	Success            ErrCode = 200 // 成功 (修改为 200 以匹配 HTTP 状态码)
	RequestParamError  ErrCode = 400 // 请求参数错误 (通用)
	InvalidRequest     ErrCode = 400 // 无效请求参数 (更具体)
	BadRequest         ErrCode = 400 // 错误的请求 (更具体)
	Unauthorized       ErrCode = 401 // 未授权
	Forbidden          ErrCode = 403 // 禁止访问
	NotFound           ErrCode = 404 // 资源未找到
	ServerCommonError  ErrCode = 500 // 服务器通用错误
	ServiceUnavailable ErrCode = 503 // 服务不可用

	// Data access error codes
	RecordNotFound ErrCode = 100001 // 记录未找到

	// Specific error codes from errMsg.go
	TokenExpireError          ErrCode = 10001 // Token expired, please log in again
	TokenGenerateError        ErrCode = 10002 // Failed to generate token
	DbError                   ErrCode = 10003 // Database busy, please try again later
	DbUpdateAffectedZeroError ErrCode = 10004 // Database update affected zero rows
	NotExist                  ErrCode = 10005 // Data does not exist
	IncorrectPwd              ErrCode = 10006 // Incorrect password
	IncorrectAccount          ErrCode = 10007 // User does not exist
	IncorrectVerifyCode       ErrCode = 10008 // Incorrect verification code
	UploadFileEmpty           ErrCode = 10009 // Uploaded file is empty
	FileNotExist              ErrCode = 10010 // File does not exist
	CascadeDataExist          ErrCode = 10011 // Cascade data exists
	IncorrectConfig           ErrCode = 10012 // Configuration error
	SecurityError             ErrCode = 10013
	PermissionDenied          ErrCode = 10014 // 权限不足

)

var (
	messageMu sync.RWMutex
	message   = map[ErrCode]string{
		Success:                   "SUCCESS",
		ServerCommonError:         "系统开小差啦，请稍后尝试",
		RequestParamError:         "请求内容无效",
		Unauthorized:              "请先登录",
		Forbidden:                 "无权执行此操作",
		NotFound:                  "请求的内容不存在",
		ServiceUnavailable:        "服务暂时不可用，请稍后再试",
		RecordNotFound:            "请求的记录不存在",
		TokenExpireError:          "登录状态已过期，请重新登录",
		TokenGenerateError:        "登录凭证生成失败",
		DbError:                   "系统开小差啦，请稍后尝试",
		DbUpdateAffectedZeroError: "数据未发生变更",
		NotExist:                  "请求的数据不存在",
		IncorrectPwd:              "账号或密码错误",
		IncorrectAccount:          "用户不存在",
		IncorrectVerifyCode:       "验证码错误",
		UploadFileEmpty:           "上传文件不能为空",
		FileNotExist:              "文件不存在",
		CascadeDataExist:          "存在关联数据，暂时无法操作",
		IncorrectConfig:           "系统配置错误",
		SecurityError:             "安全校验失败",
		PermissionDenied:          "无权执行此操作",
	}
)

// RegisterMessages adds application-owned error messages to the shared catalog.
// Registration is atomic and rejects codes that are already defined.
func RegisterMessages(messages map[ErrCode]string) error {
	messageMu.Lock()
	defer messageMu.Unlock()

	for code, msg := range messages {
		if strings.TrimSpace(msg) == "" {
			return fmt.Errorf("error code %d has an empty message", code)
		}
		if _, exists := message[code]; exists {
			return fmt.Errorf("error code %d is already registered", code)
		}
	}
	for code, msg := range messages {
		message[code] = msg
	}
	return nil
}

// MapErrMsg maps an ErrCode to its corresponding message.
func MapErrMsg(errCode ErrCode) string {
	messageMu.RLock()
	defer messageMu.RUnlock()

	if msg, ok := message[errCode]; ok {
		return msg
	} else {
		return "系统开小差啦，请稍后尝试"
	}
}

// IsCodeErr checks if an ErrCode is defined in the message map.
func IsCodeErr(errCode ErrCode) bool {
	messageMu.RLock()
	defer messageMu.RUnlock()

	_, ok := message[errCode]
	return ok
}
