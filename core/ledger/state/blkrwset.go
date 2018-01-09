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

package state

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/db/mongodb"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/ledger/contract/treap"
)

// NewBLKRWset create object
func NewBLKRWset(db *db.BlockchainDB) *BLKRWSet {
	return &BLKRWSet{
		chainCodeCF: "scontract",
		balanceCF:   "balance",
		assetCF:     "asset",
		dbHandler:   db,
	}
}

var assetIDKeyPrefix = "asset"
var assetIDKeySuffix = "$"

//BLKRWSet encapsulates the read-write set during transactions of block simulation
type BLKRWSet struct {
	chainCodeSet *KVRWSet
	chainCodeRW  sync.RWMutex
	balanceSet   *KVRWSet
	balanceRW    sync.RWMutex
	assetSet     *KVRWSet
	assetRW      sync.RWMutex

	dbHandler   *db.BlockchainDB
	chainCodeCF string
	balanceCF   string
	assetCF     string

	BlockNum uint64
}

// GetChainCodeState get state for chaincode address and key. If committed is false, this first looks in memory
// and if missing, pulls from db.  If committed is true, this pulls from the db only.
func (blk *BLKRWSet) GetChainCodeState(chaincodeAddr string, key string, committed bool) ([]byte, error) {
	blk.chainCodeRW.RLock()
	defer blk.chainCodeRW.RUnlock()
	ckey := ConstructCompositeKey(chaincodeAddr, key)
	if !committed {
		if kvw, ok := blk.chainCodeSet.Writes[ckey]; ok {
			return kvw.Value, nil
		}

		if kvr, ok := blk.chainCodeSet.Reads[ckey]; ok {
			return kvr.Value, nil
		}
	}
	return blk.dbHandler.Get(blk.chainCodeCF, []byte(ckey))
}

// GetChainCodeStateByRange get state for chaincode address and key. If committed is false, this first looks in memory
// and if missing, pulls from db.  If committed is true, this pulls from the db only.
func (blk *BLKRWSet) GetChainCodeStateByRange(chaincodeAddr string, startKey string, endKey string, committed bool) (map[string][]byte, error) {
	blk.chainCodeRW.RLock()
	defer blk.chainCodeRW.RUnlock()
	chaincodePrefix := ConstructCompositeKey(chaincodeAddr, "")
	ckeyStart := ConstructCompositeKey(chaincodeAddr, startKey)
	ckeyEnd := ConstructCompositeKey(chaincodeAddr, endKey)
	ret := make(map[string][]byte)
	if len(endKey) > 0 {
		dbValues := blk.dbHandler.GetByRange(blk.chainCodeCF, []byte(ckeyStart), []byte(ckeyEnd))
		for _, kv := range dbValues {
			ret[string(kv.Key)] = kv.Value
		}
	} else {
		dbValues := blk.dbHandler.GetByPrefix(blk.chainCodeCF, []byte(ckeyStart))
		for _, kv := range dbValues {
			ret[string(kv.Key)] = kv.Value
		}
	}

	if !committed {
		cache := treap.NewImmutable()
		for ckey, kvr := range blk.chainCodeSet.Reads {
			if strings.HasPrefix(ckey, chaincodePrefix) {
				cache.Put([]byte(ckey), kvr.Value)
			}
		}
		for ckey, kvw := range blk.chainCodeSet.Writes {
			if strings.HasPrefix(ckey, chaincodePrefix) {
				cache.Put([]byte(ckey), kvw.Value)
			}
		}

		if len(endKey) > 0 {
			for iter := cache.Iterator([]byte(ckeyStart), []byte(ckeyEnd)); iter.Next(); {
				if val := iter.Value(); val != nil {
					ret[string(iter.Key())] = val
				}
			}
		} else {
			for iter := cache.Iterator([]byte(startKey), nil); iter.Next(); {
				if !bytes.HasPrefix(iter.Key(), []byte(startKey)) {
					break
				}
				if val := iter.Value(); val != nil {
					ret[string(iter.Key())] = val
				}
			}
		}
	}
	return ret, nil
}

// SetChainCodeState set state to given value for chaincode address and key. Does not immideatly writes to DB
func (blk *BLKRWSet) SetChainCodeState(chaincodeAddr string, key string, value []byte) error {
	blk.chainCodeRW.Lock()
	defer blk.chainCodeRW.Unlock()
	ckey := ConstructCompositeKey(chaincodeAddr, key)
	blk.chainCodeSet.Writes[ckey] = &KVWrite{
		Value:    value,
		IsDelete: false,
	}
	return nil
}

// DelChainCodeState tracks the deletion of state for chaincode address and key. Does not immediately writes to DB
func (blk *BLKRWSet) DelChainCodeState(chaincodeAddr string, key string) {
	blk.chainCodeRW.Lock()
	defer blk.chainCodeRW.Unlock()
	ckey := ConstructCompositeKey(chaincodeAddr, key)
	blk.chainCodeSet.Writes[ckey] = &KVWrite{
		Value:    nil,
		IsDelete: true,
	}
}

// GetBalanceState get balance for address and assetID. If committed is false, this first looks in memory
// and if missing, pulls from db.  If committed is true, this pulls from the db only.
func (blk *BLKRWSet) GetBalanceState(addr string, assetID uint32, committed bool) (*big.Int, error) {
	blk.balanceRW.RLock()
	defer blk.balanceRW.RUnlock()
	var amount big.Int
	ckey := ConstructCompositeKey(addr, strconv.FormatUint(uint64(assetID), 10)+assetIDKeySuffix)
	if !committed {
		if kvw, ok := blk.balanceSet.Writes[ckey]; ok {
			if kvw.IsDelete {
				return nil, nil
			}
			return amount.SetBytes(kvw.Value), nil
		}

		if kvr, ok := blk.balanceSet.Reads[ckey]; ok {
			return amount.SetBytes(kvr.Value), nil
		}
	}
	value, err := blk.dbHandler.Get(blk.balanceCF, []byte(ckey))
	if err != nil {
		return nil, err
	}
	return amount.SetBytes(value), nil
}

// GetBalances get balances for address. If committed is false, this first looks in memory
// and if missing, pulls from db.  If committed is true, this pulls from the db only.
func (blk *BLKRWSet) GetBalances(addr string, committed bool) (map[uint32]*big.Int, error) {
	blk.balanceRW.RLock()
	defer blk.balanceRW.RUnlock()
	prefix := ConstructCompositeKey(addr, "")
	ret := make(map[string][]byte)
	dbValues := blk.dbHandler.GetByPrefix(blk.balanceCF, []byte(prefix))
	for _, kv := range dbValues {
		ret[string(kv.Key)] = kv.Value
	}
	if !committed {
		cache := treap.NewImmutable()
		for ckey, kvr := range blk.balanceSet.Reads {
			if strings.HasPrefix(ckey, prefix) {
				cache.Put([]byte(ckey), kvr.Value)
			}
		}
		for ckey, kvw := range blk.balanceSet.Writes {
			if strings.HasPrefix(ckey, prefix) {
				cache.Put([]byte(ckey), kvw.Value)
			}
		}

		for iter := cache.Iterator([]byte(prefix), nil); iter.Next(); {
			if !bytes.HasPrefix(iter.Key(), []byte(prefix)) {
				break
			}
			if val := iter.Value(); val != nil {
				ret[string(iter.Key())] = val
			}
		}
	}

	var amount big.Int
	balances := make(map[uint32]*big.Int)
	for k, v := range ret {
		if v != nil {
			assetID, err := strconv.ParseUint(k, 10, 32)
			if err != nil {
				return nil, err
			}
			balances[uint32(assetID)] = amount.SetBytes(v)
		}
	}
	return balances, nil
}

// SetBalacneState set balance to given value for chaincode address and key. Does not immideatly writes to DB
func (blk *BLKRWSet) SetBalacneState(addr string, assetID uint32, amount *big.Int) error {
	blk.balanceRW.Lock()
	defer blk.balanceRW.Unlock()
	value := amount.Bytes()
	ckey := ConstructCompositeKey(addr, strconv.FormatUint(uint64(assetID), 10)+assetIDKeySuffix)
	blk.balanceSet.Writes[ckey] = &KVWrite{
		Value:    value,
		IsDelete: false,
	}
	return nil
}

// DelBalanceState tracks the deletion of balance for chaincode address and key. Does not immediately writes to DB
func (blk *BLKRWSet) DelBalanceState(addr string, assetID uint32) {
	blk.balanceRW.Lock()
	defer blk.balanceRW.Unlock()
	ckey := ConstructCompositeKey(addr, strconv.FormatUint(uint64(assetID), 10)+assetIDKeySuffix)
	blk.balanceSet.Writes[ckey] = &KVWrite{
		Value:    nil,
		IsDelete: true,
	}
}

// GetAssetState get asset for assetID. If committed is false, this first looks in memory
// and if missing, pulls from db.  If committed is true, this pulls from the db only.
func (blk *BLKRWSet) GetAssetState(assetID uint32, committed bool) (*Asset, error) {
	blk.assetRW.RLock()
	defer blk.assetRW.RUnlock()
	assetInfo := &Asset{}
	ckey := ConstructCompositeKey(assetIDKeyPrefix, strconv.FormatUint(uint64(assetID), 10)+assetIDKeySuffix)
	if !committed {
		if kvw, ok := blk.assetSet.Writes[ckey]; ok {
			if kvw.IsDelete {
				return nil, nil
			}
			if err := utils.Deserialize(kvw.Value, assetInfo); err != nil {
				return nil, err
			}
			return assetInfo, nil
		}

		if kvr, ok := blk.assetSet.Reads[ckey]; ok {
			if err := utils.Deserialize(kvr.Value, assetInfo); err != nil {
				return nil, err
			}
			return assetInfo, nil
		}
	}
	value, err := blk.dbHandler.Get(blk.assetCF, []byte(ckey))
	if err != nil {
		return nil, err
	}
	if err := utils.Deserialize(value, assetInfo); err != nil {
		return nil, err
	}
	return assetInfo, nil
}

// GetAssets get assets. If committed is false, this first looks in memory
// and if missing, pulls from db.  If committed is true, this pulls from the db only.
func (blk *BLKRWSet) GetAssets(committed bool) (map[uint32]*Asset, error) {
	blk.assetRW.RLock()
	defer blk.assetRW.RUnlock()
	prefix := ConstructCompositeKey(assetIDKeyPrefix, "")
	ret := make(map[string][]byte)
	dbValues := blk.dbHandler.GetByPrefix(blk.assetCF, []byte(prefix))
	for _, kv := range dbValues {
		ret[string(kv.Key)] = kv.Value
	}
	if !committed {
		cache := treap.NewImmutable()
		for ckey, kvr := range blk.assetSet.Reads {
			if strings.HasPrefix(ckey, prefix) {
				cache.Put([]byte(ckey), kvr.Value)
			}
		}
		for ckey, kvw := range blk.assetSet.Writes {
			if strings.HasPrefix(ckey, prefix) {
				cache.Put([]byte(ckey), kvw.Value)
			}
		}

		for iter := cache.Iterator([]byte(prefix), nil); iter.Next(); {
			if !bytes.HasPrefix(iter.Key(), []byte(prefix)) {
				break
			}
			if val := iter.Value(); val != nil {
				ret[string(iter.Key())] = val
			}
		}
	}

	assetInfo := &Asset{}
	assets := make(map[uint32]*Asset)
	for _, v := range ret {
		if v != nil {
			if err := utils.Deserialize(v, assetInfo); err != nil {
				return nil, err
			}
			assets[assetInfo.ID] = assetInfo
		}
	}
	return assets, nil
}

// SetAssetState set balance to given value for assetID. Does not immideatly writes to DB
func (blk *BLKRWSet) SetAssetState(assetID uint32, assetInfo *Asset) error {
	blk.assetRW.Lock()
	defer blk.assetRW.Unlock()
	value := utils.Serialize(assetInfo)
	ckey := ConstructCompositeKey(assetIDKeyPrefix, strconv.FormatUint(uint64(assetID), 10)+assetIDKeySuffix)
	blk.assetSet.Writes[ckey] = &KVWrite{
		Value:    value,
		IsDelete: false,
	}
	return nil
}

// DelAssetState tracks the deletion of asset for assetID. Does not immediately writes to DB
func (blk *BLKRWSet) DelAssetState(assetID uint32) {
	blk.assetRW.Lock()
	defer blk.assetRW.Unlock()
	ckey := ConstructCompositeKey(assetIDKeyPrefix, strconv.FormatUint(uint64(assetID), 10)+assetIDKeySuffix)
	blk.assetSet.Writes[ckey] = &KVWrite{
		Value:    nil,
		IsDelete: true,
	}
}

// ApplyChanges merges delta
func (blk *BLKRWSet) ApplyChanges() ([]*db.WriteBatch, error) {
	blk.chainCodeRW.RLock()
	defer blk.chainCodeRW.RUnlock()
	blk.assetRW.RLock()
	defer blk.assetRW.RUnlock()
	blk.balanceRW.RLock()
	defer blk.balanceRW.RUnlock()

	writeBatchs := make([]*db.WriteBatch, 0)
	for ckey, wset := range blk.chainCodeSet.Writes {
		if wset.IsDelete {
			writeBatchs = append(writeBatchs, db.NewWriteBatch(blk.chainCodeCF, db.OperationDelete, []byte(ckey), nil, blk.chainCodeCF))
		} else {
			writeBatchs = append(writeBatchs, db.NewWriteBatch(blk.chainCodeCF, db.OperationPut, []byte(ckey), wset.Value, blk.chainCodeCF))
		}
	}

	for ckey, wset := range blk.assetSet.Writes {
		if wset.IsDelete {
			writeBatchs = append(writeBatchs, db.NewWriteBatch(blk.assetCF, db.OperationDelete, []byte(ckey), nil, blk.assetCF))
		} else {
			writeBatchs = append(writeBatchs, db.NewWriteBatch(blk.assetCF, db.OperationPut, []byte(ckey), wset.Value, blk.assetCF))
		}
	}

	for ckey, wset := range blk.balanceSet.Writes {
		if wset.IsDelete {
			writeBatchs = append(writeBatchs, db.NewWriteBatch(blk.balanceCF, db.OperationDelete, []byte(ckey), nil, blk.balanceCF))
		} else {
			writeBatchs = append(writeBatchs, db.NewWriteBatch(blk.balanceCF, db.OperationPut, []byte(ckey), wset.Value, blk.balanceCF))
		}
	}

	blk.assetSet = &KVRWSet{}
	blk.balanceSet = &KVRWSet{}
	blk.chainCodeSet = &KVRWSet{}
	return writeBatchs, nil
}

func (blk *BLKRWSet) merge(chainCodeSet *KVRWSet, assetSet *KVRWSet, balanceSet *KVRWSet) error {
	blk.chainCodeRW.Lock()
	defer blk.chainCodeRW.Unlock()
	blk.assetRW.Lock()
	defer blk.assetRW.Unlock()
	blk.balanceRW.Lock()
	defer blk.balanceRW.Unlock()

	for ckey, rset := range chainCodeSet.Reads {
		if trset, ok := blk.chainCodeSet.Reads[ckey]; ok {
			if bytes.Compare(trset.Value, rset.Value) != 0 {
				chaincodeAddr, key := DecodeCompositeKey(ckey)
				return fmt.Errorf("chaincode readset conflict -- %s %s", chaincodeAddr, key)
			}
		}
	}

	for ckey, rset := range assetSet.Reads {
		if trset, ok := blk.assetSet.Reads[ckey]; ok {
			if bytes.Compare(trset.Value, rset.Value) != 0 {
				_, key := DecodeCompositeKey(ckey)
				return fmt.Errorf("asset readset conflict -- %s", key)
			}
		}
	}

	for ckey, rset := range balanceSet.Reads {
		if trset, ok := blk.balanceSet.Reads[ckey]; ok {
			if bytes.Compare(trset.Value, rset.Value) != 0 {
				addr, key := DecodeCompositeKey(ckey)
				return fmt.Errorf("balance readset conflict -- %s %s", addr, key)
			}
		}
	}

	for ckey, wset := range chainCodeSet.Writes {
		blk.chainCodeSet.Writes[ckey] = wset
	}

	for ckey, wset := range assetSet.Writes {
		blk.assetSet.Writes[ckey] = wset
	}

	for ckey, wset := range balanceSet.Writes {
		blk.balanceSet.Writes[ckey] = wset
	}

	return nil
}

func (blk *BLKRWSet) RegisterColumn(mdb *mongodb.Mdb) {
	mdb.RegisterCollection(blk.chainCodeCF)
	mdb.RegisterCollection(blk.assetCF)
	mdb.RegisterCollection(blk.balanceCF)
}
