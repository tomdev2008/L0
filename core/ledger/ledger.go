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

package ledger

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/ledger/block_storage"
	"github.com/bocheninc/L0/core/ledger/contract"
	"github.com/bocheninc/L0/core/ledger/merge"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/vm"
)

var (
	ledgerInstance *Ledger
)

type ValidatorHandler interface {
	UpdateAccount(tx *types.Transaction) bool
	RollBackAccount(tx *types.Transaction)
	RemoveTxsInVerification(txs types.Transactions)
}

// Ledger represents the ledger in blockchain
type Ledger struct {
	dbHandler *db.BlockchainDB
	block     *block_storage.Blockchain
	state     *state.State
	storage   *merge.Storage
	contract  *contract.SmartConstract
	Validator ValidatorHandler
}

// NewLedger returns the ledger instance
func NewLedger(db *db.BlockchainDB) *Ledger {
	if ledgerInstance == nil {
		ledgerInstance = &Ledger{
			dbHandler: db,
			block:     block_storage.NewBlockchain(db),
			state:     state.NewState(db),
			storage:   merge.NewStorage(db),
		}
		_, err := ledgerInstance.Height()
		if err != nil {
			ledgerInstance.init()
		}
	}

	ledgerInstance.contract = contract.NewSmartConstract(db, ledgerInstance)
	return ledgerInstance
}

// VerifyChain verifys the blockchain data
func (ledger *Ledger) VerifyChain() {
	height, err := ledger.Height()
	if err != nil {
		panic(err)
	}
	currentBlockHeader, err := ledger.block.GetBlockByNumber(height)
	for i := height; i >= 1; i-- {
		previousBlockHeader, err := ledger.block.GetBlockByNumber(i - 1) // storage
		if previousBlockHeader != nil && err != nil {

			log.Debug("get block err")
			panic(err)
		}
		// verify previous block
		if !previousBlockHeader.Hash().Equal(currentBlockHeader.PreviousHash) {
			panic(fmt.Errorf("block [%d], veifychain breaks", i))
		}
		currentBlockHeader = previousBlockHeader
	}
}

// GetGenesisBlock returns the genesis block of the ledger
func (ledger *Ledger) GetGenesisBlock() *types.BlockHeader {

	genesisBlockHeader, err := ledger.GetBlockByNumber(0)
	if err != nil {
		panic(err)
	}
	return genesisBlockHeader
}

// AppendBlock appends a new block to the ledger,flag = true pack up block ,flag = false sync block
func (ledger *Ledger) AppendBlock(block *types.Block, flag bool) error {
	var (
		txWriteBatchs []*db.WriteBatch
		txs           types.Transactions
		errTxs        types.Transactions
	)

	bh, _ := ledger.Height()
	ledger.contract.StartConstract(bh)

	txWriteBatchs, block.Transactions, errTxs = ledger.executeTransactions(block.Transactions, flag)
	_ = errTxs

	block.Header.TxsMerkleHash = merkleRootHash(block.Transactions)
	writeBatchs := ledger.block.AppendBlock(block)
	writeBatchs = append(writeBatchs, txWriteBatchs...)
	writeBatchs = append(writeBatchs, ledger.state.WriteBatchs()...)
	if err := ledger.dbHandler.AtomicWrite(writeBatchs); err != nil {
		return err
	}

	ledger.Validator.RemoveTxsInVerification(block.Transactions)

	ledger.contract.StopContract(bh)

	for _, tx := range block.Transactions {
		if (tx.GetType() == types.TypeMerged && !ledger.checkCoordinate(tx)) || tx.GetType() == types.TypeAcrossChain {
			txs = append(txs, tx)
		}
	}
	if err := ledger.storage.ClassifiedTransaction(txs); err != nil {
		return err
	}
	log.Infoln("blockHeight: ", block.Height(), "need merge Txs len : ", len(txs), "all Txs len: ", len(block.Transactions))

	return nil
}

// GetBlockByNumber gets the block by the given number
func (ledger *Ledger) GetBlockByNumber(number uint32) (*types.BlockHeader, error) {
	return ledger.block.GetBlockByNumber(number)
}

// GetBlockByHash returns the block detail by hash
func (ledger *Ledger) GetBlockByHash(blockHashBytes []byte) (*types.BlockHeader, error) {
	return ledger.block.GetBlockByHash(blockHashBytes)
}

//GetTransactionHashList returns transactions hash list by block number
func (ledger *Ledger) GetTransactionHashList(number uint32) ([]crypto.Hash, error) {

	txHashsBytes, err := ledger.block.GetTransactionHashList(number)
	if err != nil {
		return nil, err
	}

	txHashs := []crypto.Hash{}

	utils.Deserialize(txHashsBytes, &txHashs)

	return txHashs, nil
}

// Height returns height of ledger
func (ledger *Ledger) Height() (uint32, error) {
	return ledger.block.GetBlockchainHeight()
}

//GetLastBlockHash returns last block hash
func (ledger *Ledger) GetLastBlockHash() (crypto.Hash, error) {
	height, err := ledger.block.GetBlockchainHeight()
	if err != nil {
		return crypto.Hash{}, err
	}
	lastBlock, err := ledger.block.GetBlockByNumber(height)
	if err != nil {
		return crypto.Hash{}, err
	}
	return lastBlock.Hash(), nil
}

//GetBlockHashByNumber returns block hash by block number
func (ledger *Ledger) GetBlockHashByNumber(blockNum uint32) (crypto.Hash, error) {

	hashBytes, err := ledger.block.GetBlockHashByNumber(blockNum)
	if err != nil {
		return crypto.Hash{}, err
	}

	blockHash := new(crypto.Hash)

	blockHash.SetBytes(hashBytes)

	return *blockHash, err
}

// GetTxsByBlockHash returns transactions  by block hash and transactionType
func (ledger *Ledger) GetTxsByBlockHash(blockHashBytes []byte, transactionType uint32) (types.Transactions, error) {
	return ledger.block.GetTransactionsByHash(blockHashBytes, transactionType)
}

//GetTxsByBlockNumber returns transactions by blcokNumber and transactionType
func (ledger *Ledger) GetTxsByBlockNumber(blockNumber uint32, transactionType uint32) (types.Transactions, error) {
	return ledger.block.GetTransactionsByNumber(blockNumber, transactionType)
}

//GetTxByTxHash returns transaction by tx hash []byte
func (ledger *Ledger) GetTxByTxHash(txHashBytes []byte) (*types.Transaction, error) {
	return ledger.block.GetTransactionByTxHash(txHashBytes)
}

// GetBalanceFromDB returns balance by account
func (ledger *Ledger) GetBalanceFromDB(addr accounts.Address) (*state.Balance, error) {
	return ledger.state.GetBalance(addr)
}

// GetAssetFromDB returns asset
func (ledger *Ledger) GetAssetFromDB(id uint32) (*state.Asset, error) {
	return ledger.state.GetAsset(id)
}

//GetMergedTransaction returns merged transaction within a specified period of time
func (ledger *Ledger) GetMergedTransaction(duration uint32) (types.Transactions, error) {

	txs, err := ledger.storage.GetMergedTransaction(duration)
	if err != nil {
		return nil, err
	}
	return txs, nil
}

//PutTxsHashByMergeTxHash put transactions hashs by merge transaction hash
func (ledger *Ledger) PutTxsHashByMergeTxHash(mergeTxHash crypto.Hash, txsHashs []crypto.Hash) error {
	return ledger.storage.PutTxsHashByMergeTxHash(mergeTxHash, txsHashs)
}

//GetTxsByMergeTxHash gets transactions
func (ledger *Ledger) GetTxsByMergeTxHash(mergeTxHash crypto.Hash) (types.Transactions, error) {
	txsHashs, err := ledger.storage.GetTxsByMergeTxHash(mergeTxHash)
	if err != nil {
		return nil, err
	}

	txs := types.Transactions{}
	for _, v := range txsHashs {
		tx, err := ledger.GetTxByTxHash(v.Bytes())
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

//QueryContract processes new contract query transaction
func (ledger *Ledger) QueryContract(tx *types.Transaction) ([]byte, error) {
	contractSpec := new(types.ContractSpec)
	utils.Deserialize(tx.Payload, contractSpec)
	ledger.contract.ExecTransaction(tx, string(contractSpec.ContractAddr))

	result, err := vm.Query(tx, contractSpec, ledger.contract)
	if err != nil {
		log.Error("contract query execute failed  ", err)
		return nil, fmt.Errorf("contract query execute failed : %v ", err)
	}
	return result, nil
}

// init generates the genesis block
func (ledger *Ledger) init() error {
	// genesis block
	blockHeader := new(types.BlockHeader)
	blockHeader.TimeStamp = uint32(0)
	blockHeader.Nonce = uint32(100)
	blockHeader.Height = 0

	genesisBlock := new(types.Block)
	genesisBlock.Header = blockHeader
	writeBatchs := ledger.block.AppendBlock(genesisBlock)
	writeBatchs = append(writeBatchs, ledger.state.WriteBatchs()...)

	// admin address
	writeBatchs = append(writeBatchs,
		db.NewWriteBatch(contract.ColumnFamily,
			db.OperationPut,
			[]byte(contract.AdminKey),
			contract.DefaultAdminAddr.Bytes()))

	return ledger.dbHandler.AtomicWrite(writeBatchs)
}

func (ledger *Ledger) executeTransactions(txs types.Transactions, flag bool) ([]*db.WriteBatch, types.Transactions, types.Transactions) {
	var (
		err                error
		errTxs             types.Transactions
		syncTxs            types.Transactions
		syncContractGenTxs types.Transactions
		writeBatchs        []*db.WriteBatch
	)

	for _, tx := range txs {
		switch tp := tx.GetType(); tp {
		case types.TypeJSContractInit, types.TypeLuaContractInit, types.TypeContractInvoke:
			if err := ledger.executeTransaction(tx, false); err != nil {
				errTxs = append(errTxs, tx)
				//rollback Validator balance cache
				if ledger.Validator != nil {
					ledger.Validator.RollBackAccount(tx)
				}
				log.Errorf("execute Tx hash: %s, type: %d,err: %v", tx.Hash(), tp, err)
				continue
			}

			ttxs, err := ledger.executeSmartContractTx(tx)
			if err != nil {
				errTxs = append(errTxs, tx)
				//rollback Validator balance cache
				if ledger.Validator != nil {
					ledger.Validator.RollBackAccount(tx)
				}
				log.Errorf("execute Tx hash: %s, type: %d,err: %v", tx.Hash(), tp, err)
				continue
			} else {
				var tttxs types.Transactions
				for _, tt := range ttxs {
					if err = ledger.executeTransaction(tt, false); err != nil {
						break
					}
					tttxs = append(tttxs, tt)
				}
				if len(tttxs) != len(ttxs) {
					for _, tt := range tttxs {
						ledger.executeTransaction(tt, true)
					}
					errTxs = append(errTxs, tx)
					//rollback Validator balance cache
					if ledger.Validator != nil {
						ledger.Validator.RollBackAccount(tx)
					}
					log.Errorf("execute Tx hash: %s, type: %d,err: %v", tx.Hash(), tp, err)
					continue
				}
				syncContractGenTxs = append(syncContractGenTxs, tttxs...)
			}
			syncTxs = append(syncTxs, tx)
		default:
			if err := ledger.executeTransaction(tx, false); err != nil {
				errTxs = append(errTxs, tx)
				//rollback Validator balance cache
				if ledger.Validator != nil {
					ledger.Validator.RollBackAccount(tx)
				}
				log.Errorf("execute Tx hash: %s, type: %d,err: %v", tx.Hash(), tp, err)
				continue
			}
			syncTxs = append(syncTxs, tx)
		}
	}
	for _, tx := range syncContractGenTxs {
		if ledger.Validator != nil {
			ledger.Validator.UpdateAccount(tx)
		}
	}
	writeBatchs, err = ledger.contract.AddChangesForPersistence(writeBatchs)
	if err != nil {
		panic(err)
	}
	if flag {
		syncTxs = append(syncTxs, syncContractGenTxs...)
	}
	return writeBatchs, syncTxs, errTxs
}

func (ledger *Ledger) executeTransaction(tx *types.Transaction, rollback bool) error {
	if tp := tx.GetType(); tp == types.TypeIssue || tp == types.TypeIssueUpdate {
		if err := ledger.state.UpdateAsset(tx.AssetID(), string(tx.Payload)); err != nil {
			return err
		}
	}
	plusAmount := big.NewInt(tx.Amount().Int64())
	plusFee := big.NewInt(tx.Fee().Int64())
	subAmount := big.NewInt(int64(0)).Neg(tx.Amount())
	subFee := big.NewInt(int64(0)).Neg(tx.Fee())
	if rollback {
		plusAmount, plusFee, subAmount, subFee = subAmount, subFee, plusAmount, plusFee
	}
	assetID := tx.AssetID()
	if fromChainID := coordinate.HexToChainCoordinate(tx.FromChain()).Bytes(); bytes.Equal(fromChainID, params.ChainID) {
		sender := tx.Sender()
		if err := ledger.state.UpdateBalance(sender, assetID, subAmount, tx.Nonce()); err != nil {
			if tx.GetType() == types.TypeIssue && err == state.ErrNegativeBalance {

			} else {
				ledger.state.UpdateBalance(sender, assetID, plusAmount, tx.Nonce())
				return err
			}
		}
		if err := ledger.state.UpdateBalance(sender, assetID, subFee, tx.Nonce()); err != nil {
			if tx.GetType() == types.TypeIssue && err == state.ErrNegativeBalance {

			} else {
				ledger.state.UpdateBalance(sender, assetID, plusAmount, tx.Nonce())
				ledger.state.UpdateBalance(sender, assetID, plusFee, 0)
				return err
			}
		}
		if tx.GetType() == types.TypeDistribut { //???
			chainAddress := accounts.ChainCoordinateToAddress(coordinate.HexToChainCoordinate(tx.ToChain()))
			ledger.state.UpdateBalance(chainAddress, assetID, plusAmount, 0)
			ledger.state.UpdateBalance(chainAddress, assetID, plusFee, 0)
		}
	}

	if toChainID := coordinate.HexToChainCoordinate(tx.ToChain()).Bytes(); bytes.Equal(toChainID, params.ChainID) {
		recipient := tx.Recipient()
		if err := ledger.state.UpdateBalance(recipient, assetID, plusAmount, 0); err != nil {
			ledger.state.UpdateBalance(recipient, assetID, subAmount, 0)
			return err
		}
		if err := ledger.state.UpdateBalance(recipient, assetID, plusFee, 0); err != nil {
			ledger.state.UpdateBalance(recipient, assetID, subAmount, 0)
			ledger.state.UpdateBalance(recipient, assetID, subFee, 0)
			return err
		}
		if tx.GetType() == types.TypeBackfront { //???
			chainAddress := accounts.ChainCoordinateToAddress(coordinate.HexToChainCoordinate(tx.ToChain()))
			ledger.state.UpdateBalance(chainAddress, assetID, subAmount, 0)
			ledger.state.UpdateBalance(chainAddress, assetID, subFee, 0)
		}
	}
	return nil
}

func (ledger *Ledger) executeSmartContractTx(tx *types.Transaction) (types.Transactions, error) {
	contractSpec := new(types.ContractSpec)
	utils.Deserialize(tx.Payload, contractSpec)
	ledger.contract.ExecTransaction(tx, utils.BytesToHex(contractSpec.ContractAddr))

	_, err := vm.RealExecute(tx, contractSpec, ledger.contract)
	if err != nil {
		return nil, fmt.Errorf("contract execute failed : %v ", err)
	}

	smartContractTxs, err := ledger.contract.FinishContractTransaction()
	if err != nil {
		log.Error("FinishContractTransaction: ", err)
		return nil, err
	}

	return smartContractTxs, nil
}

func (ledger *Ledger) checkCoordinate(tx *types.Transaction) bool {
	fromChainID := coordinate.HexToChainCoordinate(tx.FromChain()).Bytes()
	toChainID := coordinate.HexToChainCoordinate(tx.ToChain()).Bytes()
	if bytes.Equal(fromChainID, toChainID) {
		return true
	}
	return false
}

//GetTmpBalance get balance
func (ledger *Ledger) GetTmpBalance(addr accounts.Address) (*state.Balance, error) {
	balance, err := ledger.state.GetTmpBalance(addr)
	if err != nil {
		log.Error("can't get balance from db")
	}

	return balance, err
}

func merkleRootHash(txs []*types.Transaction) crypto.Hash {
	if len(txs) > 0 {
		hashs := make([]crypto.Hash, 0)
		for _, tx := range txs {
			hashs = append(hashs, tx.Hash())
		}
		return crypto.ComputeMerkleHash(hashs)[0]
	}
	return crypto.Hash{}
}
