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
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
)

var permissionPrefix = "permission."

func (tx *TXRWSet) verifyPermission(key string) error {
	var dataAdmin []byte
	var err error
	if key == params.AdminKey || key == params.GlobalContractKey {
		dataAdmin, err = tx.GetChainCodeState(params.GlobalStateKey, params.AdminKey, false)
		if err != nil {
			return err
		}
	} else {
		var permissionKey string
		if strings.Contains(key, permissionPrefix) {
			permissionKey = key
		} else {
			permissionKey = permissionPrefix + key
		}

		dataAdmin, err = tx.GetChainCodeState(params.GlobalStateKey, permissionKey, false)
		if err != nil {
			return err
		}

		if len(dataAdmin) == 0 {
			dataAdmin, err = tx.GetChainCodeState(params.GlobalStateKey, params.AdminKey, false)
			if err != nil {
				return err
			}
		}
	}

	sender := tx.currentTx.Sender().Bytes()
	if len(dataAdmin) > 0 {
		var dataAdminAddr accounts.Address
		err = json.Unmarshal(dataAdmin, &dataAdminAddr)
		if err != nil {
			return nil
		}

		if !bytes.Equal(sender, dataAdminAddr[:]) {
			log.Errorf("change global state, permission denied, \n%#v\n%#v\n",
				sender, dataAdminAddr[:])
			return fmt.Errorf("change global state, permission denied")
		}
	}
	return nil
}

func (tx *TXRWSet) GetGlobalState(key string) ([]byte, error) {
	log.Debugf("GetGlobalState key=[%s]", key)
	return tx.GetChainCodeState(params.GlobalStateKey, key, false)
}

func (tx *TXRWSet) PutGlobalState(key string, value []byte) error {
	if err := tx.verifyPermission(key); err != nil {
		return err
	}
	log.Debugf("SetGlobalState key=[%s], value=[%#v]", key, value)
	return tx.SetChainCodeState(params.GlobalStateKey, key, value)
}

func (tx *TXRWSet) DelGlobalState(key string) error {
	if err := tx.verifyPermission(key); err != nil {
		return err
	}
	log.Debugf("DelGlobalState key=[%s]", key)
	tx.DelChainCodeState(params.GlobalStateKey, key)
	return nil
}

func (tx *TXRWSet) ComplexQuery(key string) ([]byte, error) {
	chaincodeAddr := tx.currentTx.Recipient().String()
	log.Debugf("ComplexQuery chaincode=[%s], key=[%s]", chaincodeAddr, key)
	return nil, errors.New("vp can't support complex qery")
}

func (tx *TXRWSet) GetState(key string) ([]byte, error) {
	chaincodeAddr := tx.currentTx.Recipient().String()
	log.Debugf("GetState chaincode=[%s], key=[%s]", chaincodeAddr, key)
	return tx.GetChainCodeState(chaincodeAddr, key, false)
}

func (tx *TXRWSet) PutState(key string, value []byte) error {
	chaincodeAddr := tx.currentTx.Recipient().String()
	log.Debugf("SetState chaincode=[%s], key=[%s], value=[%#v]", chaincodeAddr, key, value)
	return tx.SetChainCodeState(chaincodeAddr, key, value)
}

func (tx *TXRWSet) DelState(key string) error {
	chaincodeAddr := tx.currentTx.Recipient().String()
	log.Debugf("DelState chaincode=[%s], key=[%s]", chaincodeAddr, key)
	tx.DelChainCodeState(chaincodeAddr, key)
	return nil
}

func (tx *TXRWSet) GetByPrefix(key string) ([]*db.KeyValue, error) {
	chaincodeAddr := tx.currentTx.Recipient().String()
	log.Debugf("GetByPrefix chaincode=[%s], key=[%s]", chaincodeAddr, key)
	ret, err := tx.GetChainCodeStateByRange(chaincodeAddr, key, "", false)
	if err != nil {
		return nil, err
	}
	kvs := []*db.KeyValue{}
	for k, v := range ret {
		kvs = append(kvs, &db.KeyValue{
			[]byte(k),
			v,
		})
	}
	return kvs, nil
}

func (tx *TXRWSet) GetByRange(startKey, endKey string) ([]*db.KeyValue, error) {
	chaincodeAddr := tx.currentTx.Recipient().String()
	log.Debugf("GetByRange chaincode=[%s], startKey=[%s], endKey=[%s]", chaincodeAddr, startKey, endKey)
	ret, err := tx.GetChainCodeStateByRange(chaincodeAddr, startKey, endKey, false)
	if err != nil {
		return nil, err
	}
	kvs := []*db.KeyValue{}
	for k, v := range ret {
		kvs = append(kvs, &db.KeyValue{
			[]byte(k),
			v,
		})
	}
	return kvs, nil
}

func (tx *TXRWSet) GetBalance(addr string, assetID uint32) (*big.Int, error) {
	log.Debugf("GetBalance addr=[%s], assetID=[%d]", addr, assetID)
	return tx.GetBalanceState(addr, assetID, false)
}

func (tx *TXRWSet) GetBalances(addr string) (*Balance, error) {
	log.Debugf("GetBalances addr=[%s]", addr)
	ret, err := tx.GetBalanceStates(addr, false)
	return &Balance{ret}, err
}

func (tx *TXRWSet) GetCurrentBlockHeight() uint32 {
	log.Debugf("GetCurrentBlockHeight")
	return tx.block.BlockIndex
}

func (tx *TXRWSet) AddTransfer(fromAddr, toAddr string, assetID uint32, amount *big.Int, fee *big.Int) error {
	log.Debugf("AddTransfer from=[%s], to=[%s], assetID=[%d], amount=[%s], fee=[%s]", fromAddr, toAddr, assetID, amount, fee)
	ttx := types.NewTransaction(
		tx.currentTx.Data.FromChain,
		tx.currentTx.Data.ToChain,
		types.TypeAtomic,
		tx.currentTx.Data.Nonce,
		accounts.HexToAddress(fromAddr),
		accounts.HexToAddress(toAddr),
		assetID,
		amount,
		fee,
		tx.currentTx.Data.CreateTime,
	)
	if err := tx.Transfer(ttx); err != nil {
		return err
	}
	tx.transferTxs = append(tx.transferTxs, ttx)
	return nil
}

func (tx *TXRWSet) Transfer(ttx *types.Transaction) error {
	log.Debugf("TXRWSet Transfer")
	sender := ttx.Sender().String()
	receiver := ttx.Recipient().String()
	assetID := ttx.AssetID()
	amount := ttx.Amount()
	fee := ttx.Fee()
	tp := ttx.GetType()
	if tp == types.TypeIssue {
		if asset, err := tx.GetAssetState(assetID, false); asset != nil || err != nil {
			if err != nil {
				return fmt.Errorf("asset id %d failed to get -- %s", assetID, err)
			}
			return fmt.Errorf("asset id %d alreay exist", assetID)
		}
		asset := &Asset{
			ID:     assetID,
			Issuer: ttx.Sender(),
			Owner:  ttx.Recipient(),
		}
		asset, err := asset.Update(string(ttx.Payload))
		if err != nil {
			return fmt.Errorf("asset id %d failed to update -- %s", assetID, err)
		}
		tx.SetAssetState(assetID, asset)
	} else if tp == types.TypeIssueUpdate {
		asset, err := tx.GetAssetState(assetID, false)
		if asset == nil {
			if err != nil {
				return fmt.Errorf("asset id %d failed to get -- %s", assetID, err)
			}
			return fmt.Errorf("asset id %d not exist", assetID)
		}
		asset, err = asset.Update(string(ttx.Payload))
		if err != nil {
			return fmt.Errorf("asset id %d failed to update -- %s", assetID, err)
		}
		tx.SetAssetState(assetID, asset)
	}

	sbalance, err := tx.GetBalanceState(sender, assetID, false)
	if err != nil {
		return err
	}
	if sbalance == nil {
		sbalance = big.NewInt(0)
	}
	rbalance, err := tx.GetBalanceState(receiver, assetID, false)
	if err != nil {
		return err
	}
	if rbalance == nil {
		rbalance = big.NewInt(0)
	}

	tamount := big.NewInt(0)
	tamount.Add(amount, fee)
	sbalance.Sub(sbalance, tamount)
	if tp != types.TypeIssue && tp != types.TypeIssueUpdate {
		if sbalance.Sign() < 0 {
			return ErrNegativeBalance
		}
	}
	rbalance.Add(rbalance, tamount)
	tx.SetBalacneState(sender, assetID, sbalance)
	tx.SetBalacneState(receiver, assetID, rbalance)
	return nil
}

type CallBackResponse struct {
	IsCanRedo bool
	Err       error
	Result    interface{}
}

func (tx *TXRWSet) CallBack(res *CallBackResponse) error {
	log.Debugf("TXRWSet CallBack txIndex: %d %v", tx.TxIndex, res)
	if res.Err != nil {
		tx.assetSet = NewKVRWSet()
		tx.balanceSet = NewKVRWSet()
		tx.chainCodeSet = NewKVRWSet()
		tx.transferTxs = nil
		if res.IsCanRedo {
			return res.Err
		}
	}
	return tx.ApplyChanges()
}
