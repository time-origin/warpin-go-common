package resx

import (
	"errors"
	"fmt"
	"testing"

	"github.com/time-origin/warpin-go-common/warpin-errors"
)

func TestNewErrorResult(t *testing.T) {
	t.Run("typed nil error", func(t *testing.T) {
		var err *errx.CodeError
		got := NewErrorResult(err)
		if got.Code != errx.ServerCommonError || got.Msg != errx.MapErrMsg(errx.ServerCommonError) {
			t.Fatalf("unexpected result: %#v", got)
		}
	})

	t.Run("business error", func(t *testing.T) {
		got := NewErrorResult(fmt.Errorf("authenticate: %w", errx.NewWithMsg(errx.Unauthorized, "token expired")))
		if got.Code != errx.Unauthorized || got.Msg != "token expired" {
			t.Fatalf("unexpected result: %#v", got)
		}
	})

	t.Run("unexpected error", func(t *testing.T) {
		got := NewErrorResult(errors.New("pq: password authentication failed"))
		if got.Code != errx.ServerCommonError || got.Msg != errx.MapErrMsg(errx.ServerCommonError) {
			t.Fatalf("unexpected result: %#v", got)
		}
	})
}
