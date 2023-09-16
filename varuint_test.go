package pack

import (
	"bytes"
	"testing"
)

func TestVarUint(t *testing.T) {
	var (
		buf = bytes.NewBuffer(nil)

		output uint64
		inputs = []uint64{
			0x00, 0x01, 0x7f, 0x80,
			0xff, 0xffff, 0xffffff, 0xffffffff,

			0x7fffffffffffffff, 0xffffffffffffffff,

			0xDEADBEEF, 0xC0FFEE, 0xCAFEBABE, 0xDEADC0DE,
		}
	)

	for _, input := range inputs {

		buf.Reset()

		written, err := writeVarUint(buf, input)
		if err != nil {
			t.Error(err)
		}

		read, err := readVarUint(buf, &output)
		if err != nil {
			t.Error(err)
		}

		if input != output {
			t.Errorf("Expected result from readVarUint(...) to be the same as input to writeVarUint(...): got %d, expected %d", output, input)
		}

		if written != read {
			t.Errorf("Expected bytes read by readVarUint(...) to be the same as bytes written by writeVarUint(...): got %d, expected %d", read, written)
		}
	}
}
