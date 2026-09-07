package response

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/time-origin/warpin-go-common/http/result"
)

// JSON sends a pre-constructed resx.Result object as a JSON response.
// The HTTP status is always 200 OK. The request parameter remains for
// compatibility with existing callers.
func JSON(w http.ResponseWriter, _ *http.Request, result *resx.Result) {
	if committed, err := writeJSON(w, http.StatusOK, result); err != nil && !committed {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// WriteJSON writes any JSON response using only the standard net/http API.
func WriteJSON(w http.ResponseWriter, status int, value any) error {
	_, err := writeJSON(w, status, value)
	return err
}

// writeJSON reports whether headers were committed, so callers never attempt a
// second response after a network write failure.
func writeJSON(w http.ResponseWriter, status int, value any) (bool, error) {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(value); err != nil {
		return false, err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write(body.Bytes())
	return true, err
}

// Error intelligently handles an error, creates a standard failure result, and sends it as a JSON response.
// It checks if the error is of type *errx.CodeError to extract the business code and message.
// If not, it defaults to a generic server error.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	JSON(w, r, resx.NewErrorResult(err))
}

// NoContent sends a response with no body and a 204 No Content status.
func NoContent(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
