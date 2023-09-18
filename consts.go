package pack

import "reflect"

type dataBuffer [10]byte

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

var kindToType = map[reflect.Kind]reflect.Type{
	reflect.Interface:  reflect.TypeOf([]any{}).Elem(),
	reflect.Bool:       reflect.TypeOf(false),
	reflect.Int:        reflect.TypeOf(int(0)),
	reflect.Int8:       reflect.TypeOf(int8(0)),
	reflect.Int16:      reflect.TypeOf(int16(0)),
	reflect.Int32:      reflect.TypeOf(int32(0)),
	reflect.Int64:      reflect.TypeOf(int64(0)),
	reflect.Uint:       reflect.TypeOf(uint(0)),
	reflect.Uint8:      reflect.TypeOf(uint8(0)),
	reflect.Uint16:     reflect.TypeOf(uint16(0)),
	reflect.Uint32:     reflect.TypeOf(uint32(0)),
	reflect.Uint64:     reflect.TypeOf(uint64(0)),
	reflect.Uintptr:    reflect.TypeOf(uintptr(0)),
	reflect.Float32:    reflect.TypeOf(float32(0)),
	reflect.Float64:    reflect.TypeOf(float64(0)),
	reflect.Complex64:  reflect.TypeOf(complex64(complex(0, 0))),
	reflect.Complex128: reflect.TypeOf(complex128(complex(0, 0))),
	reflect.String:     reflect.TypeOf(""),
	0xff:               nil,
}
