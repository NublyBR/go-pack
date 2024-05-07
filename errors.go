package pack

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrInvalidPackedInt         = errors.New("invalid packed int")
	ErrInvalidPackedUint        = errors.New("invalid packed uint")
	ErrInvalidReceiver          = errors.New("invalid argument given to Decode, not a pointer type")
	ErrNil                      = errors.New("attempted to encode nil outside of pointer or interface context")
	ErrNilObject                = errors.New("may not encode nil in object mode")
	ErrMustBePointerToInterface = errors.New("in Objects mode, value given to Decode must be of type *interface{}")
)

type ErrNotDefined struct {
	typ reflect.Type
	oid uint
}

func (e *ErrNotDefined) Error() string {
	if e.typ != nil {
		return fmt.Sprintf("value not registered in Objects: %s", e.typ.String())
	}

	return fmt.Sprintf("id not registered in Objects: %d", e.oid)
}

type ErrInvalidType struct {
	typ reflect.Type
}

func (e *ErrInvalidType) Error() string {
	return fmt.Sprintf("invalid type provided: %q", e.typ.String())
}

type ErrInvalidTypeKey struct {
	typ reflect.Type
}

func (e *ErrInvalidTypeKey) Error() string {
	return fmt.Sprintf("invalid type provided for a map key: %q", e.typ.String())
}

type ErrDataTooLarge struct {
	typ reflect.Type

	max, size uint64
}

func (e *ErrDataTooLarge) Error() string {
	if e.typ == nil {
		return fmt.Sprintf("data exceeds maximum allowed size; max: %d, got: %d", e.max, e.size)
	}
	return fmt.Sprintf("value of type %q exceeds maximum allowed size; max: %d, got: %d", e.typ.String(), e.max, e.size)
}

type ErrCantUseInInterfaceMode struct {
	kind reflect.Kind
	typ  reflect.Type
}

func (e *ErrCantUseInInterfaceMode) Error() string {
	return fmt.Sprintf("cannot encode type %q in interface mode in %q", e.kind, e.typ.String())
}
