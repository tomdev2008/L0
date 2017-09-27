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
	"container/list"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
)

type txPool struct {
	config           *Config
	txslist          *list.List
	cursor           *list.Element
	consenter        consensus.Consenter
	requestBathTimer *time.Timer
	mapping          map[crypto.Hash]*list.Element
	blacklist        map[string]*time.Ticker
	txChan           chan *types.Transaction
	requestBatchChan chan types.Transactions
	sync.RWMutex
}

func newTxPool(config *Config) *txPool {
	txslist := list.New()
	txslist.PushFront(&types.Transaction{})
	return &txPool{
		config:           config,
		txslist:          txslist,
		cursor:           txslist.Front(),
		mapping:          make(map[crypto.Hash]*list.Element),
		blacklist:        make(map[string]*time.Ticker),
		txChan:           make(chan *types.Transaction, 1),
		requestBatchChan: make(chan types.Transactions, 100),
	}
}

func (tp *txPool) pushTxInTxPool(tx *types.Transaction) bool {
	tp.Lock()
	defer tp.Unlock()
	//excess capacity
	if tp.txslist.Len() > tp.config.TxPoolCapacity {
		removedTx := tp.txslist.Remove(tp.txslist.Front().Next()).(*types.Transaction)
		hash := removedTx.Hash()
		delete(tp.mapping, hash)
		log.Warnf("[txPool]  excess capacity, remove front transaction, tx_hash: %s", hash.String())
	}

	//refuse wrongful transaction
	if err := tp.checkTransaction(tx); err != nil {
		log.Errorf("[txPool] add tx fail, %v", err)
		return false
	}

	if _, ok := tp.mapping[tx.Hash()]; ok {
		log.Errorf("[txPool] add tx fail,transaction already in txpool, tx_hash: %s", tx.Hash().String())
		return false
	}

	for pre := tp.txslist.Back(); pre != nil; pre = pre.Prev() {
		otx := pre.Value.(*types.Transaction)
		if tx.CreateTime() >= otx.CreateTime() {
			element := tp.txslist.InsertAfter(tx, pre)
			tp.mapping[tx.Hash()] = element
			break
		}
	}

	if tp.txslist.Len() > tp.config.TxsListDelay && tp.checkCursor() {
		tp.txChan <- tp.getTx()
	}

	log.Debugf("[txPool] add transaction success, tx_hash: %s,txpool_len: %d", tx.Hash().String(), tp.txslist.Len())

	return true
}

func (tp *txPool) pushTxsInTxPool(txs types.Transactions) {
	for _, v := range txs {
		tp.pushTxInTxPool(v)
	}
}

func (tp *txPool) checkTransaction(tx *types.Transaction) error {

	//refuse other chain transaction
	if !(strings.Compare(tx.FromChain(), params.ChainID.String()) == 0 || (strings.Compare(tx.ToChain(), params.ChainID.String()) == 0)) {
		return fmt.Errorf(" refuse other chain transaction, fromCahin %s or toChain %s == params.ChainID %s",
			tx.FromChain(), tx.ToChain(), params.ChainID.String())
	}

	//refuse timeout transaction
	txCreated := time.Unix(int64(tx.CreateTime()), 0)
	if txCreated.Add(tp.config.TxPoolTimeOut).Before(time.Now()) {
		return fmt.Errorf(" refuse timeout transaction, tx_hash: %s, tx_create: %s",
			tx.Hash().String(), txCreated.Format("2006-01-02 15:04:05"))
	}

	//refuse blacklist transaction
	if _, ok := tp.blacklist[tx.Sender().String()]; ok {
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
		if ok := tp.checkIssueTransaction(tx); !ok {
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
		return fmt.Errorf(" varify fail, tx_hash: %s, tx_address: %s, tx_sender: %s",
			tx.Hash().String(), address.String(), tx.Sender().String())
	}

	return nil
}

func (tp *txPool) checkIssueTransaction(tx *types.Transaction) bool {
	address := tx.Sender()
	addressHex := utils.BytesToHex(address.Bytes())
	for _, addr := range params.PublicAddress {
		if strings.Compare(addressHex, addr) == 0 {
			return true
		}
	}
	return false
}

func (tp *txPool) addBlacklist(address string) {
	tp.Lock()
	defer tp.Unlock()
	tp.blacklist[address] = time.NewTicker(tp.config.BlacklistDur)
	go tp.releaseViolator(address)

	//print blacklist
	var addresses []string
	for k := range tp.blacklist {
		addresses = append(addresses, k)
	}
	log.Debugln("[txPool] add violator address: %s blacklist: %v", address, addresses)
}

func (tp *txPool) releaseViolator(address string) {
	tp.Lock()
	defer tp.Unlock()
	for {
		select {
		case <-tp.blacklist[address].C:
			delete(tp.blacklist, address)

			//print blacklist
			var addresses []string
			for k := range tp.blacklist {
				addresses = append(addresses, k)
			}
			log.Debugln("[txPool] release violator address: %s blacklist: %v", address, addresses)
		}
	}
}

func (tp *txPool) checkCursor() bool {
	if tp.txslist.Len() == 1 {
		return false
	}
	if _, ok := tp.mapping[tp.cursor.Value.(*types.Transaction).Hash()]; !ok {
		tp.resetCursor()
	}
	if tp.cursor.Next() == nil {
		return false
	}
	return true
}

func (tp *txPool) getTx() *types.Transaction {
	elem := tp.cursor.Next()
	tx := elem.Value.(*types.Transaction)
	tp.cursor = elem
	_, ok := tp.blacklist[tx.Sender().String()]
	if time.Unix(int64(tx.CreateTime()), 0).Add(tp.config.TxPoolTimeOut).Before(time.Now()) || ok {
		log.Errorf("[txPool] get tx fail,transaction already timeout or tx_sender in blacklist when in txpool, tx_hash: %s", tx.Hash().String())
		tp.txslist.Remove(elem)
		delete(tp.mapping, tx.Hash())
		return nil
	}
	return tx
}

func (tp *txPool) getTxs(size int) types.Transactions {
	tp.Lock()
	defer tp.Unlock()
	var requestBatch types.Transactions
	to := tp.cursor.Next().Value.(*types.Transaction).ToChain()
	for elem := tp.cursor.Next(); elem != nil; elem = elem.Next() {
		tx := elem.Value.(*types.Transaction)
		tp.cursor = elem
		_, ok := tp.blacklist[tx.Sender().String()]
		if time.Unix(int64(tx.CreateTime()), 0).Add(tp.config.TxPoolTimeOut).Before(time.Now()) || ok {
			log.Errorf("[txPool] get tx fail,transaction already timeout or tx_sender in blacklist when in txpool, tx_hash: %s", tx.Hash().String())
			tp.txslist.Remove(elem)
			delete(tp.mapping, tx.Hash())
			continue
		}
		if tx.ToChain() == to && len(requestBatch) < size {
			requestBatch = append(requestBatch, tx)
		} else {
			return requestBatch
		}
	}
	return requestBatch
}

func (tp *txPool) start() {
	log.Debugln("txpool start ...")
	var to string
	var requestBatch types.Transactions
	tp.requestBathTimer = time.NewTimer(tp.config.BatchTimeOut)
	for {
		select {
		case tx := <-tp.txChan:
			if to == "" {
				to = tx.ToChain()
			}
			if tx.ToChain() == to && len(requestBatch) < tp.config.BatchSize {
				requestBatch = append(requestBatch, tx)
			} else {
				tp.requestBatchChan <- requestBatch
				requestBatch = types.Transactions{}
				to = tx.ToChain()
				tp.requestBathTimer.Reset(tp.config.BatchTimeOut)
			}
		case <-tp.requestBathTimer.C:
			if len(requestBatch) != 0 {
				requestBatch = append(requestBatch, tp.getTxs(tp.config.BatchSize-len(requestBatch))...)
				tp.requestBatchChan <- requestBatch
				requestBatch = types.Transactions{}
			} else {
				if tp.checkCursor() {
					tp.requestBatchChan <- tp.getTxs(tp.config.BatchSize)
				}
			}
			tp.requestBathTimer.Reset(tp.config.BatchTimeOut)
		}
	}
}

func (tp *txPool) backCursor(i int) {
	tp.Lock()
	defer tp.Unlock()
	if tp.txslist.Len() == 1 {
		return
	}
	for elem := tp.cursor; elem != nil; elem = elem.Prev() {
		tp.cursor = elem
		if i == 0 {
			break
		}
		i--
	}
}

func (tp *txPool) resetCursor() {
	tp.cursor = tp.txslist.Front()
}

func (tp *txPool) getTxByHash(hash crypto.Hash) *types.Transaction {
	tp.Lock()
	defer tp.Unlock()
	if element, ok := tp.mapping[hash]; ok {
		return element.Value.(*types.Transaction)
	}
	return nil
}

func (tp *txPool) removeTxs(txs types.Transactions) {
	for _, tx := range txs {
		tp.removeTx(tx)
	}
}

func (tp *txPool) removeTx(tx *types.Transaction) {
	tp.Lock()
	defer tp.Unlock()
	if element, ok := tp.mapping[tx.Hash()]; ok {
		tp.txslist.Remove(element)
		delete(tp.mapping, tx.Hash())
		log.Debugf("remove tx in txpool,tx_hash: %s， txs_len: %d", tx.Hash(), tp.txslist.Len())
	}
}

func (tp *txPool) txIsExist(h crypto.Hash) bool {
	tp.Lock()
	defer tp.Unlock()
	_, ok := tp.mapping[h]
	return ok
}
