// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of L0
//
// The L0 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The L0 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"encoding/hex"
	"math/big"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/bocheninc/L0/core/types"
)

func TestMain(m *testing.M) {
	VMConf = DefaultConfig()
	VMConf.JSVMExeFilePath = "../build/bin/jsvm"
	VMConf.LuaVMExeFilePath = "../build/bin/luavm"

	if err := initVMProc("luavm"); err != nil {
		fmt.Println("init vm proc ", err)
	}

	flag.Parse()
	os.Exit(m.Run())
}

func TestEncode(t *testing.T) {
	d := new(InvokeData)
	d.FuncName = "fn"
	d.SessionID = 123
	d.SetParams("contract script code size illegal, max size is:5120 byte", false, []*db.KeyValue{&db.KeyValue{Key: []byte("key"), Value: []byte("value")}})

	t.Log("len:", len(d.Params))

	var str string
	var rst bool
	var kvs []*db.KeyValue

	d.DecodeParams(&str, &rst, &kvs)
	for _, v := range kvs {
		t.Log("key: ", string(v.Key), "value: ", string(v.Value))
	}
	t.Log("str:", str, " rst:", rst)
}

func TestExecute(t *testing.T) {
	tx := types.NewTransaction(nil, nil, types.TypeContractInvoke, 0, accounts.Address{}, accounts.NewAddress([]byte("999999999999999999")), 0, big.NewInt(0), big.NewInt(0), 0)

	cs := &types.ContractSpec{
		ContractCode:   getCode(),
		ContractAddr:   []byte("11111111111111111111"),
		ContractParams: []string{"transfer", hex.EncodeToString([]byte("12345678900987654321")), "99"}}

	hd := &L0Handler{}

	success, err := PreExecute(tx, cs, hd)
	if err != nil {
		t.Error("percall error", err)
	}
	t.Log("success:", success)

}

type L0Handler struct {
}

func (hd *L0Handler) GetState(key string) ([]byte, error) {
	if "balances" == key {
		buf := []byte{4, 3, 1, 1, 99, 3, 0, 0, 0, 0, 0, 192, 114, 64, 1, 8, 114, 101, 99, 101, 105, 118, 101, 114, 3, 0, 0, 0, 0, 0, 0, 105, 64, 1, 6, 115, 101, 110, 100, 101, 114, 3, 0, 0, 0, 0, 0, 0, 89, 64}
		return buf, nil
	} else if contractCodeKey == key {
		return getCode(), nil
	}

	return nil, nil
}

func (hd *L0Handler) ComplexQuery(key string) ([]byte, error) {

	return nil, nil
}

func (hd *L0Handler) AddState(key string, value []byte) {

}

func (hd *L0Handler) DelState(key string) {

}

func (hd *L0Handler) GetBalances(addr string) (*state.Balance, error) {
	//fmt.Println("test GetBalances:", addr)
	amount := make(map[uint32]*big.Int)
	amount[0] = big.NewInt(100)
	return &state.Balance{Amounts: amount, Nonce: 0}, nil
}

func (hd *L0Handler) CurrentBlockHeight() uint32 {
	return 100
}

func (hd *L0Handler) AddTransfer(fromAddr, toAddr string, assetID uint32, amount *big.Int, txType uint32) {
	fmt.Printf("AddTransfer from:%s to:%s amount:%d txType:%d", fromAddr, toAddr, amount.Int64(), txType)
}

func (hd *L0Handler) SmartContractFailed() {

}

func (hd *L0Handler) SmartContractCommitted() {

}

func (hd *L0Handler) GetByPrefix(prefix string) []*db.KeyValue {

	return []*db.KeyValue{&db.KeyValue{Key: []byte("key"), Value: []byte("value")}}
}

func (hd *L0Handler) GetByRange(startKey, limitKey string) []*db.KeyValue {

	return []*db.KeyValue{&db.KeyValue{Key: []byte("key1"), Value: []byte("value1")}}
}

func (hd *L0Handler) DelGlobalState(key string) error {
	return nil
}

func (hd *L0Handler) GetGlobalState(key string) ([]byte, error) {
	return nil, nil
}

func (hd *L0Handler) SetGlobalState(key string, value []byte) error {
	return nil
}

func getCode() []byte {
	f, _ := os.Open("../tests/contract/l0coin.lua")
	defer f.Close()
	buf, _ := ioutil.ReadAll(f)
	return buf
}
