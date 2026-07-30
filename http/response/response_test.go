package response

import (
	"encoding/json"
	"errors"
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
