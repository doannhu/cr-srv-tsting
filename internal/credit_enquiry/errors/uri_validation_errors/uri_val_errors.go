package uri_validation_errors

import (
	"fmt"
)

const (
	ErrInvalidScheme    = "INVALID_SCHEME"
	ErrInvalidAuthority = "INVALID_AUTHORITY"
	ErrInvalidUserinfo  = "INVALID_USERINFO"
	ErrInvalidHost      = "INVALID_HOST"
	ErrInvalidPort      = "INVALID_PORT"
	ErrInvalidPath      = "INVALID_PATH"
	ErrInvalidQuery     = "INVALID_QUERY"
	ErrInvalidFragment  = "INVALID_FRAGMENT"
	ErrInvalidPercent   = "INVALID_PERCENT"
)

// Error describes a validation failure.
type Error struct {
	Code  string // "INVALID_SCHEME", …
	Field string // component name
	Msg   string // detail
}

func (e *Error) Error() string {
	return fmt.Sprintf("invalid URI %s: %s", e.Field, e.Msg)
}
