package exec

import (
	"errors"
	"fmt"
	"testing"
)

func TestNotFoundError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *NotFoundError
		expected string
	}{
		{
			name:     "with resource and name",
			err:      &NotFoundError{Resource: "rig", Name: "test-rig"},
			expected: "rig not found: test-rig",
		},
		{
			name:     "with resource only",
			err:      &NotFoundError{Resource: "crew"},
			expected: "crew not found",
		},
		{
			name:     "empty error",
			err:      &NotFoundError{},
			expected: " not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("NotFoundError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "direct NotFoundError",
			err:      &NotFoundError{Resource: "rig", Name: "test"},
			expected: true,
		},
		{
			name:     "wrapped NotFoundError",
			err:      fmt.Errorf("wrapped: %w", &NotFoundError{Resource: "crew"}),
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "string containing not found",
			err:      errors.New("rig not found"),
			expected: false, // Must be the typed error, not string matching
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.expected {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNotFound_ErrorAs(t *testing.T) {
	// Test that errors.As works correctly
	original := &NotFoundError{Resource: "rig", Name: "my-rig"}
	wrapped := fmt.Errorf("outer: %w", original)
	doubleWrapped := fmt.Errorf("outer: %w", wrapped)

	var notFound *NotFoundError
	if !errors.As(doubleWrapped, &notFound) {
		t.Error("errors.As should find NotFoundError through multiple wrapping layers")
	}

	if notFound.Resource != "rig" || notFound.Name != "my-rig" {
		t.Errorf("NotFoundError mismatch: got %+v, want Resource=rig, Name=my-rig", notFound)
	}
}
