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

// Package vm the contract execute environment
package vm

import (
	"sync"

	"math/big"

	"errors"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/ledger/contract"
	"github.com/bocheninc/L0/core/types"
)

type ContractCode struct {
	Code []byte
	Type string
}

const (
	contractCodeKey = "__CONTRACT_CODE_KEY__"
)

var (
	luavmProc *VMProc
	jsvmProc  *VMProc
	locker    sync.Mutex

	zeroAddr = accounts.Address{}
)

// PreExecute execute contract but not commit change(balances and state)
func PreExecute(tx *types.Transaction, cs *types.ContractSpec, handler contract.ISmartConstract) (bool, error) {
	ret, err := execute(tx, cs, handler, false)
	if err != nil {
		return false, err
	}
	return ret.(bool), err
}

// RealExecute execute contract and commit change(balances and state)
func RealExecute(tx *types.Transaction, cs *types.ContractSpec, handler contract.ISmartConstract) (bool, error) {
	ret, err := execute(tx, cs, handler, true)
	if err != nil {
		return false, err
	}

	return ret.(bool), err
}

func Query(tx *types.Transaction, cs *types.ContractSpec, handler contract.ISmartConstract) ([]byte, error) {
	ret, err := execute(tx, cs, handler, true)
	if err != nil {
		return nil, err
	}

	return ret.([]byte), err
}

func execute(tx *types.Transaction, cs *types.ContractSpec, handler contract.ISmartConstract, realExec bool) (interface{}, error) {

	contractCode, contractType, err := getContractCode(cs, tx.GetType(), handler)
	if err != nil {
		return false, err
	}

	var vm *VMProc

	if err := initVMProc(contractType); err != nil {
		return false, err
	}

	// 根据不同的语言调用不同的vm
	switch contractType {
	case "luavm":
		vm = luavmProc
	case "jsvm":
		vm = jsvmProc
	}

	cd := NewContractData(tx, cs, contractCode)

	switch tx.GetType() {
	case types.TypeJSContractInit:
		if realExec {
			handler.AddState(contractCodeKey, utils.Serialize(&ContractCode{Code: cs.ContractCode, Type: "jsvm"})) // add js contract code into state
			return vm.PCallRealInitContract(cd, handler)
		}
		return vm.PCallPreInitContract(cd, handler)
	case types.TypeLuaContractInit:
		if realExec {
			handler.AddState(contractCodeKey, utils.Serialize(&ContractCode{Code: cs.ContractCode, Type: "luavm"})) // add lua contract code into state
			return vm.PCallRealInitContract(cd, handler)
		}
		return vm.PCallPreInitContract(cd, handler)
	case types.TypeContractInvoke:
		if realExec {
			return vm.PCallRealExecute(cd, handler)
		}
		return vm.PCallPreExecute(cd, handler)
	case types.TypeContractQuery:
		return vm.PCallQueryContract(cd, handler)
	}

	return false, errors.New("Transaction type error")
}

func initVMProc(contractType string) error {
	var err error
	switch contractType {
	case "jsvm":
		if jsvmProc == nil {
			locker.Lock()
			if jsvmProc == nil {
				if jsvmProc, err = NewVMProc(VMConf.JSVMExeFilePath); err == nil {
					jsvmProc.SetRequestHandle(requestHandle)
					jsvmProc.Selector()
				} else {
					log.Error("create jsvm proc error", err)
				}
			}
			locker.Unlock()
		}
	case "luavm":
		// create lua vm
		if luavmProc == nil {
			locker.Lock()
			if luavmProc == nil {
				if luavmProc, err = NewVMProc(VMConf.LuaVMExeFilePath); err == nil {
					luavmProc.SetRequestHandle(requestHandle)
					luavmProc.Selector()
				} else {
					log.Error("create luavm proc error", err)
				}
			}
			locker.Unlock()
		}
	}

	return err
}

func getContractCode(cs *types.ContractSpec, txType uint32, handler contract.ISmartConstract) (string, string, error) {

	code := cs.ContractCode
	if code != nil && len(code) > 0 {
		if txType == types.TypeJSContractInit {
			return string(code), "jsvm", nil
		}
		return string(code), "luavm", nil
	}

	cc := new(ContractCode)
	code, err := handler.GetState(contractCodeKey)
	if len(code) != 0 && err == nil {
		utils.Deserialize(code, cc)
		return string(cc.Code), cc.Type, nil
	} else if len(code) == 0 && err == nil {
		return "", "", errors.New("cat't find contract code in db")
	}
	return "", "", err
}

func requestHandle(vmproc *VMProc, req *InvokeData) (interface{}, error) {
	// log.Debugf("request parent proc funcName:%s\n", req.FuncName)
	switch req.FuncName {
	case "GetGlobalState":
		var key string
		if err := req.DecodeParams(&key); err != nil {
			return nil, err
		}
		return vmproc.L0Handler.GetGlobalState(key)
	case "SetGlobalState":
		var key string
		var value []byte
		if err := req.DecodeParams(&key, &value); err != nil {
			return nil, err
		}
		return nil, vmproc.L0Handler.SetGlobalState(key, value)
	case "DelGlobalState":
		var key string
		if err := req.DecodeParams(&key); err != nil {
			return nil, err
		}
		return nil, vmproc.L0Handler.DelGlobalState(key)
	case "GetState":
		var key string
		if err := req.DecodeParams(&key); err != nil {
			return nil, err
		}
		return vmproc.L0Handler.GetState(key)

	case "PutState":
		var key string
		var value []byte
		if err := req.DecodeParams(&key, &value); err != nil {
			return nil, err
		}
		vmproc.L0Handler.AddState(key, value)
		return true, nil

	case "DelState":
		var key string
		if err := req.DecodeParams(&key); err != nil {
			return nil, err
		}
		vmproc.L0Handler.DelState(key)
		return true, nil

	case "GetBalances":
		var addr string
		if err := req.DecodeParams(&addr); err != nil {
			return nil, err
		}
		b, err := vmproc.L0Handler.GetBalances(addr)
		return b.Int64(), err

	case "CurrentBlockHeight":
		height := vmproc.L0Handler.CurrentBlockHeight()
		return height, nil

	case "AddTransfer":
		var (
			fromAddr, toAddr string
			amount           int64
			txType           uint32
		)
		if err := req.DecodeParams(&fromAddr, &toAddr, &amount, &txType); err != nil {
			return nil, err
		}
		vmproc.L0Handler.AddTransfer(fromAddr, toAddr, big.NewInt(amount), txType)
		return true, nil

	case "SmartContractFailed":
		vmproc.L0Handler.SmartContractFailed()
		return true, nil

	case "SmartContractCommitted":
		vmproc.L0Handler.SmartContractCommitted()
		return true, nil

	}

	return false, errors.New("no method match:" + req.FuncName)
}
