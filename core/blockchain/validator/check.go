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

package validator

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
)

func (v *Verification) isOverCapacity() bool {
	return v.txpool.Len() > v.config.TxPoolCapacity
}

func (v *Verification) isExist(tx *types.Transaction) bool {
	if _, ok := v.inTxs[tx.Hash()]; ok {
		return true
	}

	if ledgerTx, _ := v.ledger.GetTxByTxHash(tx.Hash().Bytes()); ledgerTx != nil {
		return true
	}

	return false
}

func (v *Verification) isLegalTransaction(tx *types.Transaction) bool {
	if !(strings.Compare(tx.FromChain(), params.ChainID.String()) == 0 || (strings.Compare(tx.ToChain(), params.ChainID.String()) == 0)) {
		log.Errorf("[validator] illegal transaction %s : fromCahin %s or toChain %s == params.ChainID %s", tx.Hash(), tx.FromChain(), tx.ToChain(), params.ChainID.String())
		return false
	}

	isOK := true
	switch tx.GetType() {
	case types.TypeAtomic:
		//TODO fromChain==toChain
		if strings.Compare(tx.FromChain(), tx.ToChain()) != 0 {
			log.Errorf("[validator] illegal transaction %s : fromchain %s == tochain %s", tx.Hash(), tx.FromChain(), tx.ToChain())
			isOK = false
		}
	case types.TypeAcrossChain:
		//TODO the len of fromchain == the len of tochain
		if !(len(tx.FromChain()) == len(tx.ToChain()) && strings.Compare(tx.FromChain(), tx.ToChain()) != 0) {
			log.Errorf("[validator] illegal transaction %s : wrong chain floor, fromchain %s ==  tochain %s", tx.Hash(), tx.FromChain(), tx.ToChain())
			isOK = false
		}
	case types.TypeDistribut:
		//TODO |fromChain - toChain| = 1 and sender_addr == receive_addr
		address := tx.Sender()
		fromChain := coordinate.HexToChainCoordinate(tx.FromChain())
		toChainParent := coordinate.HexToChainCoordinate(tx.ToChain()).ParentCoorinate()
		if !bytes.Equal(fromChain, toChainParent) || strings.Compare(address.String(), tx.Recipient().String()) != 0 {
			log.Errorf("[validator] illegal transaction %s :wrong chain floor, fromChain %s - toChain %s = 1", tx.Hash(), tx.FromChain(), tx.ToChain())
			isOK = false
		}
	case types.TypeBackfront:
		address := tx.Sender()
		fromChainParent := coordinate.HexToChainCoordinate(tx.FromChain()).ParentCoorinate()
		toChain := coordinate.HexToChainCoordinate(tx.ToChain())
		if !bytes.Equal(fromChainParent, toChain) || strings.Compare(address.String(), tx.Recipient().String()) != 0 {
			log.Errorf("[validator] illegal transaction %s :wrong chain floor, fromChain %s - toChain %s = 1", tx.Hash(), tx.FromChain(), tx.ToChain())
			isOK = false
		}
	case types.TypeMerged:
	//TODO nothing to do
	case types.TypeIssue, types.TypeIssueUpdate:
		//TODO the first floor and meet issue account
		fromChain := coordinate.HexToChainCoordinate(tx.FromChain())
		toChain := coordinate.HexToChainCoordinate(tx.FromChain())

		// && strings.Compare(fromChain.String(), "00") == 0)
		if len(fromChain) != len(toChain) {
			log.Errorf("[validator] illegal transaction %s: should issue chain floor, fromChain %s or toChain %s", tx.Hash(), tx.FromChain(), tx.ToChain())
			isOK = false
		}

		if !v.isIssueTransaction(tx) {
			log.Errorf("[validator] illegal transaction %s: valid issue tx public key fail", tx.Hash())
			isOK = false
		}

		if len(tx.Payload) > 0 {
			if tp := tx.GetType(); tp == types.TypeIssue {
				asset := &state.Asset{
					ID:     tx.AssetID(),
					Issuer: tx.Sender(),
					Owner:  tx.Recipient(),
				}
				if _, err := asset.Update(string(tx.Payload)); err != nil {
					log.Errorf("[validator] illegal transaction %s: invalid issue coin(%s)", tx.Hash(), string(tx.Payload))
					isOK = false
				}
			} else if tp == types.TypeIssueUpdate {
				asset := &state.Asset{
					ID:     tx.AssetID(),
					Issuer: tx.Sender(),
					Owner:  tx.Recipient(),
				}
				if _, err := asset.Update(string(tx.Payload)); err != nil {
					asset := &state.Asset{
						ID:     tx.AssetID(),
						Issuer: tx.Recipient(),
						Owner:  tx.Sender(),
					}
					if _, err := asset.Update(string(tx.Payload)); err != nil {
						log.Errorf("[validator] illegal transaction %s: invalid issue coin(%s)", tx.Hash(), string(tx.Payload))
						isOK = false
					}
				}
			}
		}
	}

	if !isOK {
		v.notify(tx, " illegal transaction")
	}

	return isOK
}

func (v *Verification) isIssueTransaction(tx *types.Transaction) bool {
	address := tx.Sender()
	addressHex := utils.BytesToHex(address.Bytes())
	for _, addr := range params.PublicAddress {
		if strings.Compare(addressHex, addr) == 0 {
			return true
		}
	}
	return false
}

func (v *Verification) checkTransaction(tx *types.Transaction) bool {
	if !v.isLegalTransaction(tx) {
		return false
	}

	address, err := tx.Verfiy()
	if err != nil || !bytes.Equal(address.Bytes(), tx.Sender().Bytes()) {
		log.Debugf("[validator] illegal transaction %s: invalid signature", tx.Hash())
		return false
	}

	v.rwInTxs.Lock()
	if v.isExist(tx) {
		v.rwInTxs.Unlock()
		return false
	}

	if v.isOverCapacity() {
		elem := v.txpool.RemoveFront()
		delete(v.inTxs, elem.(*types.Transaction).Hash())
		log.Warnf("[validator]  excess capacity, remove front transaction")
	}

	return true
}

func (v *Verification) checkTransactionInConsensus(tx *types.Transaction) bool {
	if !v.isLegalTransaction(tx) {
		return false
	}

	address, err := tx.Verfiy()
	if err != nil || !bytes.Equal(address.Bytes(), tx.Sender().Bytes()) {
		log.Debugf("[validator] illegal transaction %s: invalid signature", tx.Hash())
		return false
	}

	return true
}

// SecurityPluginDir returns the directory of security plugin.
func (v *Verification) SecurityPluginDir() string {
	return v.config.SecurityPluginDir
}

// SecurityVerifier is the security contract verifier.
type SecurityVerifier func(*types.Transaction, func(key string) ([]byte, error)) error

// SecurityVerifierManager managers the security contract verifier.
type SecurityVerifierManager struct {
	sync.Mutex
	securityPath string
	verifier     SecurityVerifier
}

var securityVerifierMnger SecurityVerifierManager

func (v *Verification) getSecurityVerifier() SecurityVerifier {
	securityVerifierMnger.Lock()
	defer securityVerifierMnger.Unlock()

	securityPathData, err := v.sctx.GetContractStateData(params.GlobalStateKey, params.SecurityContractKey)
	if err != nil {
		log.Errorf("get security contract path failed, %v", err)
		return nil
	}

	if len(securityPathData) == 0 {
		log.Warning("there is no security contract yet")
		return nil
	}

	var securityPath string
	err = json.Unmarshal(securityPathData, &securityPath)
	if err != nil {
		log.Errorf("unmarshal security contract path failed, %v", err)
		return nil
	}

	if securityPath == securityVerifierMnger.securityPath {
		return securityVerifierMnger.verifier
	}

	security, err := plugin.Open(filepath.Join(v.SecurityPluginDir(), securityPath))
	if err != nil {
		log.Errorf("load security contract failed, %v", err)
		return nil
	}

	verifyFn, err := security.Lookup("Verify")
	if err != nil {
		log.Errorf("can't find security contract verifier, %v", err)
		return nil
	}

	verifier, ok := verifyFn.(func(*types.Transaction, func(key string) ([]byte, error)) error)
	if !ok {
		log.Error("invalid security contract verifier format")
		return nil
	}

	securityVerifierMnger.verifier = SecurityVerifier(verifier)
	securityVerifierMnger.securityPath = securityPath
	return securityVerifierMnger.verifier
}

func (v *Verification) checkTransactionSecurity(tx *types.Transaction) bool {
	verifier := v.getSecurityVerifier()
	if verifier == nil {
		return true
	}

	err := verifier(tx, func(key string) ([]byte, error) {
		data, err := v.sctx.GetContractStateData(params.GlobalStateKey, key)
		if err != nil {
			return nil, err
		}
		return data, nil
	})
	if err != nil {
		log.Error(err)
		return false
	}

	return true
}
