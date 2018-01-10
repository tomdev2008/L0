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

package luavm

import (
	"math/big"
	"testing"

	"bytes"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/core/ledger/state"
	lua "github.com/yuin/gopher-lua"
)

func TestLValueConvert(t *testing.T) {
	ls := lua.LString("hello")
	data := lvalueToByte(ls)
	buf := bytes.NewBuffer(data)
	v, err := byteToLValue(buf)
	if err != nil || string(v.(lua.LString)) != "hello" {
		t.Error("convert string error")
	}

	lb := lua.LBool(true)
	data = lvalueToByte(lb)
	buf = bytes.NewBuffer(data)
	v, err = byteToLValue(buf)
	if err != nil || bool(v.(lua.LBool)) != true {
		t.Error("convert bool error")
	}

	ln := lua.LNumber(float64(123456789.4321))
	data = lvalueToByte(ln)
	buf = bytes.NewBuffer(data)
	v, err = byteToLValue(buf)
	if err != nil || float64(v.(lua.LNumber)) != 123456789.4321 {
		t.Error("convert number error")
	}

	ltb := new(lua.LTable)
	ltb.RawSetString("str", ls)
	ltb.RawSetInt(1, lb)
	ltb.RawSet(lb, lb)

	lctb := new(lua.LTable)
	ltb.RawSetInt(10, ls)

	ltb.RawSet(lctb, lctb)

	data = lvalueToByte(ltb)
	buf = bytes.NewBuffer(data)
	v, err = byteToLValue(buf)
	if err != nil {
		t.Error("convert table error")
	}
	ntb := v.(*lua.LTable)
	ntb.ForEach(func(key lua.LValue, value lua.LValue) {
		t.Log("key： ", key, "value：", value)

	})
}

func TestKvsToLValue(t *testing.T) {

	kvs := []*db.KeyValue{&db.KeyValue{Key: []byte("hello"), Value: lvalueToByte(lua.LString("word"))}}

	v, err := kvsToLValue(kvs)
	if err != nil {
		t.Error("convert kvs error", err)
	}

	ntb := v.(*lua.LTable)
	ntb.ForEach(func(key lua.LValue, value lua.LValue) {
		t.Log("key： ", key, "value：", value)
	})
}

func TestObjToLValue(t *testing.T) {
	balance := state.NewBalance()
	balance.Amounts[0] = big.NewInt(0)
	balance.Amounts[1] = big.NewInt(-1)
	balance.Amounts[2] = big.NewInt(2)

	balance.Nonce = 100

	v := objToLValue(balance)

	ntb := v.(*lua.LTable)
	ntb.ForEach(func(key lua.LValue, value lua.LValue) {
		switch value.(type) {
		case *lua.LTable:
			amountsTb := value.(*lua.LTable)
			amountsTb.ForEach(func(key lua.LValue, value lua.LValue) {
				t.Log("amounts key： ", key, ",amounts value：", value)
			})
		}
		t.Log("key： ", key, ",value：", value)
	})

}
