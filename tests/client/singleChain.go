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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

var (
	srvAddress = []string{
		"127.0.0.1:20166",
		//"127.0.0.1:20167",
		//"127.0.0.1:20168",
		//"127.0.0.1:20169",
		//"127.0.0.1:20170",
	}

	index  = time.Now().Nanosecond()
	list   = make(chan *crypto.PrivateKey, 10)
	txChan = make(chan *types.Transaction, 1000)
	//issuePriKeyHex = "496c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
)

func sendTx() {
	TCPSend(srvAddress)
	time.Sleep(time.Second * 20)
	fmt.Println("start Send ...")
	//go generateIssueTx()
	//go generateAtomicTx()
	//go generateContract()
	go generateSecurity()
	for {
		select {
		case tx := <-txChan:
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "Hash:", tx.Hash(), "Sender:", tx.Sender(), " Nonce: ", tx.Nonce(), "Asset: ", tx.AssetID(), " Type:", tx.GetType(), "txChan size:", len(txChan))
			Relay(NewMsg(0x14, tx.Serialize()))
		}
	}
}

func generateContract() {
	ct := &Contract{}
	txChan <- ct.init()
	time.Sleep(2 * time.Second)
	txChan <- ct.createInitTransaction()
	time.Sleep(2 * time.Second)
	for {
		time.Sleep(time.Second)
		txChan <- ct.createInvokeTransaction()
	}
}

func generateSecurity() {
	s := &Security{}
	txChan <- s.init()
	time.Sleep(1 * time.Second)
	txChan <- s.createSetAccountTx()
	time.Sleep(1 * time.Second)
	txChan <- s.createDeployTx()
	time.Sleep(1 * time.Second)
	txChan <- s.createSetPluginTx()
	time.Sleep(10 * time.Second)
	for {
		time.Sleep(10 * time.Millisecond)
		txChan <- s.createAtomicTx()
	}
}

func generateAtomicTx() {
	var (
		fromChain = []byte{0}
		toChain   = []byte{0}
	)

	id := uint32(0)
	for {
		select {
		case key := <-list:
			go func(privateKey *crypto.PrivateKey, id uint32) {
				time.Sleep(time.Second * 60)
				sender := accounts.PublicKeyToAddress(*privateKey.Public())
				nonce := uint32(0)
				for {
					nonce++
					privkey, _ := crypto.GenerateKey()
					addr := accounts.PublicKeyToAddress(*privkey.Public())
					tx := types.NewTransaction(
						coordinate.NewChainCoordinate(fromChain),
						coordinate.NewChainCoordinate(toChain),
						uint32(0),
						nonce,
						sender,
						addr,
						uint32(id+uint32(index)),
						big.NewInt(10),
						big.NewInt(1),
						uint32(time.Now().Unix()),
					)
					sig, _ := privateKey.Sign(tx.SignHash().Bytes())
					tx.WithSignature(sig)
					txChan <- tx
				}
			}(key, id)
		}
		id++
	}
}

func generateIssueTx() {
	var (
		fromChain = []byte{0}
		toChain   = []byte{0}
	)
	nonce := getNonce()
	issueKey, _ := crypto.HexToECDSA(issuePriKeyHex)
	sender := accounts.PublicKeyToAddress(*issueKey.Public())

	for i := 0; i < 10; i++ {
		privateKey, _ := crypto.GenerateKey()
		list <- privateKey
		addr := accounts.PublicKeyToAddress(*privateKey.Public())
		tx := types.NewTransaction(
			coordinate.NewChainCoordinate(fromChain),
			coordinate.NewChainCoordinate(toChain),
			uint32(5),
			nonce,
			sender,
			addr,
			uint32(i+index),
			big.NewInt(10e11),
			big.NewInt(1),
			uint32(time.Now().Unix()),
		)
		issueCoin := make(map[string]interface{})
		issueCoin["id"] = i + index
		tx.Payload, _ = json.Marshal(issueCoin)
		sig, _ := issueKey.Sign(tx.SignHash().Bytes())
		tx.WithSignature(sig)
		txChan <- tx
		nonce = nonce + 1
	}
}

func getNonce() uint32 {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 5,
		},
	}

	issueKey, _ := crypto.HexToECDSA(issuePriKeyHex)
	address := accounts.PublicKeyToAddress(*issueKey.Public()).String()
	req, err := http.NewRequest("POST", "http://127.0.0.1:8881", bytes.NewBufferString(
		`{"id":1,"method":"Ledger.GetBalance","params":["`+address+`"]}`,
	))
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		panic(fmt.Errorf("Couldn't parse response body. %+v", err))
	}
	var dat map[string]interface{}
	json.Unmarshal(body, &dat)
	bn := dat["result"].(map[string]interface{})
	nonceStart := bn["Nonce"].(float64)
	return uint32(nonceStart + 1)
}
