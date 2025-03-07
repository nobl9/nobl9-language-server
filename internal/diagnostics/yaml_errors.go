package diagnostics

import (
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/token"
	"github.com/pkg/errors"
)

// tokenScopedError represents an error associated with a specific [token.Token].
type tokenScopedError struct {
	// Msg is the underlying error message.
	Msg string
	// Token is the [token.Token] associated with this error.
	Token *token.Token
	// err is the underlying, unwrapped error.
	err error
}

// Error implements the error interface.
// It returns the unwrapped error returned by go-yaml.
func (s tokenScopedError) Error() string {
	return s.err.Error()
}

// asYAMLTokenScopedError checks if the error is associated with a specific token.
// If so, it returns
// Otherwise, it returns nil.
func asYAMLTokenScopedError(err error) *tokenScopedError {
	var syntaxError *yaml.SyntaxError
	if errors.As(err, &syntaxError) {
		return &tokenScopedError{
			Msg:   syntaxError.Message,
			Token: syntaxError.Token,
			err:   err,
		}
	}
	var typeError *yaml.TypeError
	if errors.As(err, &typeError) {
		return &tokenScopedError{
			Msg:   typeError.Error(),
			Token: typeError.Token,
			err:   err,
		}
	}
	return nil
}
