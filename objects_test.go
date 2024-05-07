package pack

import (
	"bytes"
	"reflect"
	"testing"
)

func TestObjects(t *testing.T) {

	t.Parallel()

	type objectA struct {
		Val string
	}

	type objectB2 struct {
		Name string
	}

	type objectB struct {
		Param int

		SubObject *objectB2
	}

	type objectC struct {
		Val   string
		Param int
	}

	type recursiveObject struct {
		Level      int
		AnotherOne *recursiveObject
	}

	var (
		options = Options{
			WithObjects: NewObjects(objectA{}, objectB{}, objectC{}, recursiveObject{}),
		}

		buf = bytes.NewBuffer(nil)

		packer   = NewPacker(buf, options)
		unpacker = NewUnpacker(buf, options)

		inputs = []any{
			objectA{Val: "Hello"},
			objectB{Param: 123, SubObject: &objectB2{"sub"}},
			objectC{Val: "World", Param: 456},
			&objectB{Param: 789, SubObject: &objectB2{"obj"}},
			&objectC{Val: "Test", Param: 987},
			&objectA{Val: "Another"},

			recursiveObject{
				Level: 1,
				AnotherOne: &recursiveObject{
					Level: 2,
					AnotherOne: &recursiveObject{
						Level:      3,
						AnotherOne: nil,
					},
				},
			},
		}
	)

	for _, input := range inputs {
		err := packer.Encode(input)
		if err != nil {
			t.Error(err)
		}
	}

	for _, input := range inputs {
		var output any

		err := unpacker.Decode(&output)
		if err != nil {
			t.Error(err)
		}

		inputType := reflect.TypeOf(input)
		expect := input
		if inputType.Kind() == reflect.Pointer {
			inputType = inputType.Elem()
			expect = reflect.ValueOf(input).Elem().Interface()
		}

		if reflect.TypeOf(output) != reflect.PointerTo(inputType) {
			t.Errorf("decoded object type should be a pointer to input object type; expected: %s, got: %s",
				reflect.PointerTo(inputType).String(), reflect.TypeOf(output).String())
		} else {
			output = reflect.ValueOf(output).Elem().Interface()

			if !reflect.DeepEqual(expect, output) {
				t.Errorf("decoded object should equal input object; expected: %+v, got: %+v", expect, output)
			}
		}
	}

	if buf.Len() > 0 {
		t.Errorf("expected all bytes to be consumed after decoding all objects, got %d extra bytes: %q", buf.Len(), buf.Bytes())
	}
}

func TestSubObjects(t *testing.T) {

	t.Parallel()

	type objectA struct {
		Val string
	}

	type objectB struct {
		Name string
	}

	type objectTest struct {
		RawA any `pack:"objects:a"`
		RawB any `pack:"objects:b"`

		Slice []any          `pack:"objects:b"`
		Array [2]any         `pack:"objects:b"`
		Map   map[string]any `pack:"objects:b"`
	}

	var (
		options = Options{
			WithObjects: NewObjects(objectTest{}),
			WithSubObjects: map[string]Objects{
				"a": NewObjects(objectB{}),
				"b": NewObjects(objectA{}, objectB{}),
			},
		}

		buf = bytes.NewBuffer(nil)

		packer   = NewPacker(buf, options)
		unpacker = NewUnpacker(buf, options)

		input any = objectTest{
			RawA: &objectB{"Hello"},
			RawB: &objectA{"World"},
			Slice: []any{
				&objectB{"A"},
				&objectA{"B"},
				&objectB{"C"},
			},
			Array: [2]any{
				&objectA{"D"},
				&objectB{"E"},
			},
			Map: map[string]any{
				"F": &objectB{"G"},
				"H": &objectA{"I"},
			},
		}
		output any
	)

	err := packer.Encode(input)
	if err != nil {
		t.Fatal(err)
	}

	err = unpacker.Decode(&output)
	if err != nil {
		t.Error(err)
	}

	inputType := reflect.TypeOf(input)

	if reflect.TypeOf(output) != reflect.PointerTo(inputType) {
		t.Errorf("decoded object type should be a pointer to input object type; expected: %s, got: %s",
			reflect.PointerTo(inputType).String(), reflect.TypeOf(output).String())
	} else {
		output = reflect.ValueOf(output).Elem().Interface()

		if !reflect.DeepEqual(input, output) {
			t.Errorf("decoded object should equal input object; expected: %+v, got: %+v", input, output)
		}
	}

	if buf.Len() > 0 {
		t.Errorf("expected all bytes to be consumed after decoding all objects, got %d extra bytes: %q", buf.Len(), buf.Bytes())
	}
}
