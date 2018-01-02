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
package main

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/tests/common"
	"github.com/pborman/uuid"
	//"net/http"
	"strconv"
)

var (
	issuePriKeyHex = "496c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
	privkeyHex     = "596c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
	defaultURL     = "http://127.0.0.1:8881"
)

type Contract struct {
	dirPath string
}

func (ct *Contract) createInvokeTransaction() *types.Transaction {
	uid := uuid.NewUUID().String()
	rand.Seed(time.Now().Unix())
	amount := rand.Intn(100)
	return ct.invokeContract("invoke", "./l0coin.lua", []string{}, []string{"send", uid, strconv.Itoa(amount), uid})
}

func (ct *Contract) createInitTransaction() *types.Transaction {
	return ct.initContract("lua", "./l0coin.lua", []string{"test"}, []string{})
}

func (ct *Contract) invokeContract(typ, contractPath string, initArgs, invokeArgs []string) *types.Transaction {
	txConf := common.NewContractTxConf([]byte{0}, []byte{0}, big.NewInt(100), big.NewInt(0), privkeyHex)
	contractConf := common.NewContractConf(filepath.Join(ct.dirPath, contractPath), "invoke", false, initArgs, invokeArgs)
	tx := common.CreateContractTransaction(txConf, contractConf)

	return tx
}

func (ct *Contract) initContract(typ, contractPath string, initArgs, invokeArgs []string) *types.Transaction {
	txConf := common.NewContractTxConf([]byte{0}, []byte{0}, big.NewInt(100), big.NewInt(0), privkeyHex)
	contractConf := common.NewContractConf(filepath.Join(ct.dirPath, contractPath), typ, false, initArgs, invokeArgs)
	tx := common.CreateContractTransaction(txConf, contractConf)
	return tx
}

func (ct *Contract) init() *types.Transaction {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Errorf("filePath: %+v", err)
	}
	ct.dirPath = dir //filepath.Join(dir, )

	assetID := 1
	tx := common.CreateIssueTransaction(issuePriKeyHex, privkeyHex, assetID)
	return tx
}
