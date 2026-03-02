package exec_test

import (
	"context"
	"strings"
	"testing"

	tfexec "github.com/kybernetes-systems/terraform-provider-gastown/internal/exec"
)

func TestRunner_GT_version(t *testing.T) {
	r := tfexec.NewRunner("")
	out, err := r.GT(context.Background(), "version")
	if err != nil {
		t.Fatalf("GT version failed: %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("GT version returned empty output")
	}
}

func TestRunner_GT_nonzeroExitReturnsError(t *testing.T) {
	r := tfexec.NewRunner("")
	_, err := r.GT(context.Background(), "--no-such-flag-xyzzy")
	if err == nil {
		t.Fatal("expected error from invalid gt flag, got nil")
	}
}
