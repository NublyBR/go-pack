package pack

import "reflect"

type BeforePack interface {
	// If a struct implements this interface, this function will be called
	// right before it is about to be packed, by returning an error it will
	// cause the packer to early return said error and stop further packing.
	BeforePack() error
}

var interfaceBeforePack = reflect.TypeOf((*BeforePack)(nil)).Elem()

type AfterUnpack interface {
	// If a struct implements this interface, this function will be called
	// right after it was unpacked, by returning an error it will cause the
	// unpacker to early return said error and stop further unpacking.
	AfterUnpack() error
}

var interfaceAfterUnpack = reflect.TypeOf((*AfterUnpack)(nil)).Elem()
