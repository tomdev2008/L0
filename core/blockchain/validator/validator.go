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
	"fmt"
	"sync"
	"time"

	"github.com/bocheninc/L0/core/notify"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils/sortedlinkedlist"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/ledger"
	"github.com/bocheninc/L0/core/types"
)

type Validator interface {
	Start()
	ProcessTransaction(tx *types.Transaction) error
	VerifyTxs(txs types.Transactions) (types.Transactions, types.Transactions)
	RemoveTxsInVerification(txs types.Transactions)
	GetTransactionByHash(txHash crypto.Hash) (*types.Transaction, bool)
	SecurityPluginDir() string
}

type Verification struct {
	config             *Config
	ledger             *ledger.Ledger
	consenter          consensus.Consenter
	txpool             *sortedlinkedlist.SortedLinkedList
	requestBatchSignal chan int
	requestBatchTimer  *time.Timer
	blacklist          map[string]time.Time
	rwBlacklist        sync.RWMutex
	inTxs              map[crypto.Hash]*types.Transaction
	rwInTxs            sync.RWMutex
	sync.RWMutex
	//static map[crypto.Hash]time.Duration
}

var vf *Verification

func NewVerification(config *Config, ledger *ledger.Ledger, consenter consensus.Consenter) *Verification {
	vf = &Verification{
		config:             config,
		ledger:             ledger,
		consenter:          consenter,
		txpool:             sortedlinkedlist.NewSortedLinkedList(),
		requestBatchSignal: make(chan int),
		requestBatchTimer:  time.NewTimer(consenter.BatchTimeout()),
		blacklist:          make(map[string]time.Time),
		inTxs:              make(map[crypto.Hash]*types.Transaction),
	}

	return vf
}

func (v *Verification) Start() {
	log.Info("validator start ...")
	go v.processLoop()
}

func (v *Verification) makeRequestBatch() types.Transactions {
	var requestBatch types.Transactions
	var to string
	v.requestBatchTimer.Reset(v.consenter.BatchTimeout())
	v.txpool.IterElement(func(element sortedlinkedlist.IElement) bool {
		tx := element.(*types.Transaction)
		if to == "" {
			to = tx.ToChain()
		}
		if tx.ToChain() == to && len(requestBatch) <= v.consenter.BatchSize() {
			requestBatch = append(requestBatch, tx)
		} else {
			return true
		}
		return false
	})

	return requestBatch
}

func (v *Verification) processLoop() {
	ticker := time.NewTicker(v.config.BlacklistDur)
	for {
		select {
		case <-ticker.C:
			v.rwBlacklist.Lock()
			for address, created := range v.blacklist {
				if created.Add(v.config.BlacklistDur).Before(time.Now()) {
					delete(v.blacklist, address)
				}
			}
			v.rwBlacklist.Unlock()
		case cnt := <-v.requestBatchSignal:
			if cnt >= (v.config.TxPoolDelay + v.consenter.BatchSize()) {
				requestBatch := v.makeRequestBatch()
				log.Debugf("request Batch: %d ", len(requestBatch))
				v.consenter.ProcessBatch(requestBatch, v.consensusFailed)
			}
		case <-v.requestBatchTimer.C:
			if requestBatch := v.makeRequestBatch(); len(requestBatch) != 0 {
				log.Debugf("request Batch Timeout: %d ", len(requestBatch))
				v.consenter.ProcessBatch(requestBatch, v.consensusFailed)
			}
		}
	}
}

func (v *Verification) ProcessTransaction(tx *types.Transaction) error {
	startTime := time.Now()
	if err := v.checkTransaction(tx); err != nil {
		return err
	}

	v.rwInTxs.Lock()
	if v.isExist(tx) {
		v.rwInTxs.Unlock()
		return fmt.Errorf("transaction %s already existed", tx.Hash())
	}

	if v.isOverCapacity() {
		elem := v.txpool.RemoveFront()
		delete(v.inTxs, elem.(*types.Transaction).Hash())
		log.Warnf("[validator]  excess capacity, remove front transaction")
	}

	v.txpool.Add(tx)
	v.inTxs[tx.Hash()] = tx
	cnt := v.txpool.Len()
	v.rwInTxs.Unlock()
	if cnt == 1 {
		v.requestBatchTimer.Reset(v.consenter.BatchTimeout())
	}
	v.requestBatchSignal <- cnt
	log.Debugf("ProcessTransaction, tx_hash: %+v time: %s", tx.Hash(), time.Now().Sub(startTime))
	log.Debugf("[txPool] add transaction success, tx_hash: %s, sender: %s, receiver: %s, assetID %d, amount: %s,txpool_len: %d",
		tx.Hash().String(), tx.Sender(), tx.Recipient(), tx.AssetID(), tx.Amount().String(), cnt)
	return nil
}

func (v *Verification) consensusFailed(flag int, txs types.Transactions) {
	if len(txs) == 0 {
		return
	}
	switch flag {
	case 0: // not used, do nothing
		log.Debug("[validator] not primary replica ...")
		log.Debugf("process batch size: %d", len(txs))
		for _, tx := range txs {
			log.Debugf("process tx_hash %s", tx.Hash().String())
		}
	case 1: //used, do nothing
		log.Debug("[validator] primary replica ...")
	case 2: // add
		v.rwInTxs.Lock()
		defer v.rwInTxs.Unlock()
		for _, tx := range txs {
			v.txpool.Add(tx)
		}
	case 3: // remove
		v.rwInTxs.Lock()
		defer v.rwInTxs.Unlock()
		var elems []sortedlinkedlist.IElement
		//d:=time.Now().Sub(time.Unix(0,0))
		for _, tx := range txs {
			//v.static[tx.Hash()] =  d- v.static[tx.Hash()]
			elems = append(elems, tx)
		}
		v.txpool.Removes(elems)
	case 4: //update account

	case 5: // rollback account

	case 6: // remove when verify error
		v.rwInTxs.Lock()
		defer v.rwInTxs.Unlock()
		var elems []sortedlinkedlist.IElement
		for _, tx := range txs {
			notify.TxNotify(tx, fmt.Errorf("failed to verify"))
			delete(v.inTxs, tx.Hash())
			elems = append(elems, tx)
		}
		v.txpool.Removes(elems)
	default:
		log.Error("[validator] not support this flag ...")
	}
}

// VerifyTxs consensus verify txs
func (v *Verification) VerifyTxs(txs types.Transactions) (ttxs types.Transactions, etxs types.Transactions) {
	if len(txs) == 0 || !v.config.IsValid {
		return txs, etxs
	}
	v.rwInTxs.Lock()
	defer v.rwInTxs.Unlock()
	for _, tx := range txs {
		if !v.isExist(tx) {
			if err := v.checkTransaction(tx); err != nil {
				etxs = append(etxs, tx)
				log.Errorf("[validator] tx_hash: %s illegal, err %s", tx.Hash().String(), err)
				continue
			}
		}

		// assetID := tx.AssetID()
		// asset, _ := v.ledger.GetAsset(assetID)
		// if tx.GetType() != types.TypeIssue {
		// 	if asset == nil {
		// 		etxs = append(etxs, tx)
		// 		log.Errorf("[validator] tx_hash: %s, asset %d not exist", tx.Hash().String(), tx.AssetID())
		// 		continue
		// 	}
		// 	if tx.GetType() == types.TypeIssueUpdate && len(tx.Payload) > 0 {
		// 		_, err := asset.Update(string(tx.Payload))
		// 		if err != nil {
		// 			etxs = append(etxs, tx)
		// 			log.Errorf("[validator] tx_hash: %s, update asset %d(%s) --- %s", tx.Hash().String(), assetID, string(tx.Payload), err)
		// 			continue
		// 		}
		// 	}
		// } else {
		// 	if asset == nil {
		// 		asset := &state.Asset{
		// 			ID:     assetID,
		// 			Issuer: tx.Sender(),
		// 			Owner:  tx.Recipient(),
		// 		}
		// 		_, err := asset.Update(string(tx.Payload))
		// 		if err != nil {
		// 			etxs = append(etxs, tx)
		// 			log.Errorf("[validator] tx_hash: %s, new issue asset %d(%s) --- %s", tx.Hash().String(), assetID, string(tx.Payload), err)
		// 			continue
		// 		}
		// 	} else {
		// 		etxs = append(etxs, tx)
		// 		log.Errorf("[validator] tx_hash: %s, new issue asset %d(%s) --- already exist", tx.Hash().String(), assetID, string(tx.Payload))
		// 		continue
		// 	}
		// }

		ttxs = append(ttxs, tx)
	}

	return ttxs, etxs
}

func (v *Verification) RemoveTxsInVerification(txs types.Transactions) {
	v.rwInTxs.Lock()
	defer v.rwInTxs.Unlock()
	for _, tx := range txs {
		log.Debugf("[validator] remove transaction in verification ,tx_hash: %s ,txpool_len: %d", tx.Hash(), v.txpool.Len())
		delete(v.inTxs, tx.Hash())
		v.txpool.Remove(tx)
	}
}

func (v *Verification) GetTransactionByHash(txHash crypto.Hash) (*types.Transaction, bool) {
	if elem := v.txpool.GetIElementByKey(txHash.String()); elem != nil {
		return elem.(*types.Transaction), true
	}
	return nil, false
}

func GetTxPoolTransactions() []string {
	iter := vf.txpool.Iter()
	txs := []string{}
	for ele := iter(); ele != nil; ele = iter() {
		tx := ele.(*types.Transaction)
		txs = append(txs, tx.Hash().String())
	}

	return txs
}

func GetTxPoolTransacton(txHash crypto.Hash) bool {
	_, ok := vf.GetTransactionByHash(txHash)
	return ok
}
