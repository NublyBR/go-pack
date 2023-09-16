package pack

type Options struct {
	// Set the Packer/Unpacker to work in Object Mode, in which it will only
	// be able to pack/unpack pre-defined types stored in given `Objects`,
	// by tagging them with object IDs that are defined in `Objects`.
	//
	// In this mode each top-level object packed will come with it's ID prepended
	// to it, so you may unpack it from a stream without needing to know what object
	// is being decoded beforehand.
	WithObjects Objects
}
