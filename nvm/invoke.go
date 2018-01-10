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

package nvm

import (
	"encoding/hex"
	"github.com/bocheninc/L0/core/types"
	"math/big"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"errors"
	"github.com/bytom/blockchain/asset"
)

var (
	ErrNoValidParamsCnt = errors.New("invalid param count")
)

type ContractCode struct {
	Code []byte
	Type string
}


type ContractData struct {
	ContractCode   string
	ContractAddr   string
	ContractParams []string
	Transaction    *types.Transaction
}

func NewContractData(tx *types.Transaction, cs *types.ContractSpec, contractCode string) *ContractData {
	cd := new(ContractData)
	cd.ContractCode = contractCode
	cd.ContractAddr = hex.EncodeToString(cs.ContractAddr)
	cd.ContractParams = cs.ContractParams
	cd.Transaction = tx

	return cd
}

type WorkerProc struct {
	ContractData     *ContractData
	PreMethod	     string
	L0Handler        ISmartConstract
	StateChangeQueue *stateQueue
	TransferQueue    *transferQueue
}


type WorkerProcWithCallback struct {
	WorkProc *WorkerProc
	Fn func(interface{}) interface{}
}
/************************** call for parent proc (L0 proc) ******************************/
//
//func PCallPreInitContract(cd *ContractData, handler ISmartConstract) (bool, error) {
//	var success bool
//	err := pcall("PreInitContract", cd, handler, &success)
//	return success, err
//}
//
//func PCallRealInitContract(cd *ContractData, handler ISmartConstract) (bool, error) {
//	var success bool
//	err := pcall("RealInitContract", cd, handler, &success)
//	return success, err
//}
//
//func PCallPreExecute(cd *ContractData, handler ISmartConstract) (bool, error) {
//	var success bool
//	err := pcall("PreExecute", cd, handler, &success)
//	return success, err
//}
//
//func PCallRealExecute(cd *ContractData, handler ISmartConstract) (bool, error) {
//	var success bool
//	err := pcall("RealExecute", cd, handler, &success)
//	return success, err
//}
//
//func  PCallQueryContract(cd *ContractData, handler ISmartConstract) ([]byte, error) {
//	err := pcall("QueryContract", cd, handler, &result)
//	return result, err
//}
//
//func pcall(preMethod string, cd *ContractData, handler ISmartConstract) {//, callback func(data interface{}) interface{}) {
//
//}

/************************** call for child proc (vm proc) ******************************/
func (p *WorkerProc) CCallGetGlobalState(key string) ([]byte, error) {
	if err := CheckStateKey(key); err != nil {
		return nil, err
	}

	if v, ok := p.StateChangeQueue.stateMap[key]; ok {
		return v, nil
	}

	// call parent proc
	result, err := p.ccall("GetGlobalState", key)
	return result.([]byte), err
}

func (p *WorkerProc) CCallSetGlobalState(key string, value []byte) error {
	if err := CheckStateKeyValue(key, value); err != nil {
		return err
	}

	if  _, err := p.ccall("SetGlobalState", key, value); err != nil {
		return err
	}

	p.StateChangeQueue.stateMap[key] = value
	return nil
}

func (p *WorkerProc) CCallDelGlobalState(key string) error {
	if err := CheckStateKey(key); err != nil {
		return err
	}

	if _, err := p.ccall("DelGlobalState", key); err != nil {
		return err
	}

	p.StateChangeQueue.stateMap[key] = nil
	return nil
}

func (p *WorkerProc) CCallGetState(key string) ([]byte, error) {
	if err := CheckStateKey(key); err != nil {
		return nil, err
	}
	if v, ok := p.StateChangeQueue.stateMap[key]; ok {
		return v, nil
	}

	// call parent proc
	result, err := p.ccall("GetState", key)
	return result.([]byte), err
}

func (p *WorkerProc) CCallComplexQuery(key string) ([]byte, error) {
	if err := CheckStateKey(key); err != nil {
		return nil, err
	}

	// call parent proc
	result, err := p.ccall("ComplexQuery", key)
	return result.([]byte), err
}

func (p *WorkerProc) CCallPutState(key string, value []byte) error {
	if err := CheckStateKeyValue(key, value); err != nil {
		return err
	}

	//log.Debugf("====p: %p", p)
	p.StateChangeQueue.stateMap[key] = value
	p.StateChangeQueue.offer(&stateOpfunc{stateOpTypePut, key, value})
	return nil
}

func (p *WorkerProc) CCallDelState(key string) error {
	if err := CheckStateKey(key); err != nil {
		return err
	}

	p.StateChangeQueue.stateMap[key] = nil
	p.StateChangeQueue.offer(&stateOpfunc{stateOpTypeDelete, key, nil})
	return nil
}

func (p *WorkerProc) CCallGetByPrefix(key string) ([]*db.KeyValue, error) {
	if err := CheckStateKey(key); err != nil {
		return nil, err
	}

	// call parent proc
	result, err := p.ccall("GetByPrefix", key)
	return result.([]*db.KeyValue), err
}

func (p *WorkerProc) CCallGetByRange(startKey string, limitKey string) ([]*db.KeyValue, error) {
	if err := CheckStateKey(startKey); err != nil {
		return nil, err
	}

	if err := CheckStateKey(limitKey); err != nil {
		return nil, err
	}

	// call parent proc
	result, err := p.ccall("GetByRange", startKey, limitKey)
	return result.([]*db.KeyValue), err
}

func (p *WorkerProc) CCallGetBalances(addr string) (map[uint32]*big.Int, error) {
	if err := CheckAddr(addr); err != nil {
		return nil, err
	}
	if v, ok := p.TransferQueue.balancesMap[addr]; ok {
		return v, nil
	}

	// call parent proc
	result, err := p.ccall("GetBalances", addr)
	return result.(map[uint32]*big.Int), err
}

func (p *WorkerProc) CCallGetBalance(addr string, assetID uint32) (*big.Int, error) {
	if err := CheckAddr(addr); err != nil {
		return nil, err
	}
	if v, ok := p.TransferQueue.balancesMap[addr]; ok {
		return v, nil
	}

	// call parent proc
	result, err := p.ccall("GetBalance", addr, assetID)
	return result.(*big.Int), err
}

func (p *WorkerProc) CCallCurrentBlockHeight() (uint32, error) {
	result, err := p.ccall("CurrentBlockHeight")
	return result.(uint32), err
}

func (p *WorkerProc) CCallTransfer(recipientAddr string, id, amount int64, txType uint32) error {
	log.Debugf("CCallTransfer recipientAddr:%s, id:%d, amount:%d, type:%d\n", recipientAddr, id, amount, txType)
	if err := CheckAddr(recipientAddr); err != nil {
		return err
	}
	if amount <= 0 {
		return errors.New("amount must above 0")
	}

	contractAddr := p.ContractData.ContractAddr
	contractBalances := state.NewBalance()
	if v, ok := p.TransferQueue.balancesMap[contractAddr]; ok { // get contract balances from cache
		contractBalances = v
	} else { // get contract balances from parent proc
		result, err := p.ccall("GetBalances", contractAddr)
		if err != nil {
			return errors.New("get balances error")
		}
		contractBalances = result.(*state.Balance)
	}

	if contractBalances.Amounts[uint32(id)].Int64() < amount {
		return errors.New("balances not enough")
	}

	recipientBalances := state.NewBalance()
	if v, ok := p.TransferQueue.balancesMap[recipientAddr]; ok { // get recipient balances from cache
		recipientBalances = v
	} else { // get recipient balances from parent proc
		result, err := p.ccall("GetBalances", recipientAddr)
		if err != nil {
			return errors.New("get balances error")
		}
		recipientBalances = result.(*state.Balance)
	}

	contractBalances.Amounts[uint32(id)].Sub(contractBalances.Amounts[uint32(id)], big.NewInt(amount))
	p.TransferQueue.balancesMap[contractAddr] = contractBalances

	if recipientBalances.Amounts[uint32(id)] == nil {
		recipientBalances.Amounts[uint32(id)] = big.NewInt(0)
	}
	recipientBalances.Amounts[uint32(id)].Add(recipientBalances.Amounts[uint32(id)], big.NewInt(amount))
	p.TransferQueue.balancesMap[recipientAddr] = recipientBalances
	p.TransferQueue.offer(&transferOpfunc{txType, contractAddr, recipientAddr, uint32(id), amount})

	return nil
}

func (p *WorkerProc) CCallSmartContractFailed() error {
	_, err := p.ccall("SmartContractFailed")
	return err
}

func (p *WorkerProc) CCallSmartContractCommitted() error {
	_, err := p.ccall("SmartContractCommitted")
	return err
}

func (p *WorkerProc) CCallCommit() error {
	for {
		txOP := p.TransferQueue.poll()
		if txOP == nil {
			break
		}

		// call parent proc for real transfer
		if _, err := p.ccall("AddTransfer", txOP.from, txOP.to, txOP.id, txOP.amount, txOP.txType); err != nil {
			return err
		}
		// log.Debugf("commit -> AddTransfer from:%s, to:%s, amount:%d, type:%d\n", txOP.from, txOP.to, txOP.amount, txOP.txType)
	}

	for {
		stateOP := p.StateChangeQueue.poll()
		if stateOP == nil {
			break
		}

		if stateOP.optype == stateOpTypePut {
			if _, err := p.ccall("PutState", stateOP.key, stateOP.value); err != nil {
				return err
			}
			// log.Debugf("commit -> AddState key:%s", stateOP.key)
		} else if stateOP.optype == stateOpTypeDelete {
			if _, err := p.ccall("DelState", nil, stateOP.key); err != nil {
				return err
			}
			// log.Debugf("commit -> DelState key:%s", stateOP.key)
		}
	}

	return p.CCallSmartContractCommitted()
}

func (p *WorkerProc) ccall(funcName string, params ...interface{}) (interface{}, error) {
	//log.Debugf("request parent proc funcName:%s, params(%d): %+v \n", funcName, len(params), params)
	switch funcName {
	case "GetGlobalState":
		if !p.checkParamsCnt(1, params...) {
			return nil, ErrNoValidParamsCnt
		}
		return p.L0Handler.GetGlobalState(params[0].(string))
	case "SetGlobalState":
		if !p.checkParamsCnt(2, params...) {
			return nil, ErrNoValidParamsCnt
		}
		key := params[0].(string)
		value := params[1].([]byte)
		return nil, p.L0Handler.SetGlobalState(key, value)
	case "DelGlobalState":
		if !p.checkParamsCnt(1, params...) {
			return nil, ErrNoValidParamsCnt
		}
		return nil, p.L0Handler.DelGlobalState(params[0].(string))
	case "GetState":
		if !p.checkParamsCnt(1, params...) {
			return nil, ErrNoValidParamsCnt
		}
		return p.L0Handler.GetState(params[0].(string))

	case "ComplexQuery":
		if !p.checkParamsCnt(1, params...) {
			return nil, ErrNoValidParamsCnt
		}
		return p.L0Handler.ComplexQuery(params[0].(string))
	case "PutState":
		if !p.checkParamsCnt(2, params...) {
			return nil, ErrNoValidParamsCnt
		}
		key := params[0].(string)
		value := params[1].([]byte)
		p.L0Handler.AddState(key, value)
		return true, nil

	case "DelState":
		if !p.checkParamsCnt(1, params...) {
			return nil, ErrNoValidParamsCnt
		}
		p.L0Handler.DelState(params[0].(string))
		return true, nil
	case "GetByPrefix":
		if !p.checkParamsCnt(1, params...) {
			return nil, ErrNoValidParamsCnt
		}

		prefix := params[0].(string)
		return p.L0Handler.GetByPrefix(prefix), nil

	case "GetByRange":
		if !p.checkParamsCnt(2, params...) {
			return nil, ErrNoValidParamsCnt
		}

		startKey := params[0].(string)
		limitKey := params[1].(string)

		return p.L0Handler.GetByRange(startKey, limitKey), nil

	case "GetBalances":
		if !p.checkParamsCnt(1, params...) {
			return nil, ErrNoValidParamsCnt
		}

		addr := params[0].(string)
		return p.L0Handler.GetBalances(addr)

	case "CurrentBlockHeight":
		height := p.L0Handler.CurrentBlockHeight()
		return height, nil

	case "AddTransfer":
		if !p.checkParamsCnt(5, params...) {
			return nil, ErrNoValidParamsCnt
		}

		fromAddr := params[0].(string)
		toAddr := params[1].(string)
		assetID := params[2].(uint32)
		amount := params[3].(int64)
		txType := params[4].(uint32)

		p.L0Handler.AddTransfer(fromAddr, toAddr, assetID, big.NewInt(amount), txType)
		return true, nil

	case "SmartContractFailed":
		p.L0Handler.SmartContractFailed()
		return true, nil

	case "SmartContractCommitted":
		p.L0Handler.SmartContractCommitted()
		return true, nil

	}

	return false, errors.New("no method match:" + funcName)
}

func (p *WorkerProc)checkParamsCnt( wanted int, params ...interface{}) bool {
	if len(params) != wanted {
		log.Errorf("invalid param count, wanted: %+v, actual: %+v", wanted, len(params))
		return false
	}

	return true
}