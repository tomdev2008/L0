package contract

import (
	"bytes"
	"strconv"
	"testing"
)

var stateExtra *StateExtra

func init() {
	stateExtra = NewStateExtra()
}

func TestSetAndGetAndDelete(t *testing.T) {
	stateExtra.set("contractAddr", "Tom", []byte("hello"))
	stateExtra.set("contractAddr", "Marry", []byte("world"))

	stateExtra.delete("contractAddr", "Marry")

	if !bytes.Equal([]byte("hello"), stateExtra.get("contractAddr", "Tom")) {
		t.Error("get result not equal set value ")
	}

	if stateExtra.get("contractAddr", "Marry") != nil {
		t.Error("get result not equal set value ")
	}

}

func TestStateGetByPrefix(t *testing.T) {
	for i := 0; i < 10; i++ {
		stateExtra.set("contractAddr", "Tom_"+strconv.Itoa(i), []byte("hello"+strconv.Itoa(i)))
		stateExtra.set("contractAddr", "Tom_1"+strconv.Itoa(i), []byte("hello_1"+strconv.Itoa(i)))

	}

	values := stateExtra.getByPrefix("contractAddr", "Tom_1")
	for _, v := range values {
		t.Log("key: ", string(v.Key), "value: ", string(v.Value))
	}

}

func TestStateGetByRange(t *testing.T) {
	for i := 0; i < 10; i++ {
		stateExtra.set("contractAddr", "Tom_"+strconv.Itoa(i), []byte("hello"+strconv.Itoa(i)))
	}

	values := stateExtra.getByRange("contractAddr", "Tom_1", "Tom_4")
	for _, v := range values {
		t.Log("key: ", string(v.Key), "value: ", string(v.Value))
	}
}
