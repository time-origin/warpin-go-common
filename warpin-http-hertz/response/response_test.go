package response

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/time-origin/warpin-go-common/warpin-errors"
	"github.com/time-origin/warpin-go-common/warpin-http/result"
)

func TestSuccess(t *testing.T) {
	c := app.NewContext(0)

	Success(c, map[string]string{"id": "agent-1"})

	if c.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("status = %d", c.Response.StatusCode())
	}
	var got resx.Result
	if err := json.Unmarshal(c.Response.Body(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Code != errx.Success {
		t.Fatalf("code = %d", got.Code)
	}
}

func TestUnexpectedErrorDoesNotLeakInternalMessage(t *testing.T) {
	c := app.NewContext(0)

	Error(c, errors.New("redis: connection refused"))

	var got resx.Result
	if err := json.Unmarshal(c.Response.Body(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Code != errx.ServerCommonError || got.Msg != errx.MapErrMsg(errx.ServerCommonError) {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestNoContent(t *testing.T) {
	c := app.NewContext(0)
	c.Response.SetBodyString("previous response body")

	NoContent(c)

	if c.Response.StatusCode() != consts.StatusNoContent {
		t.Fatalf("status = %d", c.Response.StatusCode())
	}
	if len(c.Response.Body()) != 0 {
		t.Fatalf("body = %q", c.Response.Body())
	}
}

func TestJSONPreservesApplicationContract(t *testing.T) {
	c := app.NewContext(0)
	payload := map[string]any{"code": 10000003, "message": "forbidden"}

	JSON(c, consts.StatusForbidden, payload)

	if c.Response.StatusCode() != consts.StatusForbidden {
		t.Fatalf("status = %d", c.Response.StatusCode())
	}
	var got map[string]any
	if err := json.Unmarshal(c.Response.Body(), &got); err != nil {
		t.Fatal(err)
	}
	if got["code"] != float64(10000003) || got["message"] != "forbidden" || len(got) != 2 {
		t.Fatalf("application contract changed: %#v", got)
	}
}
