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
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/bocheninc/L0/core/ledger/state/treap"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/base/utils"
)

// NewTXRWSet create object
func NewTXRWSet(blk *BLKRWSet) *TXRWSet {
	return &TXRWSet{
		block: blk,
	}
}

// TXRWSet encapsulates the read-write set during transaction simulation
type TXRWSet struct {
	chainCodeSet *KVRWSet
	chainCodeRW  sync.RWMutex
	balanceSet   *KVRWSet
	balanceRW    sync.RWMutex
	assetSet     *KVRWSet
	assetRW      sync.RWMutex

	block       *BLKRWSet
	currentTx   *types.Transaction
	transferTxs types.Transactions
	TxIndex     uint32
}

// GetChainCodeState get state for chaincode address and key. If committed is false, this first looks in memory
// and if missing, pulls from db.  If committed is true, this pulls from the db only.
func (tx *TXRWSet) GetChainCodeState(chaincodeAddr string, key string, committed bool) ([]byte, error) {
	tx.chainCodeRW.RLock()
	defer tx.chainCodeRW.RUnlock()
	ckey := ConstructCompositeKey(chaincodeAddr, key)
	if !committed {
		if kvw, ok := tx.chainCodeSet.Writes[ckey]; ok {
			return kvw.Value, nil
		}

		if kvr, ok := tx.chainCodeSet.Reads[ckey]; ok {
			return kvr.Value, nil
		}
	}
	return tx.block.GetChainCodeState(chaincodeAddr, key, committed)
}

// GetChainCodeStateByRange get state for chaincode address and key. If committed is false, this first looks in memory
// and if missing, pulls from db.  If committed is true, this pulls from the db only.
func (tx *TXRWSet) GetChainCodeStateByRange(chaincodeAddr string, startKey string, endKey string, committed bool) (map[string][]byte, error) {
	tx.chainCodeRW.RLock()
	defer tx.chainCodeRW.RUnlock()
	chaincodePrefix := ConstructCompositeKey(chaincodeAddr, "")
	ckeyStart := ConstructCompositeKey(chaincodeAddr, startKey)
	ckeyEnd := ConstructCompositeKey(chaincodeAddr, endKey)
	ret, err := tx.block.GetChainCodeStateByRange(chaincodeAddr, startKey, endKey, committed)
	if err != nil {
		return nil, err
	}
	if !committed {
		cache := treap.NewImmutable()
		for ckey, kvr := range tx.chainCodeSet.Reads {
			if strings.HasPrefix(ckey, chaincodePrefix) {
				cache.Put([]byte(ckey), kvr.Value)
			}
		}
		for ckey, kvw := range tx.chainCodeSet.Writes {
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
func (tx *TXRWSet) SetChainCodeState(chaincodeAddr string, key string, value []byte) error {
	tx.chainCodeRW.Lock()
	defer tx.chainCodeRW.Unlock()
	ckey := ConstructCompositeKey(chaincodeAddr, key)
	tx.chainCodeSet.Writes[ckey] = &KVWrite{
		Value:    value,
		IsDelete: false,
	}
	return nil
}

// DelChainCodeState tracks the deletion of state for chaincode address and key. Does not immediately writes to DB
func (tx *TXRWSet) DelChainCodeState(chaincodeAddr string, key string) {
	tx.chainCodeRW.Lock()
	defer tx.chainCodeRW.Unlock()
	ckey := ConstructCompositeKey(chaincodeAddr, key)
	tx.chainCodeSet.Writes[ckey] = &KVWrite{
		Value:    nil,
		IsDelete: true,
	}
}

// GetBalanceState get balance for address and assetID. If committed is false, this first looks in memory
// and if missing, pulls from db.  If committed is true, this pulls from the db only.
func (tx *TXRWSet) GetBalanceState(addr string, assetID uint32, committed bool) (*big.Int, error) {
	tx.balanceRW.RLock()
	defer tx.balanceRW.RUnlock()
	var amount big.Int
	ckey := ConstructCompositeKey(addr, strconv.FormatUint(uint64(assetID), 10)+assetIDKeySuffix)
	if !committed {
		if kvw, ok := tx.balanceSet.Writes[ckey]; ok {
			if kvw.IsDelete {
				return nil, nil
			}
			return amount.SetBytes(kvw.Value), nil
		}

		if kvr, ok := tx.balanceSet.Reads[ckey]; ok {
			return amount.SetBytes(kvr.Value), nil
		}
	}
	return tx.block.GetBalanceState(addr, assetID, committed)
}

// GetBalanceStates get balances for address. If committed is false, this first looks in memory
// and if missing, pulls from db.  If committed is true, this pulls from the db only.
func (tx *TXRWSet) GetBalanceStates(addr string, committed bool) (map[uint32]*big.Int, error) {
	tx.balanceRW.RLock()
	defer tx.balanceRW.RUnlock()
	balances, err := tx.block.GetBalanceStates(addr, committed)
	if err != nil {
		return nil, err
	}

	prefix := ConstructCompositeKey(addr, "")
	ret := make(map[string][]byte)
	if !committed {
		cache := treap.NewImmutable()
		for ckey, kvr := range tx.balanceSet.Reads {
			if strings.HasPrefix(ckey, prefix) {
				cache.Put([]byte(ckey), kvr.Value)
			}
		}
		for ckey, kvw := range tx.balanceSet.Writes {
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
func (tx *TXRWSet) SetBalacneState(addr string, assetID uint32, amount *big.Int) error {
	tx.balanceRW.Lock()
	defer tx.balanceRW.Unlock()
	ckey := ConstructCompositeKey(addr, strconv.FormatUint(uint64(assetID), 10)+assetIDKeySuffix)
	tx.balanceSet.Writes[ckey] = &KVWrite{
		Value:    amount.Bytes(),
		IsDelete: false,
	}
	return nil
}

// DelBalanceState tracks the deletion of balance for chaincode address and key. Does not immediately writes to DB
func (tx *TXRWSet) DelBalanceState(addr string, assetID uint32) {
	tx.balanceRW.Lock()
	defer tx.balanceRW.Unlock()
	ckey := ConstructCompositeKey(addr, strconv.FormatUint(uint64(assetID), 10)+assetIDKeySuffix)
	tx.balanceSet.Writes[ckey] = &KVWrite{
		Value:    nil,
		IsDelete: true,
	}
}

// GetAssetState get asset for assetID. If committed is false, this first looks in memory
// and if missing, pulls from db.  If committed is true, this pulls from the db only.
func (tx *TXRWSet) GetAssetState(assetID uint32, committed bool) (*Asset, error) {
	tx.assetRW.RLock()
	defer tx.assetRW.RUnlock()
	assetInfo := &Asset{}
	ckey := ConstructCompositeKey(assetIDKeyPrefix, strconv.FormatUint(uint64(assetID), 10)+assetIDKeySuffix)
	if !committed {
		if kvw, ok := tx.assetSet.Writes[ckey]; ok {
			if kvw.IsDelete {
				return nil, nil
			}
			if err := utils.Deserialize(kvw.Value, assetInfo); err != nil {
				return nil, err
			}
			return assetInfo, nil
		}

		if kvr, ok := tx.assetSet.Reads[ckey]; ok {
			if err := utils.Deserialize(kvr.Value, assetInfo); err != nil {
				return nil, err
			}
			return assetInfo, nil
		}
	}
	return tx.block.GetAssetState(assetID, committed)
}

// GetAssetStates get assets. If committed is false, this first looks in memory
// and if missing, pulls from db.  If committed is true, this pulls from the db only.
func (tx *TXRWSet) GetAssetStates(committed bool) (map[uint32]*Asset, error) {
	tx.assetRW.RLock()
	defer tx.assetRW.RUnlock()

	assets, err := tx.block.GetAssetStates(committed)
	if err != nil {
		return nil, err
	}
	prefix := ConstructCompositeKey(assetIDKeyPrefix, "")
	ret := make(map[string][]byte)
	if !committed {
		cache := treap.NewImmutable()
		for ckey, kvr := range tx.assetSet.Reads {
			if strings.HasPrefix(ckey, prefix) {
				cache.Put([]byte(ckey), kvr.Value)
			}
		}
		for ckey, kvw := range tx.assetSet.Writes {
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
func (tx *TXRWSet) SetAssetState(assetID uint32, assetInfo *Asset) error {
	tx.assetRW.Lock()
	defer tx.assetRW.Unlock()
	value := utils.Serialize(assetInfo)
	ckey := ConstructCompositeKey(assetIDKeyPrefix, strconv.FormatUint(uint64(assetID), 10)+assetIDKeySuffix)
	tx.assetSet.Writes[ckey] = &KVWrite{
		Value:    value,
		IsDelete: true,
	}
	return nil
}

// DelAssetState tracks the deletion of asset for assetID. Does not immediately writes to DB
func (tx *TXRWSet) DelAssetState(assetID uint32) {
	tx.assetRW.Lock()
	defer tx.assetRW.Unlock()
	ckey := ConstructCompositeKey(assetIDKeyPrefix, strconv.FormatUint(uint64(assetID), 10)+assetIDKeySuffix)
	tx.assetSet.Writes[ckey] = &KVWrite{
		Value:    nil,
		IsDelete: true,
	}
}

// ApplyChanges merges delta
func (tx *TXRWSet) ApplyChanges() error {
	tx.chainCodeRW.RLock()
	defer tx.chainCodeRW.RUnlock()
	tx.assetRW.RLock()
	defer tx.assetRW.RUnlock()
	tx.balanceRW.RLock()
	defer tx.balanceRW.RUnlock()

	err := tx.block.merge(tx.chainCodeSet, tx.assetSet, tx.balanceSet, tx.currentTx, tx.transferTxs)

	tx.assetSet = &KVRWSet{}
	tx.balanceSet = &KVRWSet{}
	tx.chainCodeSet = &KVRWSet{}
	tx.transferTxs = nil
	return err
}
