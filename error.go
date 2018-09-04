package crawly

import "strings"

type errorString struct {
	Msg string
}

// NewError returns a simple implementation of error
func NewError(msg string) *errorString {
	return &errorString{Msg: msg}
}

func (e errorString) Error() string {
	return e.Msg
}

func (e errorString) MarshalBinary() ([]byte, error) {
	return []byte(strings.Join([]string{"\"", e.Msg, "\""}, "")), nil
}
