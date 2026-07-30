package errx

import (
	"strings"
	"testing"
	"unicode"
)

func TestPublicErrorMessagesAreChinese(t *testing.T) {
	codes := []ErrCode{
		RequestParamError,
		Unauthorized,
		Forbidden,
		NotFound,
		ServerCommonError,
		ServiceUnavailable,
		RecordNotFound,
		StatusTransitionError,
		TokenExpireError,
		TokenGenerateError,
		DbError,
		DbUpdateAffectedZeroError,
		NotExist,
		IncorrectPwd,
		IncorrectAccount,
		IncorrectVerifyCode,
		UploadFileEmpty,
		FileNotExist,
		CascadeDataExist,
		IncorrectConfig,
		SecurityError,
		PermissionDenied,
		SysEmpNotExist,
		UserExist,
		QueueFull,
		BalanceInsufficient,
		PromptSensitive,
		ProviderError,
	}

	for _, code := range codes {
		message := MapErrMsg(code)
		if strings.TrimSpace(message) == "" {
			t.Fatalf("error code %d has an empty public message", code)
		}
		if !containsHan(message) {
			t.Fatalf("error code %d exposes a non-Chinese message: %q", code, message)
		}
	}
}

func TestUnknownErrorCodeUsesStableChineseMessage(t *testing.T) {
	const expected = "系统开小差啦，请稍后尝试"
	if actual := MapErrMsg(ErrCode(-1)); actual != expected {
		t.Fatalf("unexpected unknown-code message: got %q want %q", actual, expected)
	}
}

func containsHan(value string) bool {
	for _, char := range value {
		if unicode.Is(unicode.Han, char) {
			return true
		}
	}
	return false
}
