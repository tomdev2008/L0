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
	"github.com/bocheninc/L0/nvm"
	"github.com/bocheninc/L0/nvm/luavm"
	"github.com/bocheninc/L0/nvm/jsvm"
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


func NewBsWorker(conf *nvm.Config) *BsWorker {
	bsWorker := &BsWorker{
		jsWorker: jsvm.NewJsWorker(conf),
		luaWorker:luavm.NewLuaWorker(conf),
	}

	return bsWorker
}


func (worker *BsWorker) VmJob(data interface{}) interface{} {
	txType := "common"
	var err error
	workProcWithCallback := data.(*nvm.WorkerProcWithCallback)

	if !worker.isCommonTransaction(workProcWithCallback) {
		if strings.Contains(workProcWithCallback.WorkProc.PreMethod, "Invoke") {
			txType, err = worker.GetInvokeType(workProcWithCallback)
			log.Debugf("txType: %+v, err: %+v", txType, err)
			if err != nil {
				log.Errorf("can't execute contract, err_msg: %+v", err.Error())
				workProcWithCallback.Fn(err)
				return nil
			}
		} else {
			txType = worker.GetInitType(workProcWithCallback)
		}
	}

	if strings.Contains(txType, "common") {
		worker.HandleCommonTransaction(workProcWithCallback)
	} else if strings.Contains(txType, "lua"){
		worker.luaWorker.VmJob(workProcWithCallback)
	} else {
		worker.jsWorker.VmJob(workProcWithCallback)
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

func (worker *BsWorker) GetInvokeType(wpwc *nvm.WorkerProcWithCallback) (string, error) {
	var err error
	cc := new(nvm.ContractCode)
	var code []byte
	if len(wpwc.WorkProc.ContractData.ContractAddr) == 0 {
		code, err = wpwc.WorkProc.L0Handler.GetGlobalState(params.GlobalContractKey)
	} else {
		code, err = wpwc.WorkProc.L0Handler.GetState(nvm.ContractCodeKey)
	}

	if len(code) != 0 && err == nil {
		contractCode, err := nvm.DoContractStateData(code)
		if err != nil {
			return "", fmt.Errorf("cat't find contract code in db, err: %+v", err)
		}
		err = json.Unmarshal(contractCode, cc)
		if err != nil {
			return "", fmt.Errorf("cat't find contract code in db, err: %+v", err)
		}
		log.Debugf("ccccctype: %+v", cc.Type)
		return cc.Type, nil
	} else {
		return "", errors.New("cat't find contract code in db")
	}
}

func (worker *BsWorker) GetInitType(wpwc *nvm.WorkerProcWithCallback) string {
	txType := wpwc.WorkProc.ContractData.Transaction.GetType()
	if txType == types.TypeLuaContractInit {
		return "lua"
	} else {
		return "js"
	}
}

func (worker *BsWorker) isCommonTransaction(wpwc *nvm.WorkerProcWithCallback) bool {
	txType := wpwc.WorkProc.ContractData.Transaction.GetType()
	if txType == types.TypeLuaContractInit || txType == types.TypeContractInvoke ||
		txType == types.TypeJSContractInit || txType == types.TypeContractQuery {
		return false
	}

	return true
}

func (worker *BsWorker) HandleCommonTransaction(wpwc *nvm.WorkerProcWithCallback) {
	err := wpwc.WorkProc.L0Handler.Transfer(wpwc.WorkProc.ContractData.Transaction)
	if err != nil {
		log.Debugf("Transaction Exec fail, tx_hash: %+v", wpwc.WorkProc.ContractData.Transaction.Hash())
	}

	res := wpwc.Fn(err)
	if res.(bool) == true {

	}
}