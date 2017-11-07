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

package contract

import (
	"bytes"
	"errors"
	"math/big"
	"strings"

	"fmt"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/bocheninc/L0/core/types"
)

const (
	// ColumnFamily is the column family of contract state in db.
	ColumnFamily = "scontract"

	// globalStateKey is the key of global state.
	globalStateKey = "globalStateKey"

	// AdminKey is the key of admin account.
	AdminKey = "admin"

	// GlobalContractKey is the key of global contract.
	GlobalContractKey = "globalContract"

	// permissionPrefix is the prefix of data permission key.
	permissionPrefix = "permission."
)

type ILedgerSmartContract interface {
	GetTmpBalance(addr accounts.Address) (*state.Balance, error)
	Height() (uint32, error)
}

type ISmartConstract interface {
	GetGlobalState(key string) ([]byte, error)
	SetGlobalState(key string, value []byte) error
	DelGlobalState(key string) error

	GetState(key string) ([]byte, error)
	AddState(key string, value []byte)
	DelState(key string)
	GetByPrefix(prefix string) []*db.KeyValue
	GetByRange(startKey, limitKey string) []*db.KeyValue
	GetBalances(addr string) (*state.Balance, error)
	CurrentBlockHeight() uint32
	AddTransfer(fromAddr, toAddr string, assetID uint32, amount *big.Int, txType uint32)
	SmartContractFailed()
	SmartContractCommitted()
}

// State represents the account state
type SmartConstract struct {
	dbHandler     *db.BlockchainDB
	balancePrefix []byte
	columnFamily  string
	ledgerHandler ILedgerSmartContract
	stateExtra    *StateExtra

	height           uint32
	scAddr           string
	committed        bool
	currentTx        *types.Transaction
	smartContractTxs types.Transactions
}

// NewSmartConstract returns a new State
func NewSmartConstract(db *db.BlockchainDB, ledgerHandler ILedgerSmartContract) *SmartConstract {
	return &SmartConstract{
		dbHandler:     db,
		balancePrefix: []byte("sc_"),
		columnFamily:  ColumnFamily,
		ledgerHandler: ledgerHandler,
		stateExtra:    NewStateExtra(),
	}
}

// StartConstract start constract
func (sctx *SmartConstract) StartConstract(blockHeight uint32) {
	log.Debugf("startConstract() for blockHeight [%d]", blockHeight)
	if !sctx.InProgress() {
		log.Errorf("A tx [%d] is already in progress. Received call for begin of another smartcontract [%d]", sctx.height, blockHeight)
	}
	sctx.height = blockHeight
}

// StopContract start contract
func (sctx *SmartConstract) StopContract(blockHeight uint32) {
	log.Debugf("stopConstract() for blockHeight [%d]", blockHeight)
	if sctx.height != blockHeight {
		log.Errorf("Different blockHeight in contract-begin [%d] and contract-finish [%d]", sctx.height, blockHeight)
	}

	sctx.height = 0
	sctx.stateExtra = NewStateExtra()
}

// ExecTransaction exec transaction
func (sctx *SmartConstract) ExecTransaction(tx *types.Transaction, scAddr string) {
	sctx.committed = false
	sctx.currentTx = tx
	sctx.scAddr = scAddr
	sctx.smartContractTxs = make(types.Transactions, 0)
}

// GetGlobalState returns the global state.
func (sctx *SmartConstract) GetGlobalState(key string) ([]byte, error) {
	if !sctx.InProgress() {
		log.Errorf("State can be changed only in context of a block.")
	}

	value := sctx.stateExtra.get(globalStateKey, key)
	if len(value) == 0 {
		var err error
		scAddrkey := EnSmartContractKey(globalStateKey, key)
		value, err = sctx.dbHandler.Get(sctx.columnFamily, []byte(scAddrkey))
		if err != nil {
			return nil, fmt.Errorf("can't get date from db %s", err)
		}
	}
	return value, nil
}

func (sctx *SmartConstract) verifyPermission(key string) error {
	var dataAdmin []byte
	var err error
	switch {
	case key == AdminKey:
		fallthrough
	case key == GlobalContractKey:
		fallthrough
	case strings.Contains(key, permissionPrefix):
		dataAdmin, err = sctx.GetGlobalState(AdminKey)
		if err != nil {
			return err
		}
	default:
		permissionKey := permissionPrefix + key
		dataAdmin, err = sctx.GetGlobalState(permissionKey)
		if err != nil {
			return err
		}

		if len(dataAdmin) == 0 {
			dataAdmin, err = sctx.GetGlobalState(AdminKey)
			if err != nil {
				return err
			}
		}
	}

	sender := sctx.currentTx.Sender().Bytes()
	if len(dataAdmin) > 0 && !bytes.Equal(sender, dataAdmin) {
		return fmt.Errorf("change global state, permission denied")
	}
	return nil
}

// SetGlobalState sets the global state.
func (sctx *SmartConstract) SetGlobalState(key string, value []byte) error {
	if !sctx.InProgress() {
		log.Errorf("State can be changed only in context of a block.")
	}

	err := sctx.verifyPermission(key)
	if err != nil {
		return err
	}

	log.Debugf("SetGlobalState key=[%s], value=[%#v]", key, value)
	sctx.stateExtra.set(globalStateKey, key, value)
	return nil
}

// DelGlobalState deletes the global state.
func (sctx *SmartConstract) DelGlobalState(key string) error {
	if !sctx.InProgress() {
		log.Errorf("State can be changed only in context of a block.")
	}

	err := sctx.verifyPermission(key)
	if err != nil {
		return err
	}

	log.Debugf("DelGlobalState key=[%s]", key)
	sctx.stateExtra.delete(globalStateKey, key)
	return nil
}

// GetState get value
func (sctx *SmartConstract) GetState(key string) ([]byte, error) {
	if !sctx.InProgress() {
		log.Errorf("State can be changed only in context of a block.")
	}

	value := sctx.stateExtra.get(sctx.scAddr, key)
	if len(value) == 0 {
		var err error
		scAddrkey := EnSmartContractKey(sctx.scAddr, key)
		log.Debugf("sctx.scAddr: %s,%s,%s", sctx.scAddr, key, scAddrkey)
		value, err = sctx.dbHandler.Get(sctx.columnFamily, []byte(scAddrkey))
		if err != nil {
			return nil, fmt.Errorf("can't get date from db %s", err)
		}
	}

	return value, nil
}

// AddState put key-value into cache
func (sctx *SmartConstract) AddState(key string, value []byte) {
	log.Debugf("PutState smartcontract=[%x], key=[%s], value=[%#v]", sctx.scAddr, key, value)
	if !sctx.InProgress() {
		log.Errorf("State can be changed only in context of a block.")
	}

	sctx.stateExtra.set(sctx.scAddr, key, value)
}

// DelState remove key-value
func (sctx *SmartConstract) DelState(key string) {
	if !sctx.InProgress() {
		log.Errorf("State can be changed only in context of a block.")
	}

	sctx.stateExtra.delete(sctx.scAddr, key)
}

func (sctx *SmartConstract) GetByPrefix(prefix string) []*db.KeyValue {
	if !sctx.InProgress() {
		log.Errorf("State can be changed only in context of a block.")
	}
	scAddrkey := EnSmartContractKey(sctx.scAddr, prefix)
	cacheValues := sctx.stateExtra.getByPrefix(sctx.scAddr, prefix)
	dbValues := sctx.dbHandler.GetByPrefix(sctx.columnFamily, []byte(scAddrkey))

	return sctx.getKeyValues(cacheValues, dbValues)
}

func (sctx *SmartConstract) GetByRange(startKey, limitKey string) []*db.KeyValue {
	if !sctx.InProgress() {
		log.Errorf("State can be changed only in context of a block.")
	}

	scAddrStartKey := EnSmartContractKey(sctx.scAddr, startKey)
	scAddrlimitKey := EnSmartContractKey(sctx.scAddr, limitKey)
	cacheValues := sctx.stateExtra.getByRange(sctx.scAddr, startKey, limitKey)
	dbValues := sctx.dbHandler.GetByRange(sctx.columnFamily, []byte(scAddrStartKey), []byte(scAddrlimitKey))

	return sctx.getKeyValues(cacheValues, dbValues)
}

func (sctx *SmartConstract) getKeyValues(cacheKVs []*db.KeyValue, dbKVS []*db.KeyValue) []*db.KeyValue {
	var keyValues []*db.KeyValue

	cacheValuesMap := make(map[string]*db.KeyValue)
	for _, v := range cacheKVs {
		_, key := DeSmartContractKey(string(v.Key))
		cacheValuesMap[key] = v
		v.Key = []byte(key)
	}

	for _, v := range dbKVS {
		_, key := DeSmartContractKey(string(v.Key))
		if _, ok := cacheValuesMap[key]; !ok {
			v.Key = []byte(key)
			keyValues = append(keyValues, v)
		}
	}
	return append(keyValues, cacheKVs...)
}

// GetBalances get balance
func (sctx *SmartConstract) GetBalances(addr string) (*state.Balance, error) {
	return sctx.ledgerHandler.GetTmpBalance(accounts.HexToAddress(addr))
}

// CurrentBlockHeight get currentBlockHeight
func (sctx *SmartConstract) CurrentBlockHeight() uint32 {
	height, err := sctx.ledgerHandler.Height()
	if err == nil {
		log.Errorf("can't read blockchain height")
	}

	return height
}

// SmartContractFailed execute smartContract fail
func (sctx *SmartConstract) SmartContractFailed() {
	sctx.committed = false
	log.Errorf("VM can't put state into L0")
}

// SmartContractCommitted execute smartContract successfully
func (sctx *SmartConstract) SmartContractCommitted() {
	sctx.committed = true
}

// AddTransfer add transfer to make new transaction
func (sctx *SmartConstract) AddTransfer(fromAddr, toAddr string, assetID uint32, amount *big.Int, txType uint32) {
	tx := types.NewTransaction(sctx.currentTx.Data.FromChain, sctx.currentTx.Data.ToChain, txType,
		sctx.currentTx.Data.Nonce, accounts.HexToAddress(fromAddr), accounts.HexToAddress(toAddr),
		assetID, amount, sctx.currentTx.Data.Fee, sctx.currentTx.Data.CreateTime)

	sctx.smartContractTxs = append(sctx.smartContractTxs, tx)
}

// InProgress
func (sctx *SmartConstract) InProgress() bool {
	return true
}

// FinishContractTransaction finish contract transaction
func (sctx *SmartConstract) FinishContractTransaction() (types.Transactions, error) {
	if !sctx.committed {
		return nil, errors.New("Execute VM Fail")
	}

	return sctx.smartContractTxs, nil
}

// AddChangesForPersistence put cache data into db
func (sctx *SmartConstract) AddChangesForPersistence(writeBatch []*db.WriteBatch) ([]*db.WriteBatch, error) {
	updateContractStateDelta := sctx.stateExtra.getUpdatedContractStateDelta()
	for _, smartContract := range updateContractStateDelta {
		smartContract.cache.ForEach(func(key, value []byte) bool {
			cv := &CacheKVs{}
			cv.deserialize(value)
			if cv.Optype == db.OperationDelete {
				log.Debugln("Contract Del: ", string(key), string(cv.Value))
				writeBatch = append(writeBatch, db.NewWriteBatch(sctx.columnFamily, db.OperationDelete, key, cv.Value))
			} else if cv.Optype == db.OperationPut {
				log.Debugln("Contract Put: ", string(key), string(cv.Value))
				writeBatch = append(writeBatch, db.NewWriteBatch(sctx.columnFamily, db.OperationPut, key, cv.Value))
			} else {
				log.Errorf("invalid method ...")
			}
			return true
		})
	}

	return writeBatch, nil
}
