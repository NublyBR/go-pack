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

		output int64
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

		written, err := writeVarInt(buf, input, b[:])
		if err != nil {
			t.Error(err)
		}

		read, err := readVarInt(buf, &output, b[:])
		if err != nil {
			t.Error(err)
		}

		if input != output {
			t.Errorf("Expected result from readVarInt(...) to be the same as input to writeVarInt(...): got %d, expected %d", output, input)
		}

		if written != read {
			t.Errorf("Expected bytes read by readVarInt(...) to be the same as bytes written by writeVarInt(...): got %d, expected %d", read, written)
		}
	}
}
