package domain

import (
	"strings"
	"time"
	"unicode"
)

// Symbol represents a tracked cryptocurrency symbol
type Symbol struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewSymbol creates a new symbol with validation
func NewSymbol(name string) (*Symbol, error) {
	name = strings.ToUpper(strings.TrimSpace(name))

	if err := ValidateSymbolName(name); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &Symbol{
		Name:      name,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// ValidateSymbolName validates the symbol name format
// Symbol names must be uppercase alphanumeric, between 2-20 characters
func ValidateSymbolName(name string) error {
	if name == "" {
		return ErrInvalidSymbol
	}

	if len(name) < 2 || len(name) > 20 {
		return ErrInvalidSymbol
	}

	for _, r := range name {
		if !unicode.IsUpper(r) && !unicode.IsDigit(r) {
			return ErrInvalidSymbol
		}
	}

	return nil
}

// Deactivate marks the symbol as inactive
func (s *Symbol) Deactivate() {
	s.Active = false
	s.UpdatedAt = time.Now().UTC()
}

// Activate marks the symbol as active
func (s *Symbol) Activate() {
	s.Active = true
	s.UpdatedAt = time.Now().UTC()
}
