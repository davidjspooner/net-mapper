package asn1core

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

type Error interface {
	error
	Type() ErrorType
}

type UnexpectedError[T any] struct {
	inner            error
	units            string
	errorType        ErrorType
	expected, actual T
}

func (e *UnexpectedError[T]) Error() string {
	if e.units == "" {
		return fmt.Sprintf("asn1: %s: expected=%v, actual=%v", e.inner.Error(), e.expected, e.actual)
	}
	return fmt.Sprintf("asn1: %s: expected=%v %s, actual=%v %s", e.inner.Error(), e.expected, e.units, e.actual, e.units)
}

func (e *UnexpectedError[T]) Unwrap() error {
	return e.inner
}

func (e *UnexpectedError[T]) WithUnits(units string) *UnexpectedError[T] {
	e.units = units
	return e
}

func (e *UnexpectedError[T]) Type() ErrorType {
	return e.errorType
}

func (e *UnexpectedError[T]) WithType(errorType ErrorType) *UnexpectedError[T] {
	e.errorType = errorType
	return e
}

func NewUnexpectedError[T any](expected, actual T, format string, args ...any) *UnexpectedError[T] {
	return &UnexpectedError[T]{
		inner:    fmt.Errorf(format, args...),
		expected: expected,
		actual:   actual,
	}
}

type UnsupportedGoTypeError struct {
	GoTypeName string
}

func (e *UnsupportedGoTypeError) Error() string {
	return fmt.Sprintf("asn1: unsupported Go type: %s", e.GoTypeName)
}

func (e *UnsupportedGoTypeError) Type() ErrorType {
	return StructuralError
}

func NewUnimplementedError(format string, args ...any) *GeneralError {
	return NewErrorf("asn1: not implemented "+format, args...).WithType(ImplmentationError)
}

type GeneralError struct {
	inner error
	cause error
	eType ErrorType
	Stack string
}

func NewErrorf(format string, args ...any) *GeneralError {
	return &GeneralError{
		inner: fmt.Errorf(format, args...),
	}
}

func (e *GeneralError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s", e.inner.Error(), e.cause.Error())
	}
	return e.inner.Error()
}

func (e *GeneralError) Type() ErrorType {
	return e.eType
}

func (e *GeneralError) Unwrap() error {
	return e.inner
}

func (e *GeneralError) WithType(eType ErrorType) *GeneralError {
	e.eType = eType
	return e
}

func (e *GeneralError) WithCause(cause error) *GeneralError {
	e.cause = cause
	return e
}

func (e *GeneralError) WithStack() *GeneralError {
	e.Stack = string(debug.Stack())
	return e
}

func (e *GeneralError) TODO() *GeneralError {
	return e.WithType(FutureImplementationError).WithStack()
}

type ErrorList []error

func (el ErrorList) Error() string {
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
