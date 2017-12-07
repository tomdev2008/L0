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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"path/filepath"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/plugins"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/ledger/block_storage"
	"github.com/bocheninc/L0/core/ledger/contract"
	"github.com/bocheninc/L0/core/ledger/merge"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
)

var (
	ledgerInstance *Ledger
)

type ValidatorHandler interface {
	UpdateAccount(tx *types.Transaction) bool
	RollBackAccount(tx *types.Transaction)
	RemoveTxsInVerification(txs types.Transactions)
	SecurityPluginDir() string
	Notify(*types.Transaction, interface{})
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

func (ledger *Ledger) DBHandler() *db.BlockchainDB {
	return ledger.dbHandler
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
	return ledger.contract.QueryContract(tx)
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
	if err := ledger.state.UpdateAsset(0, accounts.Address{}, accounts.Address{}, "{}"); err != nil {
		panic(err)
	}
	writeBatchs = append(writeBatchs, ledger.state.WriteBatchs()...)

	// admin address
	buf, err := contract.ConcrateStateJson(contract.DefaultAdminAddr)
	if err != nil {
		return err
	}

	writeBatchs = append(writeBatchs,
		db.NewWriteBatch(contract.ColumnFamily,
			db.OperationPut,
			[]byte(contract.EnSmartContractKey(params.GlobalStateKey, params.AdminKey)),
			buf.Bytes()))

	// global contract
	writeBatchs = append(writeBatchs,
		db.NewWriteBatch(contract.ColumnFamily,
			db.OperationPut,
			[]byte(contract.EnSmartContractKey(params.GlobalStateKey, params.GlobalContractKey)),
			utils.Serialize(&contract.DefaultGlobalContract)))

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
				goto ctu
			}

			ttxs, err := ledger.executeSmartContractTx(tx)
			if err != nil {
				errTxs = append(errTxs, tx)
				//rollback Validator balance cache
				if ledger.Validator != nil {
					ledger.Validator.RollBackAccount(tx)
				}
				log.Errorf("execute Tx hash: %s, type: %d,err: %v", tx.Hash(), tp, err)
				goto ctu
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
					goto ctu
				}
				syncContractGenTxs = append(syncContractGenTxs, tttxs...)
			}
			syncTxs = append(syncTxs, tx)
		case types.TypeSecurity:
			adminData, err := ledger.contract.GetContractStateData(params.GlobalStateKey, params.AdminKey)
			if err != nil {
				log.Error(err)
				goto ctu
			}

			if len(adminData) == 0 {
				log.Error("need admin address")
				goto ctu
			}

			var adminAddr accounts.Address
			err = json.Unmarshal(adminData, &adminAddr)
			if err != nil {
				log.Error(err)
				goto ctu
			}

			if tx.Sender() != adminAddr {
				log.Error("deploy security plugin, permission denied")
				goto ctu
			}

			pluginData, err := plugins.Make(tx.Payload)
			if err != nil {
				log.Error("invalid security plugin data")
				goto ctu
			}

			if len(pluginData.Name) == 0 {
				log.Error("name of security plugin is empty")
				goto ctu
			}

			path := filepath.Join(ledger.Validator.SecurityPluginDir(), pluginData.Name)
			if utils.FileExist(path) {
				log.Errorf("security plugin %s already existed", pluginData.Name)
				goto ctu
			}

			err = ioutil.WriteFile(path, pluginData.Code, 0644)
			if err != nil {
				log.Error(err)
				goto ctu
			}
		default:
			if err := ledger.executeTransaction(tx, false); err != nil {
				errTxs = append(errTxs, tx)
				//rollback Validator balance cache
				if ledger.Validator != nil {
					ledger.Validator.RollBackAccount(tx)
				}
				log.Errorf("execute Tx hash: %s, type: %d,err: %v", tx.Hash(), tp, err)
				goto ctu
			}
			syncTxs = append(syncTxs, tx)
		}

	ctu:
		ledger.Validator.Notify(tx, "execute Tx error:"+err.Error())
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
	tp := tx.GetType()
	if tp == types.TypeIssue {
		if err := ledger.state.UpdateAsset(tx.AssetID(), tx.Sender(), tx.Recipient(), string(tx.Payload)); err != nil {
			return err
		}
	} else if tp == types.TypeIssueUpdate {
		if err := ledger.state.UpdateAsset(tx.AssetID(), tx.Sender(), tx.Recipient(), string(tx.Payload)); err != nil {
			if err := ledger.state.UpdateAsset(tx.AssetID(), tx.Recipient(), tx.Sender(), string(tx.Payload)); err != nil {
				return err
			}
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
			if (tx.GetType() == types.TypeIssue || tx.GetType() == types.TypeIssueUpdate) && err == state.ErrNegativeBalance {

			} else {
				ledger.state.UpdateBalance(sender, assetID, plusAmount, tx.Nonce())
				return err
			}
		}
		if err := ledger.state.UpdateBalance(sender, assetID, subFee, tx.Nonce()); err != nil {
			if (tx.GetType() == types.TypeIssue || tx.GetType() == types.TypeIssueUpdate) && err == state.ErrNegativeBalance {

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
	return ledger.contract.ExecuteSmartContractTx(tx)
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
