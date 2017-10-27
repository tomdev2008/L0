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

package state

import (
	"math/big"
	"os"
	"testing"

	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/core/accounts"
)

var (
	testConfig = &db.Config{
		DbPath:            "/tmp/rocksdb-test/",
		Columnfamilies:    []string{"balance"},
		KeepLogFileNumber: 10,
		MaxLogFileSize:    10485760,
		LogLevel:          "warn",
	}
)

func TestInitAndUpdateBalance(t *testing.T) {

	testDb := db.NewDB(testConfig)
	s := NewState(testDb)
	a := accounts.HexToAddress("0xa122277be213f56221b6140998c03d860a60e1f8")

	id := uint32(0)
	amount := big.NewInt(1024)
	nonce := uint32(10)
	if err := s.UpdateBalance(a, id, amount, nonce); err != nil {
		t.Error("update balance err:", err)
	}

	writeBatchs := s.WriteBatchs()
	s.dbHandler.AtomicWrite(writeBatchs)

	balance, err := s.GetBalance(a)
	if err != nil {
		t.Error(err)
	}
	t.Log("get balance after update", amount, nonce, balance.Get(0), balance.Nonce)

	if balance.Get(0).Cmp(amount) != 0 {
		t.Fatalf("balance %v is not equal amount %v \n", balance.Get(0), amount)
	}

	os.RemoveAll("/tmp/rocksdb-test")
}

// func TestTransfer(t *testing.T) {
// 	var writeBatchs []*db.WriteBatch
// 	testDb := db.NewDB(testConfig)
// 	s := NewState(testDb)
// 	sender := accounts.HexToAddress("0xa132277be213f56221b6140998c03d860a60e1f8")
// 	recipient := accounts.HexToAddress("0x27c649b7c4f66cfaedb99d6b38527db4deda6f41")
// 	amount := big.NewInt(1024)
// 	nonce := uint32(10)
// 	fee := big.NewInt(10)
// 	writeBatchs, err := s.UpdateBalance(sender, NewBalance(amount, nonce), fee, OperationPlus)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	s.dbHandler.AtomicWrite(writeBatchs)

// 	senderBalance, senderNonce, err := s.GetBalance(sender)
// 	recipientBalance, recipientNonce, err := s.GetBalance(recipient)

// 	t.Log("before transfer sender: ", senderBalance, senderNonce, err, "recipient", recipientBalance, recipientNonce, err)

// 	var transferWriteBatchs []*db.WriteBatch

// 	newNonce := uint32(11)
// 	transferWriteBatchs, err = s.Transfer(sender, recipient, fee, NewBalance(big.NewInt(100), newNonce), types.TypeIssue)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	s.dbHandler.AtomicWrite(transferWriteBatchs)

// 	senderBalance, senderNonce, err = s.GetBalance(sender)
// 	recipientBalance, recipientNonce, err = s.GetBalance(recipient)

// 	t.Log("after transfer sender: ", senderBalance, senderNonce, err, "recipient: ", recipientBalance, recipientNonce, err)

// 	t.Log("same address.......")

// 	newNonce = uint32(12)
// 	transferWriteBatchs, err = s.Transfer(sender, sender, fee, NewBalance(big.NewInt(100), newNonce), types.TypeIssue)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	s.dbHandler.AtomicWrite(transferWriteBatchs)

// 	senderBalance, senderNonce, err = s.GetBalance(sender)
// 	t.Log("same address after transfer: ", senderBalance, senderNonce, err)

// 	os.RemoveAll("/tmp/rocksdb-test")
// }
