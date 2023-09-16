package pack

import "bytes"

// Pack object into bytes
func Marshal(data any, options ...Options) ([]byte, error) {
	var buf = bytes.NewBuffer(nil)

	err := NewPacker(buf, options...).Encode(data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Unpack object from bytes
func Unmarshal(b []byte, data any, options ...Options) error {
	return NewUnpacker(bytes.NewBuffer(b), options...).Decode(data)
}
