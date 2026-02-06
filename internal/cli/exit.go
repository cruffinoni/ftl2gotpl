package cli

import "fmt"

const (
	ExitCodeSuccess          = 0
	ExitCodeConversionFailed = 2
	ExitCodeValidationFailed = 3
)

// ExitError carries a process exit code while preserving wrapped error context.
type ExitError struct {
	Code int
	Err  error
}

// Error returns the wrapped error message or a fallback message with exit code.
func (e *ExitError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("process failed with exit code %d", e.Code)
	}
	return e.Err.Error()
}

// Unwrap exposes the wrapped root error.
func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// newExitError builds an ExitError returned from CLI execution paths.
func newExitError(code int, err error) error {
	return &ExitError{
		Code: code,
		Err:  err,
	}
}
