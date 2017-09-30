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
	"github.com/bocheninc/L0/components/utils/sortedlinkedlist"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/ledger"
	"github.com/bocheninc/L0/core/types"
)

type Validator interface {
	Start()
	ProcessTransaction(tx *types.Transaction) bool
	VerifyTxs(txs types.Transactions) bool
	UpdateAccount(tx *types.Transaction) bool
	RollBackAccount(tx *types.Transaction)
	RemoveTxsInVerification(txs types.Transactions)
	GetTransactionByHash(txHash crypto.Hash) (*types.Transaction, bool)
	GetBalance(addr accounts.Address) (*big.Int, uint32)
}

type Verification struct {
	config             *Config
	ledger             *ledger.Ledger
	consenter          consensus.Consenter
	txpool             *sortedlinkedlist.SortedLinkedList
	requestBatchSignal chan int
	requestBatchTimer  *time.Timer
	blacklist          map[string]time.Time
	accounts           map[string]*account
	inConsensusTxs     map[crypto.Hash]*types.Transaction
	sync.RWMutex
}

func NewVerification(ledger *ledger.Ledger,
	config *Config, consenter consensus.Consenter,
	linkedList *sortedlinkedlist.SortedLinkedList) *Verification {
	return &Verification{
		config:             config,
		ledger:             ledger,
		consenter:          consenter,
		txpool:             linkedList,
		requestBatchSignal: make(chan int),
		requestBatchTimer:  time.NewTimer(consenter.BatchTimeout()),
		blacklist:          make(map[string]time.Time),
		accounts:           make(map[string]*account),
		inConsensusTxs:     make(map[crypto.Hash]*types.Transaction),
	}
}

func (v *Verification) Start() {
	log.Info("validator start ...")
	go v.ProcessBatchLoop()
	go v.processBlacklistLoop()
}

func (v *Verification) ProcessTransaction(tx *types.Transaction) bool {
	if v.pushTxInTxPool(tx) {
		v.requestBatchSignal <- 1
		v.requestBatchTimer.Reset(v.consenter.BatchTimeout())
		return true
	}
	return false
}

func (v *Verification) pushTxInTxPool(tx *types.Transaction) bool {
	if !v.checkTransactionBeforeAddTxPool(tx) {
		return false
	}

	v.txpool.Add(tx)

	if v.checkTxPoolCapacity() {
		v.txpool.RemoveFront()
		log.Warnf("[txPool]  excess capacity, remove front transaction")
	}

	log.Debugf("[txPool] add transaction success, tx_hash: %s,txpool_len: %d", tx.Hash().String(), v.txpool.Len())
	return true
}

func (v *Verification) consensusFailed(flag int, txs types.Transactions) {
	switch flag {
	// not use verify
	case 0:
		log.Debug("[validator] not use verify...")

	// use verify
	case 1:
		v.Lock()
		log.Debug("[validator] use verify...")
		for _, tx := range txs {
			v.inConsensusTxs[tx.Hash()] = tx
			v.txpool.Remove(tx)
		}
		v.Unlock()
	// consensus failed
	case 2:
		log.Debug("[validator] consensus failed...")
		v.RemoveTxsInVerification(txs)
		for _, tx := range txs {
			//v.RollBackAccount(tx)
			v.pushTxInTxPool(tx)
		}
	// consensus succeed
	case 3:

	default:
		log.Error("[validator] not support this flag ...")
	}
}

func (v *Verification) VerifyTxs(txs types.Transactions) bool {
	if !v.config.IsValid {
		return true
	}

	for _, tx := range txs {
		if !v.checkTransactionIsExist(tx) {
			if err := v.checkTransactionIsIllegal(tx); err != nil {
				log.Errorf("[validator] verify transaction tx fail, %v", err)
				return false
			}
		}
		// remove balance is negative tx
		if !v.UpdateAccount(tx) {
			log.Errorf("[validator] balance is negative ,tx_hash: %s", tx.Hash().String())
			// for _, rollbackTx := range txs[:k] {
			// 	v.RollBackAccount(rollbackTx)
			// }
			return false
		}
	}
	return true
}

func (v *Verification) UpdateAccount(tx *types.Transaction) bool {
	senderAccont := v.fetchAccount(tx.Sender())
	if senderAccont != nil {
		senderAccont.updateTransactionSenderBalance(tx)
		//	log.Debugln("[validator] updateAccount sender: ", tx.Sender(), "amount: ", senderAccont.amount)
		if senderAccont.amount.Sign() == -1 && tx.GetType() != types.TypeIssue {
			return false
		}
	}
	receiverAccount := v.fetchAccount(tx.Recipient())
	if receiverAccount != nil {
		receiverAccount.updateTransactionReceiverBalance(tx)
		//	log.Debugln("[validator] updateAccount Recipient: ", tx.Recipient(), "amount: ", receiverAccount.amount)
		if receiverAccount.amount.Sign() == -1 {
			return false
		}
	}

	return true
}

//RollBackAccount roll back account balance
func (v *Verification) RollBackAccount(tx *types.Transaction) {
	senderAccont := v.fetchAccount(tx.Sender())
	if senderAccont != nil {
		senderAccont.rollBackTransactionSenderBalance(tx)
	}
	receiverAccount := v.fetchAccount(tx.Recipient())
	if receiverAccount != nil {
		receiverAccount.rollBackTransactionReceiverBalance(tx)
	}
}

func (v *Verification) fetchAccount(address accounts.Address) *account {
	v.Lock()
	defer v.Unlock()
	account, ok := v.accounts[address.String()]
	if !ok {
		v.accounts[address.String()] = newAccount(address, v.ledger)
		account = v.accounts[address.String()]
	}
	return account
}

func (v *Verification) RemoveTxInVerification(tx *types.Transaction) {
	v.Lock()
	defer v.Unlock()
	log.Debugln("[validator] remove transaction in verification ,tx_hash: ", tx.Hash())
	delete(v.inConsensusTxs, tx.Hash())
	v.txpool.Remove(tx)
}

func (v *Verification) RemoveTxsInVerification(txs types.Transactions) {
	for _, tx := range txs {
		v.RemoveTxInVerification(tx)
	}

}

func (v *Verification) GetTransactionByHash(txHash crypto.Hash) (*types.Transaction, bool) {
	if elem := v.txpool.GetIElementByKey(txHash.String()); elem != nil {
		return elem.(*types.Transaction), true
	}
	return nil, false
}

func (v *Verification) GetBalance(addr accounts.Address) (*big.Int, uint32) {
	acconut := v.fetchAccount(addr)
	return acconut.amount, acconut.nonce
}

func (v *Verification) processBlacklistLoop() {
	ticker := time.NewTicker(v.config.BlacklistDur)
	for {
		select {
		case <-ticker.C:
			v.Lock()
			for address, created := range v.blacklist {
				if created.Add(v.config.BlacklistDur).Before(time.Now()) {
					delete(v.blacklist, address)
				}
			}
			v.Unlock()
		}
	}
}

func (v *Verification) makeRequestBatch() types.Transactions {
	var requestBatch types.Transactions
	var to string
	v.txpool.IterElement(func(element sortedlinkedlist.IElement) bool {
		tx := element.(*types.Transaction)
		if to == "" {
			to = tx.ToChain()
		}
		if tx.ToChain() == to && len(requestBatch) < v.consenter.BatchSize() {
			requestBatch = append(requestBatch, tx)
		} else {
			return true
		}
		return false
	})

	return requestBatch

}

func (v *Verification) ProcessBatchLoop() {
	for {
		select {
		case <-v.requestBatchSignal:
			if v.txpool.Len() > (v.config.TxPoolDelay + v.consenter.BatchSize()) {
				v.consenter.ProcessBatch(v.makeRequestBatch(), v.consensusFailed)
			}
		case <-v.requestBatchTimer.C:
			if requestBatch := v.makeRequestBatch(); len(requestBatch) != 0 {
				v.consenter.ProcessBatch(requestBatch, v.consensusFailed)
			}
			v.requestBatchTimer.Reset(v.consenter.BatchTimeout())
		}
	}
}
