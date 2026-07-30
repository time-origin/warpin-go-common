package errx

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

	// Workflow specific error codes
	RecordNotFound        ErrCode = 100001 // 记录未找到
	StatusTransitionError ErrCode = 100002 // 状态转换错误

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

	// System module
	SysEmpNotExist ErrCode = 20001 // Employee does not exist

	// User module
	UserExist ErrCode = 30001 // User already exists

	// Business specific error codes (VoiceCraft)
	QueueFull           ErrCode = 300001 // 排队人数过多，暂时熔断
	BalanceInsufficient ErrCode = 300002 // 积分不足
	PromptSensitive     ErrCode = 400001 // 包含敏感词
	ProviderError       ErrCode = 500001 // 上游 API 挂了
)

var message map[ErrCode]string

func init() {
	message = make(map[ErrCode]string)
	// Global error codes
	message[Success] = "SUCCESS"
	message[ServerCommonError] = "系统开小差啦，请稍后尝试"
	message[RequestParamError] = "请求参数错误"
	message[InvalidRequest] = "请求参数无效"
	message[BadRequest] = "请求内容无效"
	message[Unauthorized] = "请先登录"
	message[Forbidden] = "无权执行此操作"
	message[NotFound] = "请求的内容不存在"
	message[ServiceUnavailable] = "服务暂时不可用，请稍后再试"

	// Workflow specific error codes
	message[RecordNotFound] = "请求的记录不存在"
	message[StatusTransitionError] = "当前状态不允许此操作"

	// Specific error codes
	message[TokenExpireError] = "登录状态已过期，请重新登录"
	message[TokenGenerateError] = "登录凭证生成失败"
	message[DbError] = "系统开小差啦，请稍后尝试"
	message[DbUpdateAffectedZeroError] = "数据未发生变更"
	message[NotExist] = "请求的数据不存在"
	message[IncorrectPwd] = "账号或密码错误"
	message[IncorrectAccount] = "用户不存在"
	message[IncorrectVerifyCode] = "验证码错误"
	message[UploadFileEmpty] = "上传文件不能为空"
	message[FileNotExist] = "文件不存在"
	message[CascadeDataExist] = "存在关联数据，暂时无法操作"
	message[IncorrectConfig] = "系统配置错误"
	message[SecurityError] = "安全校验失败"
	message[PermissionDenied] = "无权执行此操作"

	// System module
	message[SysEmpNotExist] = "员工不存在"

	// User module
	message[UserExist] = "用户已存在"

	// Business specific error codes (VoiceCraft)
	message[QueueFull] = "当前排队人数较多，请稍后再试"
	message[BalanceInsufficient] = "能量不足"
	message[PromptSensitive] = "创作内容包含不适宜信息，请修改后重试"
	message[ProviderError] = "创作服务暂时不可用，请稍后再试"
}

// MapErrMsg maps an ErrCode to its corresponding message.
func MapErrMsg(errCode ErrCode) string {
	if msg, ok := message[errCode]; ok {
		return msg
	} else {
		return "系统开小差啦，请稍后尝试"
	}
}

// IsCodeErr checks if an ErrCode is defined in the message map.
func IsCodeErr(errCode ErrCode) bool {
	_, ok := message[errCode]
	return ok
}
