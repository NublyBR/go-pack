package pack

import (
	"bytes"
	"crypto/rand"
	"reflect"
	"testing"
)

func TestFuzz(t *testing.T) {
	// This test is meant to find any uncaught panics that
	// might happen while decoding untrusted user data.

	type object struct {
		Ptr *[]*any
		Num map[int][2]complex64
	}

	var (
		data = make([]byte, 1024)

		buffer = bytes.NewBuffer(nil)

		buf [1]byte

		receivers = []reflect.Type{
			reflect.TypeOf(""),
			reflect.TypeOf(0),

			reflect.TypeOf([]any{}),
			reflect.TypeOf(map[string]any{}),
			reflect.TypeOf([5]any{}),

			reflect.TypeOf([]*any{}),
			reflect.TypeOf(map[string]*any{}),
			reflect.TypeOf([5]*any{}),

			reflect.TypeOf(object{}),
		}

		receiver any

		options = Options{
			SizeLimit: 2048,
		}

		packer = NewPacker(buffer, options)
	)

	for i := 0; i < 100_000; i++ {

		rand.Read(data)
		rand.Read(buf[:])

		receiver = reflect.New(receivers[int(buf[0])%len(receivers)]).Interface()

		err := Unmarshal(data, receiver, options)

		if err == nil {
			buffer.Reset()

			packer.Encode(receiver)
		}

	}

}
