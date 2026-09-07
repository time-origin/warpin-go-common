// Package response adapts the framework-independent result envelope to Hertz.
package response

import (
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/time-origin/warpin-go-common/errors"
	"github.com/time-origin/warpin-go-common/http/result"
)

// JSON writes a native Hertz JSON response.
func JSON(c *app.RequestContext, status int, value any) {
	c.JSON(status, value)
}

// Success writes a successful result envelope.
func Success(c *app.RequestContext, data any) {
	JSON(c, consts.StatusOK, resx.NewResult(data, errx.Success))
}

// Error writes a failure result envelope without leaking unexpected errors.
func Error(c *app.RequestContext, err error) {
	JSON(c, consts.StatusOK, resx.NewErrorResult(err))
}

// NoContent writes a 204 response without a body.
func NoContent(c *app.RequestContext) {
	c.Status(consts.StatusNoContent)
}
