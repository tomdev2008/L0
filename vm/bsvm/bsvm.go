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

package bsvm

import (
	"errors"
	"fmt"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/vm"
	"github.com/bocheninc/L0/vm/luavm"
	"github.com/bocheninc/L0/vm/jsvm"
	"strings"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/core/params"
	"encoding/json"
)

type BsWorker struct {
	jsWorker *jsvm.JsWorker
	luaWorker *luavm.LuaWorker
	//can't use lock
}


func NewBsWorker(conf *vm.Config) *BsWorker {
	bsWorker := &BsWorker{
		jsWorker: jsvm.NewJsWorker(conf),
		luaWorker:luavm.NewLuaWorker(conf),
	}

	return bsWorker
}


func (worker *BsWorker) VmJob(data interface{}) interface{} {
	txType := "common"
	var err error
	workerProcWithCallback := data.(*vm.WorkerProcWithCallback)

	if !worker.isCommonTransaction(workerProcWithCallback) {
		if workerProcWithCallback.WorkProc.ContractData.Transaction.GetType() == types.TypeContractInvoke {
			txType, err = worker.GetInvokeType(workerProcWithCallback)
			if err != nil {
				log.Errorf("can't execute contract, err_msg: %+v", err.Error())
				workerProcWithCallback.Fn(err)
				return nil
			}
		} else {
			txType = worker.GetInitType(workerProcWithCallback)
		}
	}

	if strings.Contains(txType, "common") {
		worker.HandleCommonTransaction(workerProcWithCallback)
	} else if strings.Contains(txType, "lua"){
		worker.luaWorker.VmJob(workerProcWithCallback)
	} else {
		worker.jsWorker.VmJob(workerProcWithCallback)
	}
	return nil
}

func (worker *BsWorker) VmReady() bool {
	return true
}

func (worker *BsWorker) VmInitialize() {
	//pass
}

func (worker *BsWorker) VmTerminate() {
	//pass
}

func (worker *BsWorker) GetInvokeType(wpwc *vm.WorkerProcWithCallback) (string, error) {
	var err error
	cc := new(vm.ContractCode)
	var code []byte
	if len(wpwc.WorkProc.ContractData.ContractAddr) == 0 {
		code, err = wpwc.WorkProc.L0Handler.GetGlobalState(params.GlobalContractKey)
	} else {
		code, err = wpwc.WorkProc.L0Handler.GetState(vm.ContractCodeKey)
	}

	if len(code) != 0 && err == nil {
		contractCode, err := vm.DoContractStateData(code)
		if err != nil {
			return "", fmt.Errorf("cat't find contract code in db, err: %+v", err)
		}
		err = json.Unmarshal(contractCode, cc)
		if err != nil {
			return "", fmt.Errorf("cat't find contract code in db, err: %+v", err)
		}
		wpwc.WorkProc.ContractData.ContractCode = string(cc.Code)
		return cc.Type, nil
	} else {
		return "", errors.New("cat't find contract code in db")
	}
}

func (worker *BsWorker) GetInitType(wpwc *vm.WorkerProcWithCallback) string {
	txType := wpwc.WorkProc.ContractData.Transaction.GetType()
	if txType == types.TypeLuaContractInit {
		return "lua"
	} else {
		return "js"
	}
}

func (worker *BsWorker) isCommonTransaction(wpwc *vm.WorkerProcWithCallback) bool {
	txType := wpwc.WorkProc.ContractData.Transaction.GetType()
	if txType == types.TypeLuaContractInit || txType == types.TypeContractInvoke ||
		txType == types.TypeJSContractInit || txType == types.TypeContractQuery {
		return false
	}

	return true
}

func (worker *BsWorker) HandleCommonTransaction(wpwc *vm.WorkerProcWithCallback) {
	err := wpwc.WorkProc.L0Handler.Transfer(wpwc.WorkProc.ContractData.Transaction)
	if err != nil {
		log.Errorf("Transaction Exec fail, tx_hash: %+v, err: %s", wpwc.WorkProc.ContractData.Transaction.Hash(), err)
	}

	res := wpwc.Fn(err)
	if res.(bool) == true {

	}
}