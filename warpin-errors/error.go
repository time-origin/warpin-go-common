package errx

import "fmt"

// CodeError is a custom error type that includes a business error code.
type CodeError struct {
	Code    ErrCode
	Message string
}

// New creates a new CodeError with a given code and its default message.
func New(code ErrCode) *CodeError {
	return &CodeError{
		Code:    code,
		Message: MapErrMsg(code),
	}
}

// NewWithMsg creates a new CodeError with a given code and a custom message.
func NewWithMsg(code ErrCode, msg string) *CodeError {
	return &CodeError{
		Code:    code,
		Message: msg,
	}
}

// Error implements the error interface.
func (e *CodeError) Error() string {
	return fmt.Sprintf("Code: %d, Message: %s", e.Code, e.Message)
}
