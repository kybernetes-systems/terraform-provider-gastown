package exec

import (
	"errors"
	"fmt"
)

// NotFoundError indicates that a requested resource was not found.
// This is used for robust error handling instead of brittle string matching.
type NotFoundError struct {
	Resource string
	Name     string
}

func (e *NotFoundError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("%s not found: %s", e.Resource, e.Name)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// IsNotFound checks if an error is a NotFoundError.
func IsNotFound(err error) bool {
	var notFound *NotFoundError
	return errors.As(err, &notFound)
}
