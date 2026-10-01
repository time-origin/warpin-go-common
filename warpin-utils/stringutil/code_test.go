package stringutil

import (
	"regexp"
	"testing"
)

func TestNormalizeOrGenerateCodeNormalizesSuppliedValue(t *testing.T) {
	code, err := NormalizeOrGenerateCode("  cms.menu.example-1  ")
	if err != nil {
		t.Fatalf("normalize code: %v", err)
	}
	if code != "CMS.MENU.EXAMPLE-1" {
		t.Fatalf("unexpected normalized code: %q", code)
	}
}

func TestNormalizeOrGenerateCodeGeneratesTwelveCharacterUppercaseAlphanumericValue(t *testing.T) {
	code, err := NormalizeOrGenerateCode("")
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	if !regexp.MustCompile(`^[A-Z0-9]{12}$`).MatchString(code) {
		t.Fatalf("generated code has invalid format: %q", code)
	}
}
