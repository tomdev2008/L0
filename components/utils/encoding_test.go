package utils

import (
	"testing"
)

func TestStruct(t *testing.T) {
	type st struct {
		A string
		b int
	}

	from := st{
		A: "a",
		b: 1,
	}

	var to st
	err := Deserialize(Serialize(from), &to)
	if err != nil {
		t.Error(err)
	}

	target := st{
		A: "a",
	}
	if to != target {
		t.Errorf("de-serialize mismatch, get %#v, need %#v", to, target)
	}
}
