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
	"fmt"
	"strings"
	"time"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
)

func (v *Verification) checkTransactionBeforeAddTxPool(tx *types.Transaction) bool {
	if v.checkTransactionIsExist(tx) {
		log.Errorf("[txPool] add tx fail,transaction already in txpool or consensus or ledger, tx_hash: %s", tx.Hash().String())
		return false
	}

	//refuse wrongful transaction
	if err := v.checkTransactionIsIllegal(tx); err != nil {
		log.Errorf("[txPool] add tx fail, %v", err)
		return false
	}

	return true
}

func (v *Verification) checkTransactionIsExist(tx *types.Transaction) bool {
	v.Lock()
	defer v.Unlock()

	if _, ok := v.inConsensusTxs[tx.Hash()]; ok {
		return true
	}

	if v.txpool.IsExist(tx) {
		return true
	}

	if ledgerTx, _ := v.ledger.GetTxByTxHash(tx.Hash().Bytes()); ledgerTx != nil {
		return true
	}

	return false
}

func (v *Verification) checkTxPoolCapacity() bool {
	if v.txpool.Len() > v.config.TxPoolCapacity {
		return true
	}
	return false
}

func (v *Verification) checkTransactionIsIllegal(tx *types.Transaction) error {
	v.Lock()
	defer v.Unlock()
	//refuse other chain transaction
	if !(strings.Compare(tx.FromChain(), params.ChainID.String()) == 0 || (strings.Compare(tx.ToChain(), params.ChainID.String()) == 0)) {
		return fmt.Errorf(" refuse other chain transaction, fromCahin %s or toChain %s == params.ChainID %s",
			tx.FromChain(), tx.ToChain(), params.ChainID.String())
	}

	//refuse timeout transaction
	txCreated := time.Unix(int64(tx.CreateTime()), 0)
	if txCreated.Add(v.config.TxPoolTimeOut).Before(time.Now()) {
		return fmt.Errorf(" refuse timeout transaction, tx_hash: %s, tx_create: %s",
			tx.Hash().String(), txCreated.Format("2006-01-02 15:04:05"))
	}

	//refuse blacklist transaction
	if _, ok := v.blacklist[tx.Sender().String()]; ok {
		return fmt.Errorf(" refuse blacklist transaction, tx_hash: %s, tx_sender: %s",
			tx.Hash().String(), tx.Sender().String())
	}

	//refuse wrongful transaction
	switch tx.GetType() {
	case types.TypeAtomic:
		if strings.Compare(tx.FromChain(), tx.ToChain()) != 0 {
			return fmt.Errorf(" [should fromchain == tochain], Tx_hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
		}
	case types.TypeAcrossChain:
		if !(len(tx.FromChain()) == len(tx.ToChain()) && strings.Compare(tx.FromChain(), tx.ToChain()) != 0) {
			return fmt.Errorf(" [should(chain same floor, and different)], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
		}
	case types.TypeDistribut:
		address := tx.Sender()
		fromChain := coordinate.HexToChainCoordinate(tx.FromChain())
		toChainParent := coordinate.HexToChainCoordinate(tx.ToChain()).ParentCoorinate()
		if !bytes.Equal(fromChain, toChainParent) || strings.Compare(address.String(), tx.Recipient().String()) != 0 {
			return fmt.Errorf(" [should(|fromChain - toChain| = 1 and sender_addr == receive_addr)], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
		}
	case types.TypeBackfront:
		address := tx.Sender()
		fromChainParent := coordinate.HexToChainCoordinate(tx.FromChain()).ParentCoorinate()
		toChain := coordinate.HexToChainCoordinate(tx.ToChain())
		if !bytes.Equal(fromChainParent, toChain) || strings.Compare(address.String(), tx.Recipient().String()) != 0 {
			return fmt.Errorf(" [should(|fromChain - toChain| = 1 and sender_addr == receive_addr)], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
		}
	case types.TypeIssue:
		fromChain := coordinate.HexToChainCoordinate(tx.FromChain())
		toChain := coordinate.HexToChainCoordinate(tx.FromChain())
		//TODO && strings.Compare(fromChain.String(), "00") == 0)
		if len(fromChain) != len(toChain) {
			return fmt.Errorf(" [should(the first floor)], Tx-hash: %v, tx_type: %v, tx_fchain: %v, tx_tchain: %v",
				tx.Hash().String(), tx.GetType(), tx.FromChain(), tx.ToChain())
		}
		if ok := v.checkIssueTransaction(tx); !ok {
			return fmt.Errorf(" valid issue tx public key fail, tx: %v", tx.Hash().String())
		}
	case types.TypeMerged:
		//TODO
	case types.TypeJSContractInit:
		//TODO
	case types.TypeLuaContractInit:
		//TODO
	case types.TypeContractInvoke:
		//TODO
	case types.TypeContractQuery:
		return fmt.Errorf(" this transaction of type: %v is not put in tx pool", tx.GetType())
	default:
		return fmt.Errorf(" not support this transaction of type: %v", tx.GetType())
	}

	//refuse verfiy failed transaction
	address, err := tx.Verfiy()
	if err != nil || !bytes.Equal(address.Bytes(), tx.Sender().Bytes()) {
		v.blacklist[address.String()] = time.Now()
		return fmt.Errorf(" varify fail, tx_hash: %s, tx_address: %s, tx_sender: %s",
			tx.Hash().String(), address.String(), tx.Sender().String())
	}

	return nil
}

func (v *Verification) checkIssueTransaction(tx *types.Transaction) bool {
	address := tx.Sender()
	addressHex := utils.BytesToHex(address.Bytes())
	for _, addr := range params.PublicAddress {
		if strings.Compare(addressHex, addr) == 0 {
			return true
		}
	}
	return false
}
