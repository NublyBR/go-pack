package pack

import (
	"testing"
)

type objectBeforePack struct {
	Val string

	wasCalled bool
}

func (a *objectBeforePack) BeforePack() error {
	a.wasCalled = true
	return nil
}

func TestBeforePack(t *testing.T) {

	t.Parallel()

	var (
		input = objectBeforePack{
			Val: "Hello, World!",
		}
	)

	_, err := Marshal(&input)
	if err != nil {
		t.Error(err)
	}

	if !input.wasCalled {
		t.Errorf("expected method output.BeforePack() to be called during object packing")
	}
}

type objectAfterUnpack struct {
	Val string

	wasCalled bool
}

func (a *objectAfterUnpack) AfterUnpack() error {
	a.wasCalled = true
	return nil
}
func TestAfterUnpack(t *testing.T) {

	t.Parallel()

	var (
		input = objectAfterUnpack{
			Val: "Hello, World!",
		}
		output objectAfterUnpack
	)

	data, err := Marshal(input)
	if err != nil {
		t.Error(err)
	}

	err = Unmarshal(data, &output)
	if err != nil {
		t.Error(err)
	}

	if !output.wasCalled {
		t.Errorf("expected method output.AfterUnpack() to be called after object unpacking")
	}
}
