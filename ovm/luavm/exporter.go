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

// generate go function and register to vm, call from lua script

package luavm

import (
	"bytes"
	"time"

	"github.com/bocheninc/L0/vm/luavm/utils"
	"github.com/yuin/gopher-lua"
)

func exporter() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"Account":            accountFunc,
		"TxInfo":             txInfoFunc,
		"Transfer":           transferFunc,
		"CurrentBlockHeight": currentBlockHeightFunc,
		"GetGlobalState":     getGlobalStateFunc,
		"SetGlobalState":     setGlobalStateFunc,
		"DelGlobalState":     delGlobalStateFunc,
		"GetState":           getStateFunc,
		"PutState":           putStateFunc,
		"DelState":           delStateFunc,
		"GetByPrefix":        getByPrefixFunc,
		"GetByRange":         getByRangeFunc,
		"ComplexQuery":       complexQueryFunc,
		"jsonEncode":         utils.ApiEncode,
		"jsonDecode":         utils.ApiDecode,
		"sleep":              sleepFunc,
	}
}

func sleepFunc(l *lua.LState) int {
	if l.GetTop() != 1 {
		l.RaiseError("param illegality when invoke sleep")
		return 1
	}
	n := time.Duration(int64(float64(l.CheckNumber(1))))
	time.Sleep(n * time.Millisecond)
	l.Push(lua.LBool(true))
	return 1
}

func accountFunc(l *lua.LState) int {
	var addr, sender, recipient string
	var amount int64
	if l.GetTop() == 1 {
		addr = l.CheckString(1)
	} else {
		addr = vmproc.ContractData.ContractAddr
	}

	balances, err := vmproc.CCallGetBalances(addr)
	if err != nil {
		l.RaiseError("get balances error addr:%s  err:%s", addr, err)
		return 1
	}
	sender = vmproc.ContractData.Transaction.Sender().String()
	amount = vmproc.ContractData.Transaction.Amount().Int64()
	recipient = vmproc.ContractData.Transaction.Recipient().String()
	tb := l.NewTable()
	tb.RawSetString("Sender", lua.LString(sender))
	tb.RawSetString("Address", lua.LString(addr))
	tb.RawSetString("Recipient", lua.LString(recipient))
	tb.RawSetString("Amount", lua.LNumber(amount))
	tb.RawSetString("Balances", objToLValue(balances))
	l.Push(tb)
	return 1
}

func txInfoFunc(l *lua.LState) int {
	sender := vmproc.ContractData.Transaction.Sender().String()
	recipient := vmproc.ContractData.Transaction.Recipient().String()
	assetID := vmproc.ContractData.Transaction.AssetID()
	amount := vmproc.ContractData.Transaction.Amount().Int64()
	tb := l.NewTable()
	tb.RawSetString("Sender", lua.LString(sender))
	tb.RawSetString("Recipient", lua.LString(recipient))
	tb.RawSetString("AssetID", lua.LNumber(assetID))
	tb.RawSetString("Amount", lua.LNumber(amount))
	l.Push(tb)
	return 1
}

func transferFunc(l *lua.LState) int {
	if l.GetTop() != 3 {
		l.RaiseError("param illegality when invoke Transfer")
		return 1
	}

	recipientAddr := l.CheckString(1)
	id := int64(float64(l.CheckNumber(2)))
	amout := int64(float64(l.CheckNumber(3)))
	txType := uint32(0)
	err := vmproc.CCallTransfer(recipientAddr, id, amout, txType)
	if err != nil {
		l.RaiseError("contract do transfer error recipientAddr:%s,id:%d, amout:%d, txType:%d  err:%s", recipientAddr, id, amout, txType, err)
		return 1
	}

	l.Push(lua.LBool(true))
	return 1
}

func currentBlockHeightFunc(l *lua.LState) int {
	height, err := vmproc.CCallCurrentBlockHeight()
	if err != nil {
		l.RaiseError("get currentBlockHeight error")
		return 1
	}

	l.Push(lua.LNumber(height))
	return 1
}

func getGlobalStateFunc(l *lua.LState) int {
	if l.GetTop() != 1 {
		l.RaiseError("param illegality when invoke GetGlobalState")
		return 1
	}

	key := l.CheckString(1)
	data, err := vmproc.CCallGetGlobalState(key)
	if err != nil {
		l.RaiseError("GetGlobalState error key:%s  err:%s", key, err)
		return 1
	}

	if data == nil {
		l.Push(lua.LNil)
		return 1
	}

	buf := bytes.NewBuffer(data)
	if lv, err := byteToLValue(buf); err != nil {
		l.RaiseError("byteToLValue error")
	} else {
		l.Push(lv)
	}

	return 1
}

func setGlobalStateFunc(l *lua.LState) int {
	if l.GetTop() != 2 {
		l.RaiseError("param illegality when invoke SetGlobalState")
		return 1
	}

	key := l.CheckString(1)
	value := l.Get(2)

	data := lvalueToByte(value)
	err := vmproc.CCallSetGlobalState(key, data)
	if err != nil {
		l.RaiseError("SetGlobalState error key:%s  err:%s", key, err)
	} else {
		l.Push(lua.LBool(true))
	}

	return 1
}

func delGlobalStateFunc(l *lua.LState) int {
	if l.GetTop() != 1 {
		l.RaiseError("param illegality when invoke DelGlobalState")
		return 1
	}

	key := l.CheckString(1)
	err := vmproc.CCallDelGlobalState(key)
	if err != nil {
		l.RaiseError("DelGlobalState error key:%s   err:%s", key, err)
	} else {
		l.Push(lua.LBool(true))
	}

	return 1
}

func getStateFunc(l *lua.LState) int {
	if l.GetTop() != 1 {
		l.RaiseError("param illegality when invoke GetState")
		return 1
	}

	key := l.CheckString(1)
	data, err := vmproc.CCallGetState(key)
	if err != nil {
		l.RaiseError("getState error key:%s  err:%s", key, err)
		return 1
	}
	if data == nil {
		l.Push(lua.LNil)
		return 1
	}

	buf := bytes.NewBuffer(data)
	if lv, err := byteToLValue(buf); err != nil {
		l.RaiseError("byteToLValue error")
	} else {
		l.Push(lv)
	}

	return 1
}

func putStateFunc(l *lua.LState) int {
	if l.GetTop() != 2 {
		l.RaiseError("param illegality when invoke PutState")
		return 1
	}

	key := l.CheckString(1)
	value := l.Get(2)

	data := lvalueToByte(value)
	err := vmproc.CCallPutState(key, data)
	if err != nil {
		l.RaiseError("putState error key:%s  err:%s", key, err)
	} else {
		l.Push(lua.LBool(true))
	}

	return 1
}

func delStateFunc(l *lua.LState) int {
	if l.GetTop() != 1 {
		l.RaiseError("param illegality when invoke DelState")
		return 1
	}

	key := l.CheckString(1)
	err := vmproc.CCallDelState(key)
	if err != nil {
		l.RaiseError("delState error key:%s   err:%s", key, err)
	} else {
		l.Push(lua.LBool(true))
	}

	return 1
}

func getByPrefixFunc(l *lua.LState) int {
	if l.GetTop() != 1 {
		l.RaiseError("param illegality when invoke getByPrefix")
		return 1
	}

	key := l.CheckString(1)

	values, err := vmproc.CCallGetByPrefix(key)

	if err != nil {
		l.RaiseError("getByPrefix error key:%s  err:%s", key, err)
	}
	if values == nil {
		l.Push(lua.LNil)
		return 1
	}

	if lv, err := kvsToLValue(values); err != nil {
		l.RaiseError("byteToLValue error")
	} else {
		l.Push(lv)
	}

	return 1
}

func getByRangeFunc(l *lua.LState) int {
	if l.GetTop() != 2 {
		l.RaiseError("param illegality when invoke getByRange")
		return 1
	}

	startKey := l.CheckString(1)
	limitKey := l.CheckString(2)
	values, err := vmproc.CCallGetByRange(startKey, limitKey)
	if err != nil {
		l.RaiseError("getByRange error startKey:%s ,limitKsy:%s ,err:%s", startKey, limitKey, err)
	}
	if values == nil {
		l.Push(lua.LNil)
		return 1
	}

	lv, err := kvsToLValue(values)
	if err != nil {
		l.RaiseError("byteToLValue error")
	} else {
		l.Push(lv)
	}

	return 1
}

func complexQueryFunc(l *lua.LState) int {
	if l.GetTop() != 1 {
		l.RaiseError("param illegality when invoke complexQuery")
		return 1
	}

	key := l.CheckString(1)
	data, err := vmproc.CCallComplexQuery(key)
	if err != nil {
		l.RaiseError("complexQuery error  key:%s  err:%s", key, err)
		return 1
	}
	if data == nil {
		l.Push(lua.LNil)
		return 1
	}

	lv := lua.LString(string(data))
	l.Push(lv)
	return 1
}
