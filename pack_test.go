package pack

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestPack(t *testing.T) {
	type customSubType struct {
		Value string
	}

	type customType struct {
		String  string
		Value   interface{}
		Test    map[string]any
		Pointer *customSubType
	}

	type renamedInt int

	var (
		buf = bytes.NewBuffer(nil)

		packer   = NewPacker(buf).(*packer)
		unpacker = NewUnpacker(buf).(*unpacker)

		str = "string"

		sstr = &str

		inputs = []any{

			// bool
			false, true,

			// int, int8, int16, int32, int64
			int(123), int8(123), int16(123), int32(123), int64(123),

			// uint, uint8, uint16, uint32, uint64, uintptr
			uint(123), uint8(123), uint16(123), uint32(123), uint64(123), uintptr(123),

			// float32, float64
			float32(13.37), float64(13.37),

			// complex64, complex128
			complex64(complex(13, 37)), complex128(complex(13, 37)),

			// renamed int type
			renamedInt(1337), renamedInt(-1337_1337),

			// string
			"456", "", "тестирование юникода на всякий случай",

			// *string
			&str,

			// []byte
			[]byte("Hello, World!"),

			// []string
			[]string{"Hello", "World"},

			// [...]string
			[2]string{"Hello", "World"},

			// map[string]string
			map[string]string{
				"Hello": "World",
			},

			// Types that require type annotation

			// []any
			[]any{123, "abc", 45.6},

			// []any with *string and **string
			[]any{
				&str, &sstr,
			},

			// [3]any
			[3]any{123, "abc", 45.6},

			// map[string]any
			map[string]any{
				"a": 123,
				"b": "abc",
				"c": 45.6,
			},

			// [][]any
			[][]any{
				{1, 2, 3},
				{"4", "5", "6"},
			},

			// []any with []any and [...]any
			[]any{
				[]any{1, 2, 3},
				[3]any{"4", "5", "6"},
			},

			// map[string]any with []any and map[string]any
			map[string]any{
				"a": []any{1, 2.3, "4.5"},
				"b": []any{"4", "5", "6"},
				"c": map[string]any{
					"sub": "map",
				},
			},

			// Custom struct
			customType{
				String: "",
				Value:  123,
				Test:   map[string]any{},
				Pointer: &customSubType{
					Value: "Hello",
				},
			},

			// Custom struct with nil
			customType{
				String:  "",
				Value:   nil,
				Test:    nil,
				Pointer: nil,
			},
		}
	)

	for _, input := range inputs {
		buf.Reset()
		packer.written = 0
		unpacker.read = 0

		err := packer.encode(input, packerInfo{})
		if err != nil {
			t.Error(err)
		}

		content := make([]byte, len(buf.Bytes()))
		copy(content, buf.Bytes())

		// fmt.Printf("AFTER ENCODED: %x %q\n", buf.Bytes(), buf.Bytes())

		receiver := reflect.New(reflect.TypeOf(input))

		err = unpacker.decode(receiver.Interface(), packerInfo{})
		if err != nil {
			t.Error(err)
		}

		if !reflect.DeepEqual(input, receiver.Elem().Interface()) {
			t.Errorf("expected unpacker.Decode(...) to equal %+v, got %+v, buffer content:\n%q", input, receiver.Elem().Interface(), content)
		}

		if buf.Len() > 0 {
			t.Errorf("expected all bytes to be consumed after decode, got %d extra bytes: %q", buf.Len(), buf.Bytes())
		}

		if packer.written != unpacker.read {
			t.Errorf("expected bytes written to equal bytes read, written: %d / read: %d / actual: %d - %s",
				packer.written, unpacker.read, len(content), reflect.TypeOf(input).String())
		}

		// fmt.Printf("AFTER DECODED: %+v %x %q\n", input, buf.Bytes(), buf.Bytes())
	}

}

func TestIgnore(t *testing.T) {
	type structIgnore struct {
		Value string `pack:"ignore"`
	}

	var (
		src = structIgnore{
			Value: "This field should be ignored!",
		}

		dst structIgnore
	)

	data, err := Marshal(src)
	if err != nil {
		t.Error(err)
	}

	err = Unmarshal(data, &dst)
	if err != nil {
		t.Error(err)
	}

	if dst.Value != "" {
		t.Errorf("expected ignored field to be empty, got %q", dst.Value)
	}
}

func TestPackerLimit(t *testing.T) {

	type test struct {
		input  any
		expect *ErrDataTooLarge
	}

	var (
		options = Options{
			SizeLimit: 100,
		}
		buffer = bytes.NewBuffer(nil)

		tests = []test{
			{
				input:  make([]byte, 100),
				expect: &ErrDataTooLarge{max: 100, size: 101},
			},
			{
				input:  string(make([]byte, 100)),
				expect: &ErrDataTooLarge{max: 100, size: 101},
			},
			{
				input:  make([]int, 100),
				expect: &ErrDataTooLarge{max: 100, size: 101},
			},
			{
				input: func() any {
					mp := make(map[int]bool)
					for i := 0; i < 100; i++ {
						mp[i] = true
					}
					return mp
				}(),
				expect: &ErrDataTooLarge{max: 100, size: 101},
			},
			{
				input:  [101]int{},
				expect: &ErrDataTooLarge{max: 100, size: 101},
			},
		}
	)

	for _, test := range tests {
		p := NewPacker(buffer, options)

		err := p.Encode(test.input)

		if err == nil {
			t.Errorf("expected p.Encode(%s) to error, got nil", reflect.TypeOf(test.input).String())
			continue
		}

		if !reflect.DeepEqual(test.expect, err) {
			t.Errorf("expected p.Encode(%s) to equal %q, got %q", reflect.TypeOf(test.input).String(), test.expect, err)
		}
	}
}

func TestUnpackerLimit(t *testing.T) {

	type test struct {
		input    []byte
		receiver reflect.Value
		expect   *ErrDataTooLarge
	}

	var (
		options = Options{
			SizeLimit: 100,
		}

		tests = []test{
			{
				input:    []byte{100}, // varUint(100)
				receiver: reflect.New(reflect.SliceOf(reflect.TypeOf(byte(0)))),
				expect:   &ErrDataTooLarge{max: 100, size: 101},
			},
			{
				input:    []byte{100}, // varUint(100)
				receiver: reflect.New(reflect.TypeOf("")),
				expect:   &ErrDataTooLarge{max: 100, size: 101},
			},
			{
				input:    []byte{100}, // varUint(100)
				receiver: reflect.New(reflect.SliceOf(reflect.TypeOf(0))),
				expect:   &ErrDataTooLarge{max: 100, size: 101},
			},
			{
				input:    []byte{},
				receiver: reflect.New(reflect.ArrayOf(101, reflect.TypeOf(0))),
				expect:   &ErrDataTooLarge{max: 100, size: 101},
			},
			{
				input:    []byte{164, 1}, // varInt(100)
				receiver: reflect.New(reflect.MapOf(reflect.TypeOf(0), reflect.TypeOf(false))),
				expect:   &ErrDataTooLarge{max: 100, size: 102},
			},
		}
	)

	for _, test := range tests {
		buffer := bytes.NewBuffer(test.input)

		p := NewUnpacker(buffer, options)

		err := p.Decode(test.receiver.Interface())

		if err == nil {
			t.Errorf("expected u.Decode(%s) to error, got nil", test.receiver.String())
			continue
		}

		if !reflect.DeepEqual(test.expect, err) {
			t.Errorf("expected u.Decode(%s) to equal %q, got %q", test.receiver.String(), test.expect, err)
		}
	}
}

func BenchmarkPacker(b *testing.B) {
	type object struct {
		String string
		Int    int
		Float  float64
		Slice  []any
		Map    map[string]any
	}

	var (
		input = object{
			String: "Hello, World!",
			Int:    1337_1337,
			Float:  1337.1337,
			Slice:  []any{"Hello, World!", 1337_1337, 1337.1337},
			Map: map[string]any{
				"abc": 1337_1337,
				"def": 1337.1337,
			},
		}

		buffer = bytes.NewBuffer(nil)

		packer = NewPacker(buffer)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := packer.Encode(input); err != nil {
			b.Error(err)
		}

		buffer.Reset()
	}
}

func BenchmarkUnpacker(b *testing.B) {
	type object struct {
		String string
		Int    int
		Float  float64
		Slice  []any
		Map    map[string]any
	}

	var (
		input, _ = Marshal(object{
			String: "Hello, World!",
			Int:    1337_1337,
			Float:  1337.1337,
			Slice:  []any{"Hello, World!", 1337_1337, 1337.1337},
			Map: map[string]any{
				"abc": 1337_1337,
				"def": 1337.1337,
			},
		})

		output object

		buffer = bytes.NewReader(input)

		unpacker = NewUnpacker(buffer)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := unpacker.Decode(&output); err != nil {
			b.Error(err)
		}

		buffer.Seek(0, io.SeekStart)
	}
}
