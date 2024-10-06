package asn1error

import (
	"fmt"
	"runtime/debug"
)

type ErrorType int

const (
	SyntaxError        ErrorType = iota // eg the framing is wrong or a class/tag is wrong
	StructuralError                     // eg the golang object did not match the asn1 type
	ImplmentationError                  // eg a feature is not implemented
	FutureImplementationError
	PanicError // eg a panic occurred
)

type Interface interface {
	error
	Type() ErrorType
}

type Unexpected[T any] struct {
	inner            error
	units            string
	errorType        ErrorType
	expected, actual T
}

func (e *Unexpected[T]) Error() string {
	if e.units == "" {
		return fmt.Sprintf("asn1: %s: expected=%v, actual=%v", e.inner.Error(), e.expected, e.actual)
	}
	return fmt.Sprintf("asn1: %s: expected=%v %s, actual=%v %s", e.inner.Error(), e.expected, e.units, e.actual, e.units)
}

func (e *Unexpected[T]) Unwrap() error {
	return e.inner
}

func (e *Unexpected[T]) WithUnits(units string) *Unexpected[T] {
	e.units = units
	return e
}

func (e *Unexpected[T]) Type() ErrorType {
	return e.errorType
}

func (e *Unexpected[T]) WithType(errorType ErrorType) *Unexpected[T] {
	e.errorType = errorType
	return e
}

func NewUnexpectedError[T any](expected, actual T, format string, args ...any) *Unexpected[T] {
	return &Unexpected[T]{
		inner:    fmt.Errorf(format, args...),
		expected: expected,
		actual:   actual,
	}
}

type UnsupportedGoType struct {
	GoTypeName string
}

func (e *UnsupportedGoType) Error() string {
	return fmt.Sprintf("asn1: unsupported Go type: %s", e.GoTypeName)
}

func (e *UnsupportedGoType) Type() ErrorType {
	return StructuralError
}

func NewUnimplementedError(format string, args ...any) *General {
	return NewErrorf("asn1: not implemented "+format, args...).WithType(ImplmentationError)
}

type General struct {
	inner error
	cause error
	eType ErrorType
	Stack string
}

func NewErrorf(format string, args ...any) *General {
	return &General{
		inner: fmt.Errorf(format, args...),
	}
}

func Wrap(err error) *General {
	return &General{
		inner: err,
	}
}

func (e *General) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s", e.inner.Error(), e.cause.Error())
	}
	return e.inner.Error()
}

func (e *General) Type() ErrorType {
	return e.eType
}

func (e *General) Unwrap() error {
	return e.inner
}

func (e *General) WithType(eType ErrorType) *General {
	e.eType = eType
	return e
}

func (e *General) WithCause(cause error) *General {
	e.cause = cause
	return e
}

func (e *General) WithStack() *General {
	e.Stack = string(debug.Stack())
	return e
}

func (e *General) TODO() *General {
	return e.WithType(FutureImplementationError).WithStack()
}

type List []error

func (el List) Error() string {
	if len(el) == 0 {
		return ""
	}
	if len(el) == 1 {
		return el[0].Error()
	}
	var s string
	for i, e := range el {
		if i > 0 {
			s += "; "
		}
		s += e.Error()
	}
	return s
}
