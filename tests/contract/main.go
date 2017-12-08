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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/plugins"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

var (
	fromChain = []byte{0}
	toChain   = []byte{0}

	issuePriKeyHex = "496c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
	privkeyHex     = "596c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
	privkey, _     = crypto.HexToECDSA(privkeyHex)
	sender         = accounts.PublicKeyToAddress(*privkey.Public())

	senderAccount = "abc123"
	accountInfo   = map[string]string{
		"from_user": senderAccount,
		"to_user":   senderAccount,
	}

	txChan = make(chan *types.Transaction, 5)

	httpClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
		Timeout: time.Second * 500,
	}
	url = "http://localhost:8881"
)

// contract lang
type contractLang string

func (lang contractLang) ConvertInitTxType() uint32 {
	switch lang {
	case langLua:
		return types.TypeLuaContractInit
	case langJS:
		return types.TypeJSContractInit
	}
	return 0
}

const (
	langLua = "lua"
	langJS  = "js"
)

// contract config
type contractConf struct {
	path       string
	lang       contractLang
	isGlobal   bool
	initArgs   []string
	invokeArgs []string
}

func newContractConf(path string, lang contractLang, isGlobal bool, initArgs, invokeArgs []string) *contractConf {
	return &contractConf{
		path:       path,
		lang:       lang,
		isGlobal:   isGlobal,
		initArgs:   initArgs,
		invokeArgs: invokeArgs,
	}
}

var (
	voteLua = newContractConf(
		"./l0vote.lua",
		langLua,
		false,
		nil,
		[]string{"vote", "张三", "秦皇岛"})

	coinLua = newContractConf(
		"./l0coin.lua",
		langLua,
		false,
		nil,
		[]string{"transfer", "8ce1bb0858e71b50d603ebe4bec95b11d8833e68", "100"})
	//[]string{"testwrite", "8ce1bb0858e71b50d603ebe4bec95b11d8833e68", "100"})

	coinJS = newContractConf(
		"./l0coin.js",
		langJS,
		false,
		[]string{"hello", "world"},
		//[]string{"transfer", "8ce1bb0858e71b50d603ebe4bec95b11d8833e68", "100"})
		[]string{"testwrite", "8ce1bb0858e71b50d603ebe4bec95b11d8833e68", "100"})

	globalSetAccountLua = newContractConf(
		"./global.lua",
		langLua,
		true,
		nil,
		[]string{
			"SetGlobalState",
			"account." + senderAccount,
			fmt.Sprintf(`{"addr":"%s", "uid":"%s", "frozened":false}`, sender.String(), senderAccount),
		})

	securityPluginName = "security.so"
)

func main() {
	go sendTransaction()
	time.Sleep(1 * time.Microsecond)

	issueTX()
	//testSecurityContract()
	deploySmartContractTX(coinJS)
	// time.Sleep(10 * time.Second)
	// execSmartContractTX(coinJS)

	ch := make(chan struct{})
	<-ch
}

func httpPost(postForm string, resultHandler func(result map[string]interface{})) {
	req, _ := http.NewRequest("POST", url, strings.NewReader(postForm))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Errorf("Couldn't parse response body. %+v", err))
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	if resultHandler == nil {
		fmt.Println("http response:", result)
	} else {
		resultHandler(result)
	}
}

func sendTransaction() {
	for {
		select {
		case tx := <-txChan:
			fmt.Printf("hash: %s, type: %v, nonce: %v, amount: %v\n",
				tx.Hash().String(), tx.GetType(), tx.Nonce(), tx.Amount())

			httpPost(`{"id":1,"method":"Transaction.Broadcast","params":["`+hex.EncodeToString(tx.Serialize())+`"]}`, nil)
		}
	}
}

func issueTX() {
	issueKey, _ := crypto.HexToECDSA(issuePriKeyHex)
	nonce := 1
	issueSender := accounts.PublicKeyToAddress(*issueKey.Public())

	privateKey, _ := crypto.GenerateKey()
	accounts.PublicKeyToAddress(*privateKey.Public())
	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		types.TypeIssue,
		uint32(nonce),
		issueSender,
		sender,
		1,
		big.NewInt(1000),
		big.NewInt(1),
		uint32(time.Now().Unix()),
	)

	issueCoin := make(map[string]interface{})
	issueCoin["id"] = 1
	tx.Payload, _ = json.Marshal(issueCoin)
	tx.Meta, _ = json.Marshal(map[string]map[string]string{
		"account": accountInfo,
	})

	fmt.Println("issueSender address: ", issueSender.String(), " receriver: ", sender.String())
	sig, _ := issueKey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	txChan <- tx
}

func deploySmartContractTX(conf *contractConf) []byte {
	contractSpec := new(types.ContractSpec)
	contractSpec.ContractParams = conf.initArgs

	f, _ := os.Open(conf.path)
	buf, _ := ioutil.ReadAll(f)
	contractSpec.ContractCode = buf

	if !conf.isGlobal {
		var a accounts.Address
		pubBytes := []byte(sender.String() + string(buf))
		a.SetBytes(crypto.Keccak256(pubBytes[1:])[12:])
		contractSpec.ContractAddr = a.Bytes()
	}

	nonce := 1
	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		conf.lang.ConvertInitTxType(),
		uint32(nonce),
		sender,
		accounts.NewAddress(contractSpec.ContractAddr),
		1,
		big.NewInt(100),
		big.NewInt(0),
		uint32(time.Now().Unix()),
	)

	tx.Payload = utils.Serialize(contractSpec)

	sig, _ := privkey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	fmt.Println("> deploy ContractAddr:", accounts.NewAddress(contractSpec.ContractAddr).String())

	txChan <- tx
	return contractSpec.ContractAddr
}

func execSmartContractTX(conf *contractConf) {
	contractSpec := new(types.ContractSpec)
	contractSpec.ContractParams = conf.invokeArgs

	if !conf.isGlobal {
		f, _ := os.Open(conf.path)
		buf, _ := ioutil.ReadAll(f)

		var a accounts.Address
		pubBytes := []byte(sender.String() + string(buf))
		a.SetBytes(crypto.Keccak256(pubBytes[1:])[12:])

		contractSpec.ContractAddr = a.Bytes()
	}

	nonce := 2
	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		types.TypeContractInvoke,
		uint32(nonce),
		sender,
		accounts.NewAddress(contractSpec.ContractAddr),
		0,
		big.NewInt(0),
		big.NewInt(0),
		uint32(time.Now().Unix()),
	)

	fmt.Println("> exe ContractAddr:", accounts.NewAddress(contractSpec.ContractAddr).String())
	tx.Payload = utils.Serialize(contractSpec)

	sig, _ := privkey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	txChan <- tx
}

func queryGlobalContract(key string) {
	form := `{"id": 2, "method": "Transaction.Query", "params":[{"ContractAddr":"","ContractParams":["` + key + `"]}]}`
	httpPost(form, func(result map[string]interface{}) {
		if result != nil {
			fmt.Printf("> query result: %s\n", result["result"])
		} else {
			fmt.Println("> query failed")
		}
	})
}

func deploySecurity() {
	nonce := 3
	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		types.TypeSecurity,
		uint32(nonce),
		sender,
		accounts.Address{},
		0,
		big.NewInt(0),
		big.NewInt(0),
		uint32(time.Now().Unix()),
	)

	var pluginData plugins.Plugin
	pluginData.Name = securityPluginName
	pluginData.Code, _ = ioutil.ReadFile("./" + securityPluginName)
	tx.Payload = pluginData.Bytes()

	sig, _ := privkey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)

	txChan <- tx
}

func testSecurityContract() {
	issueTX()

	// global contract
	// deploySmartContractTX(globalSetAccountLua)

	time.Sleep(1 * time.Second)
	execSmartContractTX(globalSetAccountLua)

	// security contract
	time.Sleep(1 * time.Second)
	deploySecurity()

	time.Sleep(1 * time.Second)
	globalLua := newContractConf(
		"./global.lua",
		langLua,
		true,
		nil,
		[]string{"SetGlobalState", "securityContract", `"security.so"`})
	execSmartContractTX(globalLua)

	// query
	time.Sleep(6 * time.Second)
	queryGlobalContract("securityContract")

	time.Sleep(3 * time.Second)
}
