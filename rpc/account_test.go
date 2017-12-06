package rpc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

var (
	client = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 1000,
		},
		Timeout: time.Duration(60) * time.Second,
	}

	rpcURL      = "http://192.168.8.222:8881"
	ContentType = "application/json"
	rpcURLList  = []string{
		"http://192.168.8.222:8881",
		"http://192.168.8.222:8882",
		"http://192.168.8.222:8883",
		"http://192.168.8.222:8884",
	}
)

func TestAccountCreate(t *testing.T) {
	testcase := map[string]string{
		"ST_ACCOUNT_001":   `{"id": 1, "method": "Account.New", "params": [{"AccountType":-4294967295, "Passphrase": "123456"}]}`,
		"ST_ACCOUNT_002":   `{"id": 1, "method": "Account.New", "params": [{"AccountType":-2147483648, "Passphrase": "123456"}]}`,
		"ST_ACCOUNT_003":   `{"id": 1, "method": "Account.New", "params": [{"AccountType":0, "Passphrase": "123456"}]}`,
		"ST_ACCOUNT_004":   `{"id": 1, "method": "Account.New", "params": [{"AccountType":1, "Passphrase": "123456"}]}`,
		"ST_ACCOUNT_005":   `{"id": 1, "method": "Account.New", "params": [{"AccountType":2, "Passphrase": "123456"}]}`,
		"ST_ACCOUNT_006":   `{"id": 1, "method": "Account.New", "params": [{"AccountType":3, "Passphrase": "123456"}]}`,
		"ST_ACCOUNT_007":   `{"id": 1, "method": "Account.New", "params": [{"AccountType":4, "Passphrase": "123456"}]}`,
		"ST_ACCOUNT_008":   `{"id": 1, "method": "Account.New", "params": [{"AccountType":4294967295, "Passphrase": "123456"}]}`,
		"ST_ACCOUNT_009":   `{"id": 1, "method": "Account.New", "params": [{"AccountType":2147483648, "Passphrase": "123456"}]}`,
		"ST_ACCOUNT_010":   `{"id": 1, "method": "Account.New", "params": [{"AccountType":1, "Passphrase": ""}]}`,
		"ST_ACCOUNT_013":   `{"id": 1, "method": "Account.New", "params": []}`,
		"ST_ACCOUNT_014":   `{"id": 1, "method": "Account.New", "params": [{"AccountType":1, "Passphrase": 123456}]}`,
		"ST_ACCOUNT_014_2": `{"id": 1, "method": "Account.New", "params": [{"AccountType":"1", "Passphrase": "123456"}]}`,
	}

	passLen := []int{1, 5, 6, 7, 8, 9, 32, 33, 64, 65, 128, 129, 256, 257, 512, 513, 1024, 1025, 2048, 2049, 10240, 10240000}

	pass := []string{
		`             `,
		`'`,
		`"`,
		`//`,
		`/`,
		`\`,
		`\\`,
		`#`,
		`_`,
		`密码`,
	}

	for n, size := range passLen {
		testcase[fmt.Sprintf("ST_ACCOUNT_011_%d", n)] = `{"id": 1, "method": "Account.New", "params": [{"AccountType":1, "Passphrase": "` + strings.Repeat("s", size) + `"}]}`
	}

	for n, ps := range pass {
		testcase[fmt.Sprintf("ST_ACCOUNT_012_%d", n)] = `{"id": 1, "method": "Account.New", "params": [{"AccountType":1, "Passphrase": "` + ps + `"}]}`
	}

	for k, v := range testcase {
		now := time.Now()
		resp := sendRPC(v, rpcURL)
		t.Logf("TestCase [%s]  Used Time:(%v)   Result: %s ", k, time.Now().Sub(now), string(resp))
	}

}

func TestAccountList(t *testing.T) {
	testcase := `{"id": 2, "method": "Account.List", "params":["asd"]}'`
	testcase1 := `{"id": 2, "method": "Account.List", "params":[]}'`
	resp := sendRPC(testcase, rpcURL)
	now := time.Now()
	t.Logf("TestCase [ST_ACCOUNT_016] Used Time:(%v) Result: %s ", time.Now().Sub(now), string(resp))
	now = time.Now()
	resp = sendRPC(testcase1, rpcURL)
	t.Logf("TestCase [ST_ACCOUNT_015] Used Time:(%v) Result: %s ", time.Now().Sub(now), string(resp))
}

func BenchmarkAccountListTest(t *testing.B) {
	for i := 0; i < t.N; i++ {
		resp := sendRPC(`{"id": 2, "method": "Account.List", "params":[]}'`, rpcURL)
		_ = resp
		// log.Printf("TestCase Result: %s ", string(resp))
	}
}

func BenchmarkCreateAccountTest(t *testing.B) {
	for i := 0; i < t.N; i++ {
		resp := sendRPC(`{"id": 1, "method": "Account.New", "params": [{"AccountType":1, "Passphrase": "123456"}]}`, rpcURL)
		_ = resp
		// log.Printf("Result: %s ", string(resp))
	}
}

func sendRPC(params string, address ...string) []byte {

	buf := bytes.NewBuffer(nil)

	for _, addr := range address {
		req, err := http.NewRequest("POST", addr, bytes.NewBufferString(params))
		if err != nil {
			panic(err)
		}

		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		defer resp.Body.Close()

		res, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		buf.Write(res)
	}

	return buf.Bytes()
}
