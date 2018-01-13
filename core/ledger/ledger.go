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
	"io/ioutil"
	"path/filepath"

	"errors"
	"strconv"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/db/mongodb"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/ledger/block_storage"
	"github.com/bocheninc/L0/core/ledger/merge"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/vm"
	"github.com/bocheninc/L0/vm/bsvm"
)

var (
	ledgerInstance *Ledger
)

type ValidatorHandler interface {
	RemoveTxsInVerification(txs types.Transactions)
	SecurityPluginDir() string
}

// Ledger represents the ledger in blockchain
type Ledger struct {
	dbHandler *db.BlockchainDB
	block     *block_storage.Blockchain
	state     *state.BLKRWSet
	storage   *merge.Storage
	Validator ValidatorHandler
	conf      *Config
	mdb       *mongodb.Mdb
	mdbChan   chan []*db.WriteBatch
	vmEnv     map[string]*vm.VirtualMachine
}

// NewLedger returns the ledger instance
func NewLedger(kvdb *db.BlockchainDB, conf *Config) *Ledger {
	if ledgerInstance == nil {
		ledgerInstance = &Ledger{
			dbHandler: kvdb,
			block:     block_storage.NewBlockchain(kvdb),
			state:     state.NewBLKRWSet(kvdb),
			storage:   merge.NewStorage(kvdb),
		}
		_, err := ledgerInstance.Height()
		if err != nil {
			if params.Nvp && params.Mongodb {
				ledgerInstance.mdb = mongodb.MongDB()
				ledgerInstance.block.RegisterColumn(ledgerInstance.mdb)
				ledgerInstance.state.RegisterColumn(ledgerInstance.mdb)
				ledgerInstance.mdbChan = make(chan []*db.WriteBatch)
				ledgerInstance.conf = conf
			}
			ledgerInstance.init()
		}
		ledgerInstance.initVmEnv()
	}

	return ledgerInstance
}

func (ledger *Ledger) DBHandler() *db.BlockchainDB {
	return ledger.dbHandler
}

func (ledger *Ledger) initVmEnv() {
	ledger.vmEnv = make(map[string]*vm.VirtualMachine)
	bsWorkers := make([]vm.VmWorker, vm.VMConf.BsWorkerCnt)
	for i := 0; i < vm.VMConf.BsWorkerCnt; i++ {
		bsWorkers[i] = bsvm.NewBsWorker(vm.VMConf, i)
	}

	addNewEnv := func(name string, worker []vm.VmWorker) *vm.VirtualMachine {
		env := vm.CreateCustomVM(worker)
		env.Open(name)
		ledger.vmEnv[name] = env

		return env
	}

	addNewEnv("bs", bsWorkers)
}

// VerifyChain verifys the blockchain data
func (ledger *Ledger) VerifyChain() {
	height, err := ledger.Height()
	if err != nil {
		log.Panicf("VerifyChain -- Height %s", err)
	}
	currentBlockHeader, err := ledger.block.GetBlockByNumber(height)
	for i := height; i >= 1; i-- {
		previousBlockHeader, err := ledger.block.GetBlockByNumber(i - 1) // storage
		if previousBlockHeader != nil && err != nil {
			log.Panicf("VerifyChain -- GetBlockByNumber %s", err)
		}
		// verify previous block
		if !previousBlockHeader.Hash().Equal(currentBlockHeader.PreviousHash) {
			log.Panicf("VerifyChain -- block [%d] mismatch, veifychain breaks", i)
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
	//var (
	//	txWriteBatchs []*db.WriteBatch
	//	txs           types.Transactions
	//	errTxs        types.Transactions
	//)
	go ledger.Validator.RemoveTxsInVerification(block.Transactions)

	ledger.state.SetBlock(block.Height(), uint32(len(block.Transactions)))
	fn := func(data interface{}) interface{} {
		return true
	}

	wokerData := func(tx *types.Transaction, txIdx int) *vm.WorkerProc {
		return &vm.WorkerProc{
			ContractData: vm.NewContractData(tx),
			L0Handler:    state.NewTXRWSet(ledger.state, tx, uint32(txIdx)),
		}
	}

	//log.Debugf("appendBlock cnt: %+v ...........", len(block.Transactions))
	vm.NewTxSync(vm.VMConf.BsWorkerCnt)
	for idx, tx := range block.Transactions {
		ledger.vmEnv["bs"].SendWorkCleanAsync(&vm.WorkerProcWithCallback{
			WorkProc: wokerData(tx, idx),
			Idx:      idx,
			Fn:       fn,
		})
	}

	writeBatches, oktxs, errtxs, err := ledger.state.ApplyChanges()
	if err != nil || len(errtxs) != 0 {
		//TODO
		log.Errorf("AppendBlock Err: %+v, errtxs: %+v", err, len(errtxs))
	}

	log.Warnf("appendBlock cnt: %+v, oktxs: %+v, errtxs: %+v ...........", len(block.Transactions), len(oktxs), len(errtxs))

	block.Transactions = oktxs
	block.Header.TxsMerkleHash = merkleRootHash(block.Transactions)
	block.Header.StateHash = ledger.state.RootHash()
	blkWriteBatches := ledger.block.AppendBlock(block)
	writeBatches = append(writeBatches, blkWriteBatches...)

	if err := ledger.dbHandler.AtomicWrite(writeBatches); err != nil {
		return err
	}

	// bh, _ := ledger.Height()
	// ledger.state.SetHeight(bh)

	// txWriteBatchs, block.Transactions, errTxs = ledger.executeTransactions(block.Transactions, flag)

	// block.Header.TxsMerkleHash = merkleRootHash(block.Transactions)
	// writeBatchs := ledger.block.AppendBlock(block)
	// writeBatchs = append(writeBatchs, txWriteBatchs...)
	// writeBatchs = append(writeBatchs, ledger.state.WriteBatchs()...)
	// if err := ledger.dbHandler.AtomicWrite(writeBatchs); err != nil {
	// 	return err
	// }
	// if params.Nvp && params.Mongodb {
	// 	ledger.mdbChan <- writeBatchs
	// }

	// ledger.Validator.RemoveTxsInVerification(errTxs)
	// ledger.Validator.RemoveTxsInVerification(block.Transactions)

	// for _, tx := range block.Transactions {
	// 	if (tx.GetType() == types.TypeMerged && !ledger.checkCoordinate(tx)) || tx.GetType() == types.TypeAcrossChain {
	// 		txs = append(txs, tx)
	// 	}
	// }
	// if err := ledger.storage.ClassifiedTransaction(txs); err != nil {
	// 	return err
	// }
	// log.Infoln("blockHeight: ", block.Height(), "need merge Txs len : ", len(txs), "all Txs len: ", len(block.Transactions))

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

//ComplexQuery com
func (ledger *Ledger) ComplexQuery(key string) ([]byte, error) {
	return ledger.state.ComplexQuery(key)
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

// GetBalance returns balance by account
func (ledger *Ledger) GetBalance(addr accounts.Address) (*state.Balance, error) {
	return ledger.state.GetBalances(addr.String())
}

// GetAsset returns asset
func (ledger *Ledger) GetAsset(id uint32) (*state.Asset, error) {
	return ledger.state.GetAsset(id)
}

// GetAssets returns assets
func (ledger *Ledger) GetAssets() (map[uint32]*state.Asset, error) {
	return ledger.state.GetAssets()
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
	//return ledger.state.QueryContract(tx)
	return nil, nil
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

	// admin address
	buf, err := state.ConcrateStateJson(state.DefaultAdminAddr)
	if err != nil {
		return err
	}

	writeBatchs = append(writeBatchs,
		db.NewWriteBatch(ledger.state.GetChainCodeCF(),
			db.OperationPut,
			[]byte(state.ConstructCompositeKey(params.GlobalStateKey, params.AdminKey)),
			buf.Bytes(), ledger.state.GetChainCodeCF()))

	// global contract
	buf, err = state.ConcrateStateJson(&vm.ContractCode{
		state.DefaultGlobalContractCode,
		state.DefaultGlobalContractType,
	})
	if err != nil {
		return err
	}

	writeBatchs = append(writeBatchs,
		db.NewWriteBatch(ledger.state.GetChainCodeCF(),
			db.OperationPut,
			[]byte(state.ConstructCompositeKey(params.GlobalStateKey, params.GlobalContractKey)),
			buf.Bytes(), ledger.state.GetChainCodeCF()))

	err = ledger.dbHandler.AtomicWrite(writeBatchs)
	if err != nil {
		return err
	}
	if params.Nvp && params.Mongodb {
		ledger.mdb.RegisterCollection(params.GlobalStateKey)
		ledger.mdbChan <- writeBatchs
	}

	return err

}

func (ledger *Ledger) checkCoordinate(tx *types.Transaction) bool {
	fromChainID := coordinate.HexToChainCoordinate(tx.FromChain()).Bytes()
	toChainID := coordinate.HexToChainCoordinate(tx.ToChain()).Bytes()
	if bytes.Equal(fromChainID, toChainID) {
		return true
	}
	return false
}

func (ledger *Ledger) writeBlock(data interface{}) error {
	var bvalue []byte
	switch data.(type) {
	case []*db.WriteBatch:
		orgData := data.([]*db.WriteBatch)
		bvalue = utils.Serialize(orgData)
	}

	height, err := ledger.Height()
	if err != nil {
		log.Errorf("can't get blockchain height")
	}

	path := ledger.conf.ExceptBlockDir + string(filepath.Separator)
	fileName := path + strconv.Itoa(int(height))
	if utils.FileExist(fileName) {
		log.Infof("BlockChan have error, please check ...")
		return errors.New("except block have existed")
	}

	err = ioutil.WriteFile(fileName, bvalue, 0666)
	if err != nil {
		log.Errorf("write file: %s fail err: %+v", fileName, err)
	}
	return err
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

func IsJson(src []byte) bool {
	var value interface{}
	return json.Unmarshal(src, &value) == nil
}
