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
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/bocheninc/L0/core/types"
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

func NewContractData(tx *types.Transaction) *ContractData {
	cd := new(ContractData)
	cs := &types.ContractSpec{}
	err := utils.Deserialize(tx.Payload, cs)
	if err != nil {
		return cd
	}
	cd.ContractCode = string(cs.ContractCode)
	cd.ContractAddr = hex.EncodeToString(cs.ContractAddr)
	cd.ContractParams = cs.ContractParams
	cd.Transaction = tx

	return cd
}

type WorkerProc struct {
	ContractData     *ContractData
	L0Handler        ISmartConstract
	StateChangeQueue *stateQueue
	TransferQueue    *transferQueue
}

type WorkerProcWithCallback struct {
	WorkProc *WorkerProc
	Idx      int
	Fn       func(interface{}) interface{}
}

func (p *WorkerProc) CCallGetGlobalState(key string) ([]byte, error) {
	if err := CheckStateKey(key); err != nil {
		return nil, err
	}

	if v, ok := p.StateChangeQueue.stateMap[key]; ok {
		return v, nil
	}

	result, err := p.ccall("GetGlobalState", key)
	return result.([]byte), err
}

func (p *WorkerProc) CCallSetGlobalState(key string, value []byte) error {
	if err := CheckStateKeyValue(key, value); err != nil {
		return err
	}

	if _, err := p.ccall("SetGlobalState", key, value); err != nil {
		return err
	}

	p.StateChangeQueue.stateMap[key] = value
	p.StateChangeQueue.offer(&stateOpfunc{stateOpTypePut, key, value})
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
	p.StateChangeQueue.offer(&stateOpfunc{stateOpTypeDelete, key, nil})
	return nil
}

func (p *WorkerProc) CCallGetState(key string) ([]byte, error) {
	if err := CheckStateKey(key); err != nil {
		return nil, err
	}
	if v, ok := p.StateChangeQueue.stateMap[key]; ok {
		return v, nil
	}

	result, err := p.ccall("GetState", key)
	return result.([]byte), err
}

func (p *WorkerProc) CCallPutState(key string, value []byte) error {
	if err := CheckStateKeyValue(key, value); err != nil {
		return err
	}

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

	result, err := p.ccall("GetByRange", startKey, limitKey)
	return result.([]*db.KeyValue), err
}

func (p *WorkerProc) CCallComplexQuery(key string) ([]byte, error) {
	if err := CheckStateKey(key); err != nil {
		return nil, err
	}

	result, err := p.ccall("ComplexQuery", key)
	return result.([]byte), err
}

func (p *WorkerProc) CCallGetBalance(addr string, assetID uint32) (*big.Int, error) {
	if err := CheckAddr(addr); err != nil {
		return nil, err
	}
	if v, ok := p.TransferQueue.balancesMap[addr]; ok {
		return v.Amounts[assetID], nil
	}

	result, err := p.ccall("GetBalance", addr, assetID)
	return result.(*big.Int), err
}

func (p *WorkerProc) CCallGetBalances(addr string) (*state.Balance, error) {
	if err := CheckAddr(addr); err != nil {
		return nil, err
	}
	if v, ok := p.TransferQueue.balancesMap[addr]; ok {
		return v, nil
	}

	result, err := p.ccall("GetBalances", addr)
	return result.(*state.Balance), err
}

func (p *WorkerProc) CCallCurrentBlockHeight() (uint32, error) {
	result, err := p.ccall("CurrentBlockHeight")
	return result.(uint32), err
}

func (p *WorkerProc) CCallTransfer(recipientAddr string, id, amount int64, fee int64) error {
	log.Debugf("CCallTransfer recipientAddr:%s, id:%d, amount:%d, fee:%d\n", recipientAddr, id, amount, fee)
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
		result, err := p.ccall("GetBalance", contractAddr, uint32(id))
		if err != nil || result == nil {
			return fmt.Errorf("get balance error -- %s", err)
		}
		contractBalances.Amounts[uint32(id)] = result.(*big.Int)
	}

	if b, ok := contractBalances.Amounts[uint32(id)]; !ok || b.Int64() < amount {
		return errors.New("balances not enough")
	}

	recipientBalances := state.NewBalance()
	if v, ok := p.TransferQueue.balancesMap[recipientAddr]; ok { // get recipient balances from cache
		recipientBalances = v
	} else { // get recipient balances from parent proc
		result, err := p.ccall("GetBalance", recipientAddr, uint32(id))
		if err != nil || result == nil {
			return errors.New("get balance error")
		}
		recipientBalances.Amounts[uint32(id)] = result.(*big.Int)
	}

	contractBalances.Amounts[uint32(id)].Sub(contractBalances.Amounts[uint32(id)], big.NewInt(amount))
	p.TransferQueue.balancesMap[contractAddr] = contractBalances

	if recipientBalances.Amounts[uint32(id)] == nil {
		recipientBalances.Amounts[uint32(id)] = big.NewInt(0)
	}
	recipientBalances.Amounts[uint32(id)].Add(recipientBalances.Amounts[uint32(id)], big.NewInt(amount))
	p.TransferQueue.balancesMap[recipientAddr] = recipientBalances
	p.TransferQueue.offer(&transferOpfunc{fee, contractAddr, recipientAddr, uint32(id), amount})

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
		if _, err := p.ccall("AddTransfer", txOP.from, txOP.to, txOP.id, txOP.amount, txOP.fee); err != nil {
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
		} else if stateOP.optype == stateOpTypeDelete {
			if _, err := p.ccall("DelState", nil, stateOP.key); err != nil {
				return err
			}
		}
	}

	return nil
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
		return nil, p.L0Handler.PutGlobalState(key, value)
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
		p.L0Handler.PutState(key, value)
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
		return p.L0Handler.GetByPrefix(prefix)

	case "GetByRange":
		if !p.checkParamsCnt(2, params...) {
			return nil, ErrNoValidParamsCnt
		}

		startKey := params[0].(string)
		limitKey := params[1].(string)

		return p.L0Handler.GetByRange(startKey, limitKey)
	case "GetBalance":
		if !p.checkParamsCnt(2, params...) {
			return nil, ErrNoValidParamsCnt
		}

		addr := params[0].(string)
		assetID := params[1].(uint32)
		return p.L0Handler.GetBalance(addr, assetID)

	case "GetBalances":
		if !p.checkParamsCnt(1, params...) {
			return nil, ErrNoValidParamsCnt
		}

		addr := params[0].(string)
		return p.L0Handler.GetBalances(addr)

	case "CurrentBlockHeight":
		height := p.L0Handler.GetCurrentBlockHeight()
		return height, nil

	case "AddTransfer":
		if !p.checkParamsCnt(5, params...) {
			return nil, ErrNoValidParamsCnt
		}

		fromAddr := params[0].(string)
		toAddr := params[1].(string)
		assetID := params[2].(uint32)
		amount := params[3].(int64)
		fee := params[4].(int64)

		p.L0Handler.AddTransfer(fromAddr, toAddr, assetID, big.NewInt(amount), big.NewInt(fee))
		return true, nil
	}

	return false, errors.New("no method match:" + funcName)
}

func (p *WorkerProc) checkParamsCnt(wanted int, params ...interface{}) bool {
	if len(params) != wanted {
		log.Errorf("invalid param count, wanted: %+v, actual: %+v", wanted, len(params))
		return false
	}

	return true
}
