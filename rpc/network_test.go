package rpc

import (
	"testing"
)

func BenchmarkNetworkGetPeers(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp := sendRPC(`{"id":1, "method":"Net.GetPeers", "params":[""]}`, rpcURL)
		_ = resp
		b.Logf(string(resp))
	}
}

func BenchmarkNetworkGetLocalPeer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp := sendRPC(`{"id":1, "method":"Net.GetLocalPeer", "params":[""]}`, rpcURL)
		_ = resp
		b.Logf(string(resp))
	}
}
