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

package blockchain

import (
	"container/list"
	"math/big"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/ledger"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
)

// NetworkStack defines the relay interface
type NetworkStack interface {
	Relay(inv types.IInventory)
}

var validTxPoolSize = 1000000

type Status struct {
	Height uint32
	Tps    int
}

// Blockchain is blockchain instance
type Blockchain struct {
	// global chain config
	// config
	mu                 sync.Mutex
	wg                 sync.WaitGroup
	currentBlockHeader *types.BlockHeader
	ledger             *ledger.Ledger
	txValidator        *Validator
	// consensus
	consenter consensus.Consenter
	// network stack
	pm NetworkStack

	quitCh       chan bool
	txCh         chan *types.Transaction
	blkCh        chan *types.Block
	heightStatus chan *Status

	orphans *list.List
	// 0 respresents sync block, 1 respresents sync done
	synced bool
}

// load loads local blockchain data
func (bc *Blockchain) load() {

	t := time.Now()
	bc.ledger.VerifyChain()
	delay := time.Since(t)

	height, err := bc.ledger.Height()

	if err != nil {
		log.Error("GetBlockHeight error", err)
		return
	}
	bc.currentBlockHeader, err = bc.ledger.GetBlockByNumber(height)

	if bc.currentBlockHeader == nil || err != nil {
		log.Errorf("GetBlockByNumber error %v ", err)
		panic(err)
	}

	log.Debugf("Load blockchain data, bestblockhash: %s height: %d load delay : %v ", bc.currentBlockHeader.Hash(), height, delay)
}

// NewBlockchain returns a fully initialised blockchain service using input data
func NewBlockchain(ledger *ledger.Ledger) *Blockchain {
	bc := &Blockchain{
		mu:                 sync.Mutex{},
		wg:                 sync.WaitGroup{},
		ledger:             ledger,
		quitCh:             make(chan bool),
		txCh:               make(chan *types.Transaction, 10000),
		blkCh:              make(chan *types.Block, 10),
		heightStatus:       make(chan *Status, 100),
		currentBlockHeader: new(types.BlockHeader),
		orphans:            list.New(),
	}
	bc.load()
	return bc
}

// SetBlockchainConsenter sets the consenter of the blockchain
func (bc *Blockchain) SetBlockchainConsenter(consenter consensus.Consenter) {
	bc.consenter = consenter
}

// SetNetworkStack sets the node of the blockchain
func (bc *Blockchain) SetNetworkStack(pm NetworkStack) {
	bc.pm = pm
}

// CurrentHeight returns current heigt of the current block
func (bc *Blockchain) CurrentHeight() uint32 {
	return bc.currentBlockHeader.Height
}

// CurrentBlockHash returns current block hash of the current block
func (bc *Blockchain) CurrentBlockHash() crypto.Hash {
	return bc.currentBlockHeader.Hash()
}

// GetNextBlockHash returns the next block hash
func (bc *Blockchain) GetNextBlockHash(h crypto.Hash) (crypto.Hash, error) {
	blockHeader, err := bc.ledger.GetBlockByHash(h.Bytes())
	if blockHeader == nil || err != nil {
		return h, err
	}
	nextBlockHeader, err := bc.ledger.GetBlockByNumber(blockHeader.Height + 1)
	if nextBlockHeader == nil || err != nil {
		return h, err
	}
	hash := nextBlockHeader.Hash()
	return hash, nil
}

// GetBalanceNonce returns balance and nonce
func (bc *Blockchain) GetBalanceNonce(addr accounts.Address) (*big.Int, uint32) {
	if bc.txValidator == nil {
		amount, nonce, _ := bc.ledger.GetBalance(addr)
		return amount, nonce + 1
	}
	return bc.txValidator.getBalanceNonce(addr)
}

// GetTransaction returns transaction in ledger first then txBool
func (bc *Blockchain) GetTransaction(txHash crypto.Hash) (*types.Transaction, error) {
	tx, err := bc.ledger.GetTxByTxHash(txHash.Bytes())
	if bc.txValidator != nil && err != nil {
		var ok bool
		if tx, ok = bc.txValidator.getTransactionByHash(txHash); ok {
			return tx, nil
		}
	}

	return tx, err
}

// Start starts blockchain services
func (bc *Blockchain) Start() {
	// bc.wg.Add(1)
	// start consesnus
	bc.StartConsensusService()
	// start txpool
	bc.StartTxPool()
	log.Debug("BlockChain Service start")
	// bc.wg.Wait()
}

func (bc *Blockchain) Synced() bool {
	return bc.synced
}

// StartConsensusService starts consensus service
func (bc *Blockchain) StartConsensusService() {
	go func() {
		for {
			select {
			case commitedTxs := <-bc.consenter.CommittedTxsChannel():
/*				if len(commitedTxs.Outputs[0].Transactions) > 0 && commitedTxs.Outputs[0].Skip != true {
					break
				}
*/
				//add lo
				log.Infof("Outputs StartConsensusService len=%d",len(commitedTxs.Outputs))
				for  x  :=range  commitedTxs.Outputs{
					log.Infof("Outputs StartConsensusService %d",len(commitedTxs.Outputs[x].Transactions))
				}
				height, _ := bc.ledger.Height()
				height++
				if commitedTxs.Height == height {
					if !bc.synced {
						bc.synced = true
					}
					bc.processConsensusOutput(commitedTxs)
				} else if commitedTxs.Height > height {
					//orphan
					for elem := bc.orphans.Front(); elem != nil; elem = elem.Next() {
						ocommitedTxs := elem.Value.(*consensus.OutputTxs)
						if ocommitedTxs.Height < height {
							bc.orphans.Remove(elem)
						} else if ocommitedTxs.Height == height {
							bc.orphans.Remove(elem)
							bc.processConsensusOutput(ocommitedTxs)
							height++
						} else {
							break
						}
					}
					if bc.orphans.Len() > 100 {
						bc.orphans.Remove(bc.orphans.Front())
					}
					bc.orphans.PushBack(commitedTxs)
				} /*else if bc.synced {
					log.Panicf("Height %d already exist in ledger", commitedTxs.Height)
				}*/
			}
		}
	}()
}

func (bc *Blockchain) processConsensusOutput(output *consensus.OutputTxs) {
	txs, time := bc.txValidator.GetCommittedTxs(output.Outputs)
	//if txs != nil && len(txs) > 0 {
	blk := bc.GenerateBlock(txs, time)
	if blk.Height() == output.Height {
		bc.pm.Relay(blk)
		//bc.ProcessBlock(blk)
	}
	//}
}

// StartTxPool starts txpool service
func (bc *Blockchain) StartTxPool() {
	bc.txValidator = NewValidator(bc.ledger)
	bc.ledger.Validator = bc.txValidator
	if params.Validator {
		bc.txValidator.startValidator()
	} else {
		bc.txValidator.stopValidator()
	}
}

// ProcessTransaction processes new transaction from the network
func (bc *Blockchain) ProcessTransaction(tx *types.Transaction) bool {
	// step 1: validate and mark transaction
	// step 2: add transaction to txPool
	// if atomic.LoadUint32(&bc.synced) == 0 {
	log.Debugf("[Blockchain] new tx, tx_hash: %v, tx_sender: %v, tx_nonce: %v", tx.Hash().String(), tx.Sender().String(), tx.Nonce())
	if bc.txValidator == nil {
		return true
	}
	if bc.txValidator.getValidatorSize() < validTxPoolSize {
		if ok := bc.txValidator.PushTxInTxPool(tx); ok {
			return true
		}
	} else {
		log.Warnf("over max txs in txpool, %d", bc.txValidator.getValidatorSize())
		return true
	}
	return false
}

// ProcessBlock processes new block from the network,flag = true pack up block ,flag = false sync block
func (bc *Blockchain) ProcessBlock(blk *types.Block, flag bool) bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	log.Debugf("block previoushash %s, currentblockhash %s", blk.PreviousHash(), bc.CurrentBlockHash())
	if blk.PreviousHash() == bc.CurrentBlockHash() {
		bc.ledger.AppendBlock(blk, flag)
		log.Infof("New Block  %s, height: %d Transaction Number: %d", blk.Hash(), blk.Height(), len(blk.Transactions))
		bc.currentBlockHeader = blk.Header
		bc.heightStatus <- &Status{Height: blk.Height(), Tps: len(blk.Transactions) / 10}
		return true
	}
	return false
}

func (bc *Blockchain) HeightStatusChan() <-chan *Status {
	return bc.heightStatus
}

func (bc *Blockchain) merkleRootHash(txs []*types.Transaction) crypto.Hash {
	if len(txs) > 0 {
		hashs := make([]crypto.Hash, 0)
		for _, tx := range txs {
			hashs = append(hashs, tx.Hash())
		}
		return crypto.ComputeMerkleHash(hashs)[0]
	}
	return crypto.Hash{}
}

// GenerateBlock gets transactions from consensus service and generates a new block
func (bc *Blockchain) GenerateBlock(txs types.Transactions, createTime uint32) *types.Block {
	var (
		// default value is empty hash
		merkleRootHash crypto.Hash
	)

	// log.Debug("Generateblock ", atomicTxs, acrossChainTxs)
	//merkleRootHash = bc.merkleRootHash(txs)

	blk := types.NewBlock(bc.currentBlockHeader.Hash(),
		createTime, bc.currentBlockHeader.Height+1,
		uint32(100),
		merkleRootHash,
		txs,
	)
	return blk
}

// StartReceiveTx starts validator tx services
func (bc *Blockchain) StartReceiveTx() {
	bc.txValidator.startValidator()
}

// StopReceiveTx stops validator tx services
func (bc *Blockchain) StopReceiveTx() {
	bc.txValidator.stopValidator()
}
