package pack

import "reflect"

var canEncodeInInterface = map[reflect.Kind]bool{
	reflect.Bool:          true,
	reflect.Int:           true,
	reflect.Int8:          true,
	reflect.Int16:         true,
	reflect.Int32:         true,
	reflect.Int64:         true,
	reflect.Uint:          true,
	reflect.Uint8:         true,
	reflect.Uint16:        true,
	reflect.Uint32:        true,
	reflect.Uint64:        true,
	reflect.Uintptr:       true,
	reflect.Float32:       true,
	reflect.Float64:       true,
	reflect.Complex64:     true,
	reflect.Complex128:    true,
	reflect.Array:         true,
	reflect.Chan:          false,
	reflect.Func:          false,
	reflect.Interface:     true,
	reflect.Map:           true,
	reflect.Pointer:       true,
	reflect.Slice:         true,
	reflect.String:        true,
	reflect.Struct:        false,
	reflect.UnsafePointer: false,

	0xff: true, // special nil type
}
