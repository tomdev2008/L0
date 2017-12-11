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
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
)

var (
	issueReciepent     = accounts.HexToAddress("0xa032277be213f56221b6140998c03d860a60e1f8")
	atmoicReciepent    = accounts.HexToAddress("0xa132277be213f56221b6140998c03d860a60e1f8")
	acrossReciepent    = accounts.HexToAddress("0xa232277be213f56221b6140998c03d860a60e1f8")
	distributReciepent = accounts.HexToAddress("0xa332277be213f56221b6140998c03d860a60e1f8")
	backfrontReciepent = accounts.HexToAddress("0xa432277be213f56221b6140998c03d860a60e1f8")

	issueAmount = big.NewInt(100)
	Amount      = big.NewInt(1)
	fee         = big.NewInt(0)
)

func TestExecuteIssueTx(t *testing.T) {
	testDb := db.NewDB(db.DefaultConfig())
	li := NewLedger(testDb, &Config{"file"})
	defer os.RemoveAll("/tmp/rocksdb-test")

	params.ChainID = []byte{byte(0)}

	issueTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*issueTxKeypair.Public())

	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(0),
		addr,
		issueReciepent,
		uint32(0),
		issueAmount,
		fee,
		utils.CurrentTimestamp())
	issueCoin := make(map[string]interface{})
	issueCoin["id"] = 0
	issueTx.Payload, _ = json.Marshal(issueCoin)
	signature, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature)

	err := li.executeTransaction(issueTx, false)
	if err != nil {
		t.Error(err)
	}

	li.dbHandler.AtomicWrite(li.state.WriteBatchs())

	sender := issueTx.Sender()
	t.Log(sender.String())
	b, _ := li.GetBalanceFromDB(sender)
	t.Log(b.Amounts[0].Sign())
	t.Log(li.GetBalanceFromDB(sender))
	t.Log(li.GetBalanceFromDB(issueReciepent))

}

func TestExecuteAtmoicTx(t *testing.T) {

	testDb := db.NewDB(db.DefaultConfig())
	li := NewLedger(testDb, &Config{"file"})
	defer os.RemoveAll("/tmp/rocksdb-test")

	params.ChainID = []byte{byte(0)}

	atmoicTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*atmoicTxKeypair.Public())

	atmoicTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeAtomic,
		uint32(0),
		addr,
		atmoicReciepent,
		uint32(0),
		Amount,
		fee,
		utils.CurrentTimestamp())

	atmoicCoin := make(map[string]interface{})
	atmoicCoin["id"] = 0
	atmoicTx.Payload, _ = json.Marshal(atmoicCoin)

	signature, _ := atmoicTxKeypair.Sign(atmoicTx.Hash().Bytes())
	atmoicTx.WithSignature(signature)
	atmoicSender := atmoicTx.Sender()

	issueTxKeypair, _ := crypto.GenerateKey()
	addr = accounts.PublicKeyToAddress(*issueTxKeypair.Public())

	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(0),
		addr,
		atmoicSender,
		uint32(0),
		issueAmount,
		fee,
		utils.CurrentTimestamp())
	issueCoin := make(map[string]interface{})
	issueCoin["id"] = 0
	issueTx.Payload, _ = json.Marshal(issueCoin)
	signature1, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature1)

	_, _, errtxs := li.executeTransactions(types.Transactions{issueTx, atmoicTx}, false)
	if len(errtxs) > 0 {
		t.Error("error")
	}

	li.dbHandler.AtomicWrite(li.state.WriteBatchs())

	sender := issueTx.Sender()
	b, _ := li.GetBalanceFromDB(sender)
	t.Logf("issueTx sender : %v", b.Amounts[0].Sign())
	t.Log(li.GetBalanceFromDB(atmoicSender))
	t.Log(li.GetBalanceFromDB(atmoicReciepent))

}

func TestExecuteAcossTx1(t *testing.T) {
	testDb := db.NewDB(db.DefaultConfig())
	li := NewLedger(testDb, &Config{"file"})
	defer os.RemoveAll("/tmp/rocksdb-test")

	params.ChainID = []byte{byte(0)}

	acrossTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*acrossTxKeypair.Public())

	acrossTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(1)}),
		types.TypeAcrossChain,
		uint32(0),
		addr,
		acrossReciepent,
		uint32(0),
		Amount,
		fee,
		utils.CurrentTimestamp())
	acrossCoin := make(map[string]interface{})
	acrossCoin["id"] = 0
	acrossTx.Payload, _ = json.Marshal(acrossCoin)
	signature, _ := acrossTxKeypair.Sign(acrossTx.Hash().Bytes())
	acrossTx.WithSignature(signature)
	acrossSender := acrossTx.Sender()

	issueTxKeypair, _ := crypto.GenerateKey()
	addr = accounts.PublicKeyToAddress(*issueTxKeypair.Public())
	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(0),
		addr,
		acrossSender,
		uint32(0),
		issueAmount,
		fee,
		utils.CurrentTimestamp())
	issueCoin := make(map[string]interface{})
	issueCoin["id"] = 0
	issueTx.Payload, _ = json.Marshal(issueCoin)
	signature1, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature1)

	_, _, errtxs := li.executeTransactions(types.Transactions{issueTx, acrossTx}, false)
	if len(errtxs) > 0 {
		t.Error("error")
	}

	li.dbHandler.AtomicWrite(li.state.WriteBatchs())

	sender := issueTx.Sender()
	t.Log(li.GetBalanceFromDB(sender))
	t.Log(li.GetBalanceFromDB(acrossSender))
	t.Log(li.GetBalanceFromDB(acrossReciepent))
}

func TestExecuteAcossTx2(t *testing.T) {
	testDb := db.NewDB(db.DefaultConfig())
	li := NewLedger(testDb, &Config{"file"})
	defer os.RemoveAll("/tmp/rocksdb-test")

	params.ChainID = []byte{byte(0)}

	acrossTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*acrossTxKeypair.Public())

	acrossTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(1)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeAcrossChain,
		uint32(0),
		addr,
		acrossReciepent,
		uint32(0),
		Amount,
		fee,
		utils.CurrentTimestamp())
	acrossCoin := make(map[string]interface{})
	acrossCoin["id"] = 0
	acrossTx.Payload, _ = json.Marshal(acrossCoin)
	signature, _ := acrossTxKeypair.Sign(acrossTx.Hash().Bytes())
	acrossTx.WithSignature(signature)
	acrossSender := acrossTx.Sender()

	issueTxKeypair, _ := crypto.GenerateKey()
	addr = accounts.PublicKeyToAddress(*issueTxKeypair.Public())

	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(0),
		addr,
		acrossReciepent,
		uint32(0),
		issueAmount,
		fee,
		utils.CurrentTimestamp())
	issueCoin := make(map[string]interface{})
	issueCoin["id"] = 0
	issueTx.Payload, _ = json.Marshal(issueCoin)
	signature1, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature1)

	_, _, errtxs := li.executeTransactions(types.Transactions{issueTx, acrossTx}, false)
	if len(errtxs) > 0 {
		t.Error("error")
	}
	li.dbHandler.AtomicWrite(li.state.WriteBatchs())

	sender := issueTx.Sender()
	t.Log(li.GetBalanceFromDB(sender))
	t.Log(li.GetBalanceFromDB(acrossSender))
	t.Log(li.GetBalanceFromDB(acrossReciepent))
}

func TestExecuteMergedTx(t *testing.T) {
	testDb := db.NewDB(db.DefaultConfig())
	li := NewLedger(testDb, &Config{"file"})
	defer os.RemoveAll("/tmp/rocksdb-test")

	params.ChainID = []byte{byte(0)}

	from := coordinate.NewChainCoordinate([]byte{byte(0)})
	to := coordinate.NewChainCoordinate([]byte{byte(0)})
	sender := coordinate.NewChainCoordinate([]byte{byte(0), byte(0)})
	reciepent := coordinate.NewChainCoordinate([]byte{byte(0), byte(1)})

	mergedTx := types.NewTransaction(from,
		to,
		types.TypeMerged,
		uint32(0),
		accounts.ChainCoordinateToAddress(sender),
		accounts.ChainCoordinateToAddress(reciepent),
		uint32(0),
		Amount,
		fee,
		utils.CurrentTimestamp())
	mergeCoin := make(map[string]interface{})
	mergeCoin["id"] = 0
	mergedTx.Payload, _ = json.Marshal(mergeCoin)

	senderAddress := accounts.ChainCoordinateToAddress(sender)
	sig := &crypto.Signature{}
	copy(sig[:], senderAddress[:])
	mergedTx.WithSignature(sig)

	issueTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*issueTxKeypair.Public())

	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(0),
		addr,
		senderAddress,
		uint32(0),
		issueAmount,
		fee,
		utils.CurrentTimestamp())

	issueCoin := make(map[string]interface{})
	issueCoin["id"] = 0
	issueTx.Payload, _ = json.Marshal(issueCoin)

	signature1, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature1)

	_, _, errtxs := li.executeTransactions(types.Transactions{issueTx, mergedTx}, false)
	if len(errtxs) > 0 {
		t.Error("error")
	}
	li.dbHandler.AtomicWrite(li.state.WriteBatchs())

	issueSenderaddress := issueTx.Sender()

	t.Log(li.GetBalanceFromDB(issueSenderaddress))
	t.Log(li.GetBalanceFromDB(accounts.ChainCoordinateToAddress(sender)))
	t.Log(li.GetBalanceFromDB(accounts.ChainCoordinateToAddress(reciepent)))
}

func TestExecuteDistributTx(t *testing.T) {
	testDb := db.NewDB(db.DefaultConfig())
	li := NewLedger(testDb, &Config{"file"})
	defer os.RemoveAll("/tmp/rocksdb-test")

	params.ChainID = []byte{byte(0)}

	distributTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*distributTxKeypair.Public())

	distributTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(1)}),
		types.TypeDistribut,
		uint32(1),
		addr,
		distributReciepent,
		uint32(0),
		Amount,
		fee,
		utils.CurrentTimestamp())

	distributCoin := make(map[string]interface{})
	distributCoin["id"] = 0
	distributTx.Payload, _ = json.Marshal(distributCoin)

	signature, _ := distributTxKeypair.Sign(distributTx.Hash().Bytes())
	distributTx.WithSignature(signature)
	distributAddress := distributTx.Sender()

	issueTxKeypair, _ := crypto.GenerateKey()
	addr = accounts.PublicKeyToAddress(*issueTxKeypair.Public())
	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(0),
		addr,
		distributAddress,
		uint32(0),
		issueAmount,
		fee,
		utils.CurrentTimestamp())
	issueCoin := make(map[string]interface{})
	issueCoin["id"] = 0
	issueTx.Payload, _ = json.Marshal(issueCoin)

	signature1, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature1)

	_, _, errtxs := li.executeTransactions(types.Transactions{issueTx, distributTx}, false)
	if len(errtxs) > 0 {
		t.Error("error")
	}
	li.dbHandler.AtomicWrite(li.state.WriteBatchs())

	sender := issueTx.Sender()
	t.Log(li.GetBalanceFromDB(sender))
	t.Log(li.GetBalanceFromDB(accounts.ChainCoordinateToAddress(coordinate.NewChainCoordinate([]byte{byte(1)}))))
	t.Log(li.GetBalanceFromDB(distributAddress))
	t.Log(li.GetBalanceFromDB(distributReciepent))
}

func TestExecuteBackfrontTx(t *testing.T) {
	testDb := db.NewDB(db.DefaultConfig())
	li := NewLedger(testDb, &Config{"file"})
	defer os.RemoveAll("/tmp/rocksdb-test")

	params.ChainID = []byte{byte(1)}

	backfrontTxKeypair, _ := crypto.GenerateKey()
	addr := accounts.PublicKeyToAddress(*backfrontTxKeypair.Public())

	backfrontTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(1)}),
		types.TypeBackfront,
		uint32(1),
		addr,
		backfrontReciepent,
		uint32(0),
		Amount,
		fee,
		utils.CurrentTimestamp())
	backforntCoin := make(map[string]interface{})
	backforntCoin["id"] = 0
	backfrontTx.Payload, _ = json.Marshal(backforntCoin)
	signature, _ := backfrontTxKeypair.Sign(backfrontTx.Hash().Bytes())
	backfrontTx.WithSignature(signature)
	backfrontAddrress := backfrontTx.Sender()

	issueTxKeypair, _ := crypto.GenerateKey()
	addr = accounts.PublicKeyToAddress(*issueTxKeypair.Public())
	issueTx := types.NewTransaction(coordinate.NewChainCoordinate([]byte{byte(0)}),
		coordinate.NewChainCoordinate([]byte{byte(0)}),
		types.TypeIssue,
		uint32(0),
		addr,
		backfrontAddrress,
		uint32(0),
		issueAmount,
		fee,
		utils.CurrentTimestamp())
	issueCoin := make(map[string]interface{})
	issueCoin["id"] = 0
	issueTx.Payload, _ = json.Marshal(issueCoin)
	signature1, _ := issueTxKeypair.Sign(issueTx.Hash().Bytes())
	issueTx.WithSignature(signature1)

	_, _, errtxs := li.executeTransactions(types.Transactions{issueTx}, false)
	if len(errtxs) > 0 {
		t.Error("error")
	}

	li.dbHandler.AtomicWrite(li.state.WriteBatchs())

	sender := issueTx.Sender()
	t.Log(li.GetBalanceFromDB(sender))
	t.Log(li.GetBalanceFromDB(accounts.ChainCoordinateToAddress(coordinate.NewChainCoordinate([]byte{byte(0)}))))
	t.Log(li.GetBalanceFromDB(backfrontAddrress))
	t.Log(li.GetBalanceFromDB(backfrontReciepent))

}
