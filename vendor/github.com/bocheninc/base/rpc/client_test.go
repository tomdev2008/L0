package rpc

import (
	"fmt"
	"testing"
)

type Argst struct {
	A, B int
}

func TestClient(t *testing.T) {
	client, err := DialHTTP("http://127.0.0.1:8000")
	var result int
	fmt.Println("client.Call", client.Call("Arith.Multiply", Argst{A: 1, B: 2}, &result))
	fmt.Println("result", result, err)
	// client.Do()
}
