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
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

func main() {
	for i := 0; i < 10; i++ {
		time.Sleep(10)
		go func() {
			for j := 0; j < 5; j++ {
				HttpSend(generateAtomicTx())
			}
		}()
	}

	time.Sleep(time.Minute * 3)
}

func HttpSend(param string) {
	paramStr := `{"id":1,"chainId":"00","method":"Transaction.Broadcast","params":["` + param + `"]}`
	req, err := http.NewRequest("POST", "http://127.0.0.1:8989", bytes.NewBufferString(paramStr))
	// paramStr := `{"id":1,"method":"Transaction.Broadcast","params":["` + param + `"]}`
	// req, err := http.NewRequest("POST", "http://127.0.0.1:8881", bytes.NewBufferString(paramStr))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	var client = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 1000,
		},
		Timeout: time.Duration(60) * time.Second,
	}
	t := time.Now()
	response, err := client.Do(req)

	if err == nil {
		defer response.Body.Close()
		body, er := ioutil.ReadAll(response.Body)
		if er != nil {
			log.Print("couldn't parse response body. ", err)
		}
		log.Print(time.Now().Sub(t), string(body))
	}
}

func generateAtomicTx() string {
	issuePriKeyHex := "496c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
	privateKey, _ := crypto.HexToECDSA(issuePriKeyHex)
	addr := accounts.HexToAddress("4ce1bb0858e71b50d603ebe4bec95b11d8833e6d")
	sender := accounts.PublicKeyToAddress(*privateKey.Public())
	tx := types.NewTransaction(
		coordinate.HexToChainCoordinate("00"),
		coordinate.HexToChainCoordinate("00"),
		uint32(5),
		0,
		sender,
		addr,
		big.NewInt(1000),
		big.NewInt(1),
		uint32(time.Now().Nanosecond()),
	)
	sig, _ := privateKey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	return hex.EncodeToString(tx.Serialize())
}
