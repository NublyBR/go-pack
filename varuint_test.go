package pack

import (
	"bytes"
	"testing"
)

func TestVarUint(t *testing.T) {

	t.Parallel()

	var (
		b dataBuffer

		buf = bytes.NewBuffer(nil)

		inputs = []uint64{
			0x00, 0x01, 0x7f, 0x80,
			0xff, 0xffff, 0xffffff, 0xffffffff,

			0x7fffffffffffffff, 0xffffffffffffffff,

			0xDEADBEEF, 0xC0FFEE, 0xCAFEBABE, 0xDEADC0DE,
		}
	)

	for _, input := range inputs {

		buf.Reset()

		expectedSize := SizeVarUint(input)

		written, err := WriteVarUint(buf, input, b[:])
		if err != nil {
			t.Error(err)
		}

		if written != expectedSize {
			t.Errorf("Expected bytes written by WriteVarUint(%d) to be %d, got %d", input, expectedSize, written)
		}

		output, getRead, err := GetVarUint(buf.Bytes())
		if err != nil {
			t.Error(err)
		}

		if input != output {
			t.Errorf("Expected result from GetVarUint(%d) to be the same as input to WriteVarUint(...): got %d, expected %d", input, output, input)
		}

		if written != getRead {
			t.Errorf("Expected bytes read by GetVarUint(%d) to be the same as bytes written by WriteVarUint(...): got %d, expected %d", input, getRead, written)
		}

		read, err := ReadVarUint(buf, &output, b[:])
		if err != nil {
			t.Error(err)
		}

		if input != output {
			t.Errorf("Expected result from ReadVarUint(...) to be the same as input to WriteVarUint(...): got %d, expected %d", output, input)
		}

		if written != read {
			t.Errorf("Expected bytes read by ReadVarUint(...) to be the same as bytes written by WriteVarUint(...): got %d, expected %d", read, written)
		}
	}
}
