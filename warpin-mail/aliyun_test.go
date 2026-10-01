package mail

import "testing"

func TestNewAliyunMailerRequiresCredentials(t *testing.T) {
	t.Parallel()

	if _, err := NewAliyunMailer(AliyunConfig{}); err == nil {
		t.Fatal("expected missing credentials to be rejected")
	}
}
