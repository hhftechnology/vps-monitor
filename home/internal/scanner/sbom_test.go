package scanner

import (
	"errors"
	"testing"
)

func TestSyftStreamFailureCauseReturnsError(t *testing.T) {
	cause := syftStreamFailureCause(syftStreamResult{
		stderr: "warning on stderr",
		err:    errors.New("read failure"),
	})
	if cause == nil {
		t.Fatal("expected non-nil stream failure cause")
	}
	if got := cause.(error).Error(); got != "read failure" {
		t.Fatalf("expected error cause, got %q", got)
	}
}

func TestSyftStreamFailureCauseReturnsTrimmedStderr(t *testing.T) {
	cause := syftStreamFailureCause(syftStreamResult{
		stderr: "  syft reported an error \n",
		err:    nil,
	})
	if cause == nil {
		t.Fatal("expected non-nil stream failure cause")
	}
	got, ok := cause.(string)
	if !ok {
		t.Fatalf("expected string cause from stderr, got %T", cause)
	}
	if got != "syft reported an error" {
		t.Fatalf("expected trimmed stderr cause, got %q", got)
	}
}

func TestSyftStreamFailureCauseReturnsNilOnCleanStream(t *testing.T) {
	cause := syftStreamFailureCause(syftStreamResult{
		stderr: " \n\t ",
		err:    nil,
	})
	if cause != nil {
		t.Fatalf("expected nil cause for clean stream, got %v", cause)
	}
}
