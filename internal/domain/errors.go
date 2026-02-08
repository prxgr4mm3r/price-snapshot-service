package domain

import "errors"

var (
	// Symbol errors
	ErrInvalidSymbol  = errors.New("invalid symbol format")
	ErrSymbolNotFound = errors.New("symbol not found")
	ErrSymbolExists   = errors.New("symbol already exists")

	// Snapshot errors
	ErrSnapshotNotFound = errors.New("snapshot not found")
	ErrNoSnapshots      = errors.New("no snapshots available")

	// Exchange errors
	ErrExchangeUnavailable = errors.New("exchange service unavailable")
	ErrRateLimited         = errors.New("rate limited by exchange")
	ErrInvalidResponse     = errors.New("invalid response from exchange")

	// Database errors
	ErrDatabaseConnection = errors.New("database connection error")
	ErrDatabaseQuery      = errors.New("database query error")

	// General errors
	ErrInternal = errors.New("internal server error")
)

// DomainError wraps domain errors with additional context
type DomainError struct {
	Err     error
	Message string
	Code    string
}

func (e *DomainError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewDomainError creates a new domain error with context
func NewDomainError(err error, message, code string) *DomainError {
	return &DomainError{
		Err:     err,
		Message: message,
		Code:    code,
	}
}

// IsDomainError checks if the error is a domain error
func IsDomainError(err error) bool {
	var domainErr *DomainError
	return errors.As(err, &domainErr)
}
