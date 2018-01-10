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

package jsvm

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/robertkrimen/otto"
)

func TestString(t *testing.T) {
	v, _ := otto.ToValue("123")
	buf, _ := jsvalueToByte(v)

	v2, _ := byteToJSvalue(bytes.NewBuffer(buf), nil)
	if s, err := v2.ToString(); err != nil || s != "123" {
		t.Error("string convert")
	}
}

func TestObject(t *testing.T) {
	vm := otto.New()
	obj, err := vm.Object(`p = {
		name : "namevalue",
		age  : 100
	}`)
	if err != nil {
		t.Error(err)
	}

	v, err := vm.ToValue(obj)
	if err != nil {
		t.Error(err)
	}

	buf, _ := jsvalueToByte(v)

	v2, _ := byteToJSvalue(bytes.NewBuffer(buf), vm)
	if v2.IsNull() {
		t.Error("v2 is null")
	}
	obj2 := v2.Object()
	name, _ := obj2.Get("name")
	sname, _ := name.ToString()
	age, _ := obj2.Get("age")
	iage, _ := age.ToInteger()
	if "namevalue" != sname {
		t.Error("name not equal")
	}
	if 100 != iage {
		t.Error("age not equal")
	}
}

func TestKvsToJSValue(t *testing.T) {
	vm := otto.New()
	v, _ := otto.ToValue("word")
	value, _ := jsvalueToByte(v)
	kvs := []*db.KeyValue{&db.KeyValue{Key: []byte("hello"), Value: value}}

	v, err := kvsToJSValue(kvs, vm)
	if err != nil {
		t.Error("convert kvs error", err)
	}
	obj := v.Object()
	result, _ := obj.Get("hello")
	r, _ := result.ToString()

	if r != "word" {
		t.Error("error not same", r)
	}
}

func TestObjToLValue(t *testing.T) {
	vm := otto.New()
	balance := state.NewBalance()
	balance.Amounts[0] = big.NewInt(0)
	balance.Amounts[1] = big.NewInt(-1)
	balance.Amounts[2] = big.NewInt(2)

	balance.Nonce = 100

	v, err := objToLValue(balance, vm)
	if err != nil {
		t.Error("convert obj error: ", err)
	}

	obj := v.Object()

	nonce, err := obj.Get("Nonce")
	if err != nil {
		t.Error("obj get Nonce error: ", err)
	}

	n, err := nonce.ToInteger()
	if err != nil {
		t.Error("convert Nonce error: ", err)
	}

	if uint32(n) != balance.Nonce {
		t.Error("error not same", n)
	}

	amounts, err := obj.Get("Amounts")
	if err != nil {
		t.Error("obj get amounts error: ", err)
	}
	amountsObj := amounts.Object()
	value, err := amountsObj.Get("0")
	if err != nil {
		t.Error("obj get id = 0 error: ", err)
	}
	amount, err := value.ToString()
	if amount != "0" {
		t.Error("error not same", amount)
	}

}
