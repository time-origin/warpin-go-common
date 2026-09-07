package resx

import (
	"errors"

	"github.com/time-origin/warpin-go-common/errors"
)

// NewErrorResult maps public business errors to the shared result envelope and
// hides details from unexpected internal errors.
func NewErrorResult(err error) *Result {
	var codeErr *errx.CodeError
	if errors.As(err, &codeErr) {
		return NewFailResult(codeErr.Code, codeErr.Message)
	}
	return NewFailResult(errx.ServerCommonError)
}
