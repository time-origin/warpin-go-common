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

func TestRegisterMessages(t *testing.T) {
	const code ErrCode = 900001
	const expected = "业务错误"

	if err := RegisterMessages(map[ErrCode]string{code: expected}); err != nil {
		t.Fatalf("register message: %v", err)
	}
	if actual := MapErrMsg(code); actual != expected {
		t.Fatalf("unexpected registered message: got %q want %q", actual, expected)
	}
	if !IsCodeErr(code) {
		t.Fatalf("registered code %d was not found", code)
	}
}

func TestRegisterMessagesRejectsConflictsAtomically(t *testing.T) {
	const newCode ErrCode = 900002
	err := RegisterMessages(map[ErrCode]string{
		RequestParamError: "不应覆盖",
		newCode:           "不应部分注册",
	})
	if err == nil {
		t.Fatal("expected duplicate registration to fail")
	}
	if IsCodeErr(newCode) {
		t.Fatalf("code %d was registered despite another conflict", newCode)
	}
	if actual := MapErrMsg(RequestParamError); actual != "请求内容无效" {
		t.Fatalf("existing message was overwritten: %q", actual)
	}
}

func TestRegisterMessagesRejectsEmptyMessage(t *testing.T) {
	if err := RegisterMessages(map[ErrCode]string{900003: "  "}); err == nil {
		t.Fatal("expected empty message registration to fail")
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
