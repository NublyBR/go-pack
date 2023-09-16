package pack

import "bytes"

func Marshal(data any) ([]byte, error) {
	var buf = bytes.NewBuffer(nil)

	err := NewPacker(buf).Encode(data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func Unmarshal(b []byte, data any) error {
	return NewUnpacker(bytes.NewBuffer(b)).Decode(data)
}
