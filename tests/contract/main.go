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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

var (
	fromChain      = []byte{0}
	toChain        = []byte{0}
	issuePriKeyHex = "496c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
	privkeyHex     = "596c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"

	privkey, _ = crypto.HexToECDSA(privkeyHex)
	sender     = accounts.PublicKeyToAddress(*privkey.Public())

	txChan = make(chan *types.Transaction, 5)
)

// contract lang
type contractLang string

func (lang contractLang) ConvertInitTxType() uint32 {
	if lang == langLua {
		return types.TypeLuaContractInit
	}
	return types.TypeJSContractInit
}

const (
	langLua = "lua"
	langJS  = "js"
)

// contract config
type contractConf struct {
	path     string
	lang     contractLang
	isGlobal bool
	args     []string
}

func newContractConf(path string, lang contractLang, isGlobal bool, args []string) *contractConf {
	return &contractConf{
		path:     path,
		lang:     lang,
		isGlobal: isGlobal,
		args:     args,
	}
}

var (
	gopath = os.Getenv("GOPATH")

	voteLua = newContractConf(
		gopath+"/src/github.com/bocheninc/L0/tests/contract/l0vote.lua",
		langLua,
		false,
		[]string{"vote", "张三", "秦皇岛"})

	coinLua = newContractConf(
		gopath+"/src/github.com/bocheninc/L0/tests/contract/l0coin.lua",
		langLua,
		false,
		[]string{"transfer", "8ce1bb0858e71b50d603ebe4bec95b11d8833e68", "100"})

	coinJS = newContractConf(
		gopath+"/src/github.com/bocheninc/L0/tests/contract/l0coin.js",
		langJS,
		false,
		[]string{"transfer", "8ce1bb0858e71b50d603ebe4bec95b11d8833e68", "100"})

	globalLua = newContractConf(
		gopath+"/src/github.com/bocheninc/L0/tests/contract/global.lua",
		langLua,
		false,
		[]string{"SetGlobalState", "admin", sender.String()})
)

func main() {
	go sendTransaction()

	conf := globalLua
	deploySmartContractTX(conf)
	time.Sleep(1 * time.Second)

	execSmartContractTX(conf)

	time.Sleep(5 * time.Second)
}

func sendTransaction() {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
		Timeout: time.Second * 500,
	}
	URL := "http://" + "localhost" + ":" + "8881"
	for {
		select {
		case tx := <-txChan:
			//tx := new(types.Transaction)
			//tx.Deserialize(utils.HexToBytes(txHex))
			fmt.Println(" hash: ", tx.Hash().String(), " type ", tx.GetType(), " nonce: ", tx.Nonce(), " amount: ", tx.Amount())
			req, _ := http.NewRequest("POST", URL, bytes.NewBufferString(
				`{"id":1,"method":"Transaction.Broadcast","params":["`+hex.EncodeToString(tx.Serialize())+`"]}`,
			))
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(fmt.Errorf("Couldn't parse response body. %+v", err))
			}
			var dat map[string]interface{}
			json.Unmarshal(body, &dat)
			fmt.Println(dat)
		}
	}
}

func deploySmartContractTX(conf *contractConf) {
	contractSpec := new(types.ContractSpec)
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
		big.NewInt(0),
		big.NewInt(0),
		uint32(time.Now().Unix()),
	)
	tx.Payload = utils.Serialize(contractSpec)
	sig, _ := privkey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	txChan <- tx
}

func execSmartContractTX(conf *contractConf) {
	contractSpec := new(types.ContractSpec)
	contractSpec.ContractParams = conf.args

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
		conf.lang.ConvertInitTxType(),
		uint32(nonce),
		sender,
		accounts.NewAddress(contractSpec.ContractAddr),
		big.NewInt(0),
		big.NewInt(0),
		uint32(time.Now().Unix()),
	)
	fmt.Println("ContractAddr:", accounts.NewAddress(contractSpec.ContractAddr).String())
	tx.Payload = utils.Serialize(contractSpec)
	sig, _ := privkey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	txChan <- tx
}
