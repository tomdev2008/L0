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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/plugins"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/tests/common"
)

var (
	privkey, _ = crypto.HexToECDSA(privkeyHex)
	sender     = accounts.PublicKeyToAddress(*privkey.Public())

	senderAccount = "abc123"
	accountInfo   = map[string]string{
		"from_user": senderAccount,
		"to_user":   senderAccount,
	}

	securityPluginName = "plugin.so"
)

type Security struct {
}

func (s *Security) createSetAccountTx() *types.Transaction {
	invokeArgs := []string{
		"SetGlobalState",
		"account." + senderAccount,
		fmt.Sprintf(`{"addr":"%s", "uid":"%s", "frozened":false}`, sender.String(), senderAccount),
	}
	txConf := common.NewContractTxConf([]byte{0}, []byte{0}, big.NewInt(1), big.NewInt(0), privkeyHex)
	contractConf := common.NewContractConf("", "invoke", true, []string{}, invokeArgs)
	tx := common.CreateContractTransaction(txConf, contractConf)
	return tx
}

func (s *Security) createDeployTx() *types.Transaction {
	tx := types.NewTransaction(
		coordinate.NewChainCoordinate([]byte{0}),
		coordinate.NewChainCoordinate([]byte{0}),
		types.TypeSecurity,
		uint32(time.Now().UnixNano()),
		sender,
		accounts.Address{},
		1,
		big.NewInt(0),
		big.NewInt(0),
		uint32(time.Now().Unix()))

	var pluginData plugins.Plugin
	pluginData.Name = securityPluginName
	pluginData.Code, _ = ioutil.ReadFile("./" + securityPluginName)
	tx.Payload = pluginData.Bytes()
	sig, _ := privkey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)

	return tx

}

func (s *Security) createSetPluginTx() *types.Transaction {
	invokeArgs := []string{"SetGlobalState", "securityContract", `"plugin.so"`}
	txConf := common.NewContractTxConf([]byte{0}, []byte{0}, big.NewInt(100), big.NewInt(0), privkeyHex)
	contractConf := common.NewContractConf("", "invoke", true, []string{}, invokeArgs)
	tx := common.CreateContractTransaction(txConf, contractConf)
	return tx
}

func (s *Security) createAtomicTx() *types.Transaction {
	privateKey, _ := crypto.GenerateKey()
	txConf := common.NewNormalTxConf([]byte{0}, []byte{0}, big.NewInt(100), big.NewInt(0), 1, accounts.PublicKeyToAddress(*privateKey.Public()), privkeyHex)
	meta, _ := json.Marshal(map[string]map[string]string{
		"account": accountInfo,
	})
	tx := common.CreateNormalTransaction(txConf, meta)
	return tx
}

func (s *Security) init() *types.Transaction {
	assetID := 1
	tx := common.CreateIssueTransaction(issuePriKeyHex, privkeyHex, assetID)
	return tx
}
