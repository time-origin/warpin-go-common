package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/time-origin/warpin-go-common/http/result"
)

func TestUnexpectedErrorDoesNotLeakInternalMessage(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	Error(recorder, request, errors.New("pq: column workflow_id does not exist"))

	var result resx.Result
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Msg != "系统开小差啦，请稍后尝试" {
		t.Fatalf("unexpected public message: %q", result.Msg)
	}
}

func TestWriteJSON(t *testing.T) {
	recorder := httptest.NewRecorder()

	if err := WriteJSON(recorder, http.StatusCreated, map[string]string{"id": "agent-1"}); err != nil {
		t.Fatalf("write response: %v", err)
	}
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
}

func TestJSONEncodingError(t *testing.T) {
	recorder := httptest.NewRecorder()

	JSON(recorder, httptest.NewRequest(http.MethodGet, "/", nil), &resx.Result{Data: make(chan int)})

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestNoContent(t *testing.T) {
	recorder := httptest.NewRecorder()

	NoContent(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

type failingWriter struct {
	header   http.Header
	statuses []int
	writes   int
	err      error
}

func (w *failingWriter) Header() http.Header       { return w.header }
func (w *failingWriter) WriteHeader(status int)    { w.statuses = append(w.statuses, status) }
func (w *failingWriter) Write([]byte) (int, error) { w.writes++; return 0, w.err }

func TestJSONDoesNotWriteAgainAfterNetworkFailure(t *testing.T) {
	w := &failingWriter{header: make(http.Header), err: errors.New("connection closed")}
	JSON(w, httptest.NewRequest(http.MethodGet, "/", nil), &resx.Result{})
	if len(w.statuses) != 1 || w.statuses[0] != http.StatusOK || w.writes != 1 {
		t.Fatalf("response written again after commit: statuses=%v writes=%d", w.statuses, w.writes)
	}
}

func TestWriteJSONReportsNetworkFailure(t *testing.T) {
	w := &failingWriter{header: make(http.Header), err: errors.New("connection closed")}
	if err := WriteJSON(w, http.StatusOK, "data"); !errors.Is(err, w.err) {
		t.Fatalf("write error lost: %v", err)
	}
}
