package pack

import (
	"bytes"
	"testing"
)

func TestVarInt(t *testing.T) {

	t.Parallel()

	var (
		b dataBuffer

		buf = bytes.NewBuffer(nil)

		inputs = []int64{
			0x00, 0x01, 0x7f, 0x80, -0x01, -0x7f, -0x80,
			0xff, 0xffff, 0xffffff, 0xffffffff, -0xff, -0xffff, -0xffffff, -0xffffffff,

			0x7fffffffffffffff, -0x7fffffffffffffff,

			0xDEADBEEF, 0xC0FFEE, 0xCAFEBABE, 0xDEADC0DE,
			-0xDEADBEEF, -0xC0FFEE, -0xCAFEBABE, -0xDEADC0DE,
		}
	)

	for _, input := range inputs {

		buf.Reset()

		expectedSize := SizeVarInt(input)

		written, err := WriteVarInt(buf, input, b[:])
		if err != nil {
			t.Error(err)
		}

		if written != expectedSize {
			t.Errorf("Expected bytes written by WriteVarInt(%d) to be %d, got %d", input, expectedSize, written)
		}

		output, getRead, err := GetVarInt(buf.Bytes())
		if err != nil {
			t.Error(err)
		}

		if input != output {
			t.Errorf("Expected result from GetVarInt(%d) to be the same as input to WriteVarInt(...): got %d, expected %d", input, output, input)
		}

		if written != getRead {
			t.Errorf("Expected bytes read by GetVarInt(%d) to be the same as bytes written by WriteVarInt(...): got %d, expected %d", input, getRead, written)
		}

		read, err := ReadVarInt(buf, &output, b[:])
		if err != nil {
			t.Error(err)
		}

		if input != output {
			t.Errorf("Expected result from ReadVarInt(%d) to be the same as input to WriteVarInt(...): got %d, expected %d %+v", input, output, input, b[:written])
		}

		if written != read {
			t.Errorf("Expected bytes read by ReadVarInt(%d) to be the same as bytes written by WriteVarInt(...): got %d, expected %d", input, read, written)
		}
	}
}
