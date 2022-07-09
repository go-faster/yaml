package yaml

import (
	"fmt"
	"reflect"
)

var _ = []interface {
	error
}{
	(*SyntaxError)(nil),
	(*UnmarshalError)(nil),
}

// SyntaxError is an error that occurs during parsing.
type SyntaxError struct {
	Line   int
	Offset int
	Msg    string
}

func syntaxErr(line, offset int, msgf string, args ...interface{}) error {
	return &SyntaxError{
		Line:   line,
		Offset: offset,
		Msg:    fmt.Sprintf(msgf, args...),
	}
}

// Error returns the error message.
func (s *SyntaxError) Error() string {
	if s.Line == 0 {
		return fmt.Sprintf("yaml: %s", s.Msg)
	}
	return fmt.Sprintf("yaml: line %d: %s", s.Line, s.Msg)
}

// UnknownFieldError reports an unknown field.
type UnknownFieldError struct {
	Field string
	Type  reflect.Type
}

// Error returns the error message.
func (d *UnknownFieldError) Error() string {
	return fmt.Sprintf("field %q not found in type %s", d.Field, d.Type)
}

func unknownFieldErr(field string, f *Node, typ reflect.Type) error {
	return &UnmarshalError{
		Node: f,
		Type: typ,
		Err:  &UnknownFieldError{Field: field, Type: typ},
	}
}

// DuplicateKeyError reports a duplicate key.
type DuplicateKeyError struct {
	First, Second *Node
}

func duplicateKeyErr(f, s *Node, typ reflect.Type) error {
	return &UnmarshalError{
		Node: f,
		Type: typ,
		Err:  &DuplicateKeyError{First: f, Second: s},
	}
}

// Error returns the error message.
func (d *DuplicateKeyError) Error() string {
	f, s := d.First, d.Second
	if s == nil {
		return fmt.Sprintf("duplicate key: %q", f.Value)
	}
	return fmt.Sprintf("mapping key %q already defined at line %d", s.Value, s.Line)
}

// UnmarshalError is an error that occurs during unmarshaling.
type UnmarshalError struct {
	Node *Node
	Type reflect.Type
	Err  error
}

func unmarshalErr(n *Node, typ reflect.Type, msgf string, args ...interface{}) error {
	return &UnmarshalError{
		Node: n,
		Type: typ,
		Err:  fmt.Errorf(msgf, args...),
	}
}

// Error returns the error message.
func (s *UnmarshalError) Error() string {
	n := s.Node
	if n == nil || n.Line == 0 {
		return fmt.Sprintf("yaml: %s", s.Err)
	}
	return fmt.Sprintf("yaml: line %d: %s", n.Line, s.Err)
}

// MarshalError is an error that occurs during marshaling.
type MarshalError struct {
	Msg string
}

// Error returns the error message.
func (s *MarshalError) Error() string {
	return fmt.Sprintf("yaml: %s", s.Msg)
}

// A TypeError is returned by Unmarshal when one or more fields in
// the YAML document cannot be properly decoded into the requested
// types. When this error is returned, the value is still
// unmarshaled partially.
type TypeError struct {
	Group error
}

// Unwrap returns the underlying error.
func (e *TypeError) Unwrap() error {
	return e.Group
}

// Error returns the error message.
func (e *TypeError) Error() string {
	return fmt.Sprintf("yaml: unmarshal errors:\n  %s", e.Group)
}
