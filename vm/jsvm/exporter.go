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
	"time"

	"github.com/bocheninc/base/log"
	"github.com/robertkrimen/otto"
)

func exporter(ottoVM *otto.Otto) (*otto.Object, error) {
	exporterFuncs, _ := ottoVM.Object(`L0 = {
		toNumber: function(value, def) {
			if(typeof(value) == "undefined") {
				return def;
			} else {
				return value * 1;
			}
		}
	}`)

	exporterFuncs.Set("Account", accountFunc)
	exporterFuncs.Set("TxInfo", txInfoFunc)
	exporterFuncs.Set("Transfer", transferFunc)
	exporterFuncs.Set("CurrentBlockHeight", currentBlockHeightFunc)
	exporterFuncs.Set("GetGlobalState", getGlobalStateFunc)
	exporterFuncs.Set("SetGlobalState", setGlobalStateFunc)
	exporterFuncs.Set("DelGlobalState", delGlobalStateFunc)
	exporterFuncs.Set("GetState", getStateFunc)
	exporterFuncs.Set("PutState", putStateFunc)
	exporterFuncs.Set("DelState", delStateFunc)
	exporterFuncs.Set("GetByPrefix", getByPrefixFunc)
	exporterFuncs.Set("GetByRange", getByRangeFunc)
	exporterFuncs.Set("ComplexQuery", complexQueryFunc)
	exporterFuncs.Set("Sleep", sleepFunc)

	return exporterFuncs, nil
}

func sleepFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 1 {
		log.Error("sleepFunc -> param illegality when invoke Transfer")
		return fc.Otto.MakeCustomError("sleepFunc", "param illegality when invoke Sleep")
	}

	n, err := fc.Argument(0).ToInteger()
	if err != nil {
		log.Errorf("sleepFunc -> get duration error")
		return fc.Otto.MakeCustomError("sleepFunc", err.Error())
	}
	time.Sleep(time.Duration(n) * time.Millisecond)
	val, _ := otto.ToValue(true)
	return val
}

func accountFunc(fc otto.FunctionCall) otto.Value {
	var addr, sender, recipient string
	var amount int64
	var err error
	if len(fc.ArgumentList) == 1 {
		addr, err = fc.Argument(0).ToString()
	} else {
		addr = vmproc.ContractData.ContractAddr
	}

	balances, err := vmproc.CCallGetBalances(addr)
	if err != nil {
		log.Error("accountFunc -> call CCallGetBalances error", err)
		return fc.Otto.MakeCustomError("accountFunc", "call CCallGetBalances error:"+err.Error())
	}

	balancesValue, err := objToLValue(balances, fc.Otto)
	if err != nil {
		log.Error("accountFunc -> call objToLValue error", err)
		return fc.Otto.MakeCustomError("accountFunc", "call objToLValue error:"+err.Error())
	}

	sender = vmproc.ContractData.Transaction.Sender().String()
	recipient = vmproc.ContractData.Transaction.Recipient().String()

	amount = vmproc.ContractData.Transaction.Amount().Int64()
	amountValue, err := fc.Otto.ToValue(amount)
	if err != nil {
		log.Error("accountFunc -> call amount ToLValue error", err)
		return fc.Otto.MakeCustomError("accountFunc", "call call amount ToLValue error:"+err.Error())
	}

	mp := make(map[string]interface{}, 3)
	mp["Address"] = addr
	mp["Balances"] = balancesValue
	mp["Sender"] = sender
	mp["Recipient"] = recipient
	mp["Amount"] = amountValue

	val, err := fc.Otto.ToValue(mp)
	if err != nil {
		log.Error("accountFunc -> otto ToValue error", err)
		return fc.Otto.MakeCustomError("accountFunc", "otto ToValue error:"+err.Error())
	}
	return val
}

func txInfoFunc(fc otto.FunctionCall) otto.Value {
	sender := vmproc.ContractData.Transaction.Sender().String()
	recipient := vmproc.ContractData.Transaction.Recipient().String()
	assetID := vmproc.ContractData.Transaction.AssetID()
	amount := vmproc.ContractData.Transaction.Amount().Int64()
	amountValue, err := fc.Otto.ToValue(amount)
	if err != nil {
		log.Error("accountFunc -> call amount ToLValue error", err)
		return fc.Otto.MakeCustomError("accountFunc", "call call amount ToLValue error:"+err.Error())
	}

	mp := make(map[string]interface{}, 4)
	mp["Sender"] = sender
	mp["Recipient"] = recipient
	mp["AssetID"] = assetID
	mp["Amount"] = amountValue

	val, err := fc.Otto.ToValue(mp)
	if err != nil {
		log.Error("accountFunc -> otto ToValue error", err)
		return fc.Otto.MakeCustomError("accountFunc", "otto ToValue error:"+err.Error())
	}
	return val
}

func transferFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 3 {
		log.Error("transferFunc -> param illegality when invoke Transfer")
		return fc.Otto.MakeCustomError("transferFunc", "param illegality when invoke Transfer")
	}

	recipientAddr, err := fc.Argument(0).ToString()
	if err != nil {
		log.Errorf("transferFunc -> get recipientAddr arg error")
		return fc.Otto.MakeCustomError("transferFunc", err.Error())
	}

	id, err := fc.Argument(1).ToInteger()
	if err != nil {
		log.Errorf("transferFunc -> get id arg error")
		return fc.Otto.MakeCustomError("transferFunc", err.Error())
	}

	amout, err := fc.Argument(2).ToInteger()
	if err != nil {
		log.Errorf("transferFunc -> get amout arg error")
		return fc.Otto.MakeCustomError("transferFunc", err.Error())
	}
	txType := uint32(0)
	err = vmproc.CCallTransfer(recipientAddr, id, amout, txType)
	if err != nil {
		log.Errorf("transferFunc -> contract do transfer error recipientAddr:%s, amout:%d, txType:%d  err:%s", recipientAddr, amout, txType, err)
		return fc.Otto.MakeCustomError("transferFunc", err.Error())
	}

	val, _ := otto.ToValue(true)
	return val
}

func currentBlockHeightFunc(fc otto.FunctionCall) otto.Value {
	height, err := vmproc.CCallCurrentBlockHeight()
	if err != nil {
		log.Error("currentBlockHeightFunc -> get currentBlockHeight error")
		return fc.Otto.MakeCustomError("currentBlockHeightFunc", "get currentBlockHeight error:"+err.Error())
	}

	val, err := otto.ToValue(height)
	if err != nil {
		return otto.NullValue()
	}
	return val
}

func getGlobalStateFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 1 {
		log.Error("param illegality when invoke getGlobalState")
		return fc.Otto.MakeCustomError("getGlobalStateFunc", "param illegality when invoke getGlobalState")
	}

	key, err := fc.Argument(0).ToString()
	data, err := vmproc.CCallGetGlobalState(key)
	if err != nil {
		log.Errorf("getGlobalState error key:%s  err:%s", key, err)
		return fc.Otto.MakeCustomError("getGlobalStateFunc", "getGlobalState error:"+err.Error())
	}

	if data == nil {
		return otto.NullValue()
	}

	buf := bytes.NewBuffer(data)
	val, err := byteToJSvalue(buf, fc.Otto)
	if err != nil {
		log.Error("byteToJSvalue error", err)
		return fc.Otto.MakeCustomError("getGlobalStateFunc", "byteToJSvalue error:"+err.Error())
	}
	return val
}

func setGlobalStateFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 2 {
		log.Error("param illegality when invoke SetGlobalState")
		return fc.Otto.MakeCustomError("setGlobalStateFunc", "param illegality when invoke SetGlobalState")
	}

	key, err := fc.Argument(0).ToString()
	if err != nil {
		log.Error("get string key error", err)
		return fc.Otto.MakeCustomError("setGlobalStateFunc", "get string key error"+err.Error())
	}

	value := fc.Argument(1)
	data, err := jsvalueToByte(value)
	if err != nil {
		log.Errorf("jsvalueToByte error key:%s  err:%s", key, err)
		return fc.Otto.MakeCustomError("setGlobalStateFunc", "jsvalueToByte error:"+err.Error())
	}

	err = vmproc.CCallSetGlobalState(key, data)
	if err != nil {
		log.Errorf("SetGlobalState error key:%s  err:%s", key, err)
		return fc.Otto.MakeCustomError("setGlobalStateFunc", "SetGlobalState error:"+err.Error())
	}

	val, _ := otto.ToValue(true)
	return val
}

func delGlobalStateFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 1 {
		log.Error("param illegality when invoke DelGlobalState")
		return fc.Otto.MakeCustomError("delGlobalStateFunc", "param illegality when invoke DelGlobalState")
	}

	key, err := fc.Argument(0).ToString()
	err = vmproc.CCallDelGlobalState(key)
	if err != nil {
		log.Errorf("DelGlobalState error key:%s   err:%s", key, err)
		return fc.Otto.MakeCustomError("delGlobalStateFunc", "DelGlobalState error:"+err.Error())
	}

	val, _ := otto.ToValue(true)
	return val
}

func getStateFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 1 {
		log.Error("param illegality when invoke GetState")
		return fc.Otto.MakeCustomError("getStateFunc", "param illegality when invoke GetState")
	}

	key, err := fc.Argument(0).ToString()
	data, err := vmproc.CCallGetState(key)
	if err != nil {
		log.Errorf("getState error key:%s  err:%s", key, err)
		return fc.Otto.MakeCustomError("getStateFunc", "getState error:"+err.Error())
	}
	if data == nil {
		return otto.NullValue()
	}

	buf := bytes.NewBuffer(data)
	val, err := byteToJSvalue(buf, fc.Otto)
	if err != nil {
		log.Error("byteToJSvalue error", err)
		return fc.Otto.MakeCustomError("getStateFunc", "byteToJSvalue error:"+err.Error())
	}
	return val
}

func putStateFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 2 {
		log.Error("param illegality when invoke PutState")
		return fc.Otto.MakeCustomError("putStateFunc", "param illegality when invoke PutState")
	}

	key, err := fc.Argument(0).ToString()
	if err != nil {
		log.Error("get string key error", err)
		return fc.Otto.MakeCustomError("putStateFunc", "get string key error"+err.Error())
	}

	value := fc.Argument(1)
	data, err := jsvalueToByte(value)
	if err != nil {
		log.Errorf("jsvalueToByte error key:%s  err:%s", key, err)
		return fc.Otto.MakeCustomError("putStateFunc", "jsvalueToByte error:"+err.Error())
	}

	err = vmproc.CCallPutState(key, data)
	if err != nil {
		log.Errorf("putState error key:%s  err:%s", key, err)
		return fc.Otto.MakeCustomError("putStateFunc", "putState error:"+err.Error())
	}

	val, _ := otto.ToValue(true)
	return val
}

func delStateFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 1 {
		log.Error("param illegality when invoke DelState")
		return fc.Otto.MakeCustomError("delStateFunc", "param illegality when invoke DelState")
	}

	key, err := fc.Argument(0).ToString()
	err = vmproc.CCallDelState(key)
	if err != nil {
		log.Errorf("delState error key:%s   err:%s", key, err)
		return fc.Otto.MakeCustomError("delStateFunc", "delState error:"+err.Error())
	}

	val, _ := otto.ToValue(true)
	return val
}

func getByPrefixFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 1 {
		log.Error("param illegality when invoke GetByPrefix")
		return fc.Otto.MakeCustomError("getByPrefixFunc", "param illegality when invoke GetByPrefix")
	}

	perfix, err := fc.Argument(0).ToString()
	values, err := vmproc.CCallGetByPrefix(perfix)
	if err != nil {
		log.Errorf("getByPrefix error key:%s  err:%s", perfix, err)
		return fc.Otto.MakeCustomError("getByPrefixFunc", "getByPrefix error:"+err.Error())
	}
	if values == nil {
		return otto.NullValue()
	}

	val, err := kvsToJSValue(values, fc.Otto)
	if err != nil {
		log.Error("byteToJSvalue error", err)
		return fc.Otto.MakeCustomError("getByPrefixFunc", "byteToJSvalue error:"+err.Error())
	}
	return val
}

func getByRangeFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 2 {
		log.Error("param illegality when invoke GetByRange")
		return fc.Otto.MakeCustomError("getByRangeFunc", "param illegality when invoke GetByRange")
	}

	startKey, err := fc.Argument(0).ToString()
	limitKey, err := fc.Argument(1).ToString()

	values, err := vmproc.CCallGetByRange(startKey, limitKey)
	if err != nil {
		log.Errorf("getByRange error startKey:%s  limitKey:%s  err:%s", startKey, limitKey, err)
		return fc.Otto.MakeCustomError("getByRangeFunc", "getByRange error:"+err.Error())
	}
	if values == nil {
		return otto.NullValue()
	}

	val, err := kvsToJSValue(values, fc.Otto)
	if err != nil {
		log.Error("byteToJSvalue error", err)
		return fc.Otto.MakeCustomError("getByRangeFunc", "byteToJSvalue error:"+err.Error())
	}
	return val

}

func complexQueryFunc(fc otto.FunctionCall) otto.Value {
	if len(fc.ArgumentList) != 1 {
		log.Error("param illegality when invoke complexQuery")
		return fc.Otto.MakeCustomError("complexQueryFunc", "param illegality when invoke complexQuery")
	}
	key, err := fc.Argument(0).ToString()
	data, err := vmproc.CCallComplexQuery(key)
	if err != nil {
		log.Errorf("complexQuery error key:%s  err:%s", key, err)
		return fc.Otto.MakeCustomError("complexQueryFunc", "complexQuery error:"+err.Error())
	}
	if data == nil {
		return otto.NullValue()
	}

	val, err := otto.ToValue(string(data))
	if err != nil {
		log.Error("byteToJSvalue error", err)
		return fc.Otto.MakeCustomError("complexQueryFunc", "byteToJSvalue error:"+err.Error())
	}
	return val
}
