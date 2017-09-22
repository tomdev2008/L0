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
	"math/big"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/log"

	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/ledger"
	"github.com/bocheninc/L0/core/types"
)

type Validator struct {
	config    *Config
	ledger    *ledger.Ledger
	txpool    *txPool
	consenter consensus.Consenter
	accounts  map[string]*account
	sync.Mutex
}

func NewValidator(ledger *ledger.Ledger, config *Config, consenter consensus.Consenter) *Validator {
	validator := &Validator{
		config:    config,
		ledger:    ledger,
		consenter: consenter,
		txpool:    newTxPool(config.TxPoolTimeOut, config.BlacklistDur, config.TxPoolCapacity),
		accounts:  make(map[string]*account),
	}
	return validator
}

func (vr *Validator) Start() {
	log.Info("validator start ...")
	for {
		requestBatch := vr.txpool.getTxs(vr.consenter.BatchSize(), vr.consenter.BatchTimeout())
		log.Debugln("validator get request batch len: ", len(requestBatch))
		if len(requestBatch) != 0 {
			vr.consenter.ProcessBatch(requestBatch, vr.consensusFailed)
		} else {
			log.Debugln("txPool have no transaction ...")
			time.Sleep(5 * time.Second)
		}
	}
}

func (vr *Validator) consensusFailed(flag int, txs types.Transactions) {
	switch flag {
	// not use verify
	case 0:
		vr.txpool.backCursor(len(txs))
		log.Debug("[validator] not use verify")
	// use verify
	case 1:
		log.Debug("[validator] use verify")
	// consensus succeed
	case 2:
	//nothing todo

	// consensus failed
	case 3:
		for _, tx := range txs {
			vr.RollBackAccount(tx)
		}
		vr.txpool.backCursor(len(txs))
	default:
		log.Error("[validator] not support this flag ...")
	}
}

func (vr *Validator) VerifyTransactions(txs types.Transactions) bool {
	vr.Lock()
	defer vr.Unlock()
	for k, tx := range txs {
		if !vr.txpool.txIsExist(tx.Hash()) {
			if err := vr.txpool.checkTransaction(tx); err != nil {
				log.Errorf("[validator] verify transaction tx fail, %v", err)
				vr.txpool.removeTx(tx)
				for _, rollbackTx := range txs[:k] {
					vr.RollBackAccount(rollbackTx)
				}
				return false
			}
		}
		// remove balance is negative tx
		if !vr.UpdateAccount(tx) {
			log.Errorf("[validator] balance is negative ,tx_hash: %s", tx.Hash().String())
			vr.txpool.removeTx(tx)
			for _, rollbackTx := range txs[:k] {
				vr.RollBackAccount(rollbackTx)
			}
			return false
		}
		//remove replay transaction
		if ledgerTx, _ := vr.ledger.GetTxByTxHash(tx.Hash().Bytes()); ledgerTx != nil {
			log.Errorf("[validator] transaction is all ready in ledger ,tx_hash: %s", tx.Hash().String())
			vr.txpool.removeTx(tx)
			for _, rollbackTx := range txs[:k] {
				vr.RollBackAccount(rollbackTx)
			}
			return false
		}
	}
	return true
}

func (vr *Validator) UpdateAccount(tx *types.Transaction) bool {
	sender := vr.fetchAccount(tx.Sender())
	if sender != nil {
		sender.updateTransactionSenderBalance(tx)
		if sender.balance.Sign() == -1 {
			return false
		}
	}
	receiver := vr.fetchAccount(tx.Recipient())
	if receiver != nil {
		receiver.updateTransactionReceiverBalance(tx)
		if sender.balance.Sign() == -1 {
			return false
		}
	}
	return true
}

//RollBackAccount roll back account balance
func (vr *Validator) RollBackAccount(tx *types.Transaction) {
	senderAccont := vr.fetchAccount(tx.Sender())
	if senderAccont != nil {
		senderAccont.balance.Add(senderAccont.balance, tx.Amount())
	}
	receiverAccount := vr.fetchAccount(tx.Recipient())
	if receiverAccount != nil {
		receiverAccount.balance.Sub(receiverAccount.balance, tx.Amount())
	}
}

func (vr *Validator) fetchAccount(address accounts.Address) *account {
	vr.Lock()
	defer vr.Unlock()
	account, ok := vr.accounts[address.String()]
	if !ok {
		vr.accounts[address.String()] = newAccount(address, vr.ledger)
		account = vr.accounts[address.String()]
	}
	return account
}

func (vr *Validator) PushTxInTxPool(tx *types.Transaction) bool {
	return vr.txpool.pushTxInTxPool(tx)
}

func (vr *Validator) RemoveTxsInTxPool(txs types.Transactions) {
	vr.txpool.removeTxs(txs)
}

func (vr *Validator) GetTransactionByHash(txHash crypto.Hash) (*types.Transaction, bool) {
	if tx := vr.txpool.getTxByHash(txHash); tx != nil {
		return tx, true
	}
	return nil, false
}

func (vr *Validator) GetBalance(addr accounts.Address) (*big.Int, uint32) {
	acconut := vr.fetchAccount(addr)
	return acconut.balance, acconut.nonce
}
