package xerr

import "fmt"

// Error represents a typed business error with a dedicated code and message.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Message
}

// New constructs a new typed business error.
func New(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

// WithMessage clones the error with a different message.
func (e *Error) WithMessage(message string) *Error {
	if e == nil {
		return nil
	}
	clone := *e
	clone.Message = message
	return &clone
}

// Format enables formatted output for the error value.
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v', 's':
		fmt.Fprint(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}
