package response

import (
	"errors"
	"github.com/go-chi/render"
	"github.com/time-origin/warpin-go-common/errors"
	"github.com/time-origin/warpin-go-common/http/result"
	"net/http"
)

// JSON sends a pre-constructed resx.Result object as a JSON response.
// The HTTP status is always 200 OK.
func JSON(w http.ResponseWriter, r *http.Request, result *resx.Result) {
	render.Status(r, http.StatusOK)
	render.JSON(w, r, result)
}

// Error intelligently handles an error, creates a standard failure result, and sends it as a JSON response.
// It checks if the error is of type *errx.CodeError to extract the business code and message.
// If not, it defaults to a generic server error.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	var codeErr *errx.CodeError

	if errors.As(err, &codeErr) {
		// The error is a CodeError, use its code and message
		JSON(w, r, resx.NewFailResult(codeErr.Code, codeErr.Message))
	} else {
		// Unexpected internal errors must never leak database, field, provider,
		// filesystem, or stack details through the public API.
		JSON(w, r, resx.NewFailResult(errx.ServerCommonError))
	}
}

// NoContent sends a response with no body and a 204 No Content status.
func NoContent(w http.ResponseWriter, r *http.Request) {
	render.NoContent(w, r)
}
