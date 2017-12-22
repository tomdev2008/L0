package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"bytes"
	"io/ioutil"
	"time"
	"encoding/hex"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/components/utils"
)

var DefaultURL = "http://127.0.0.1:8881"
var DefaultClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 1000, //MaxIdleConnections,
	},
	Timeout: time.Duration(200) * time.Second, // RequestTimeout
}

// GetTxByHash get tx information by txHash from Ledger
func GetTxByHash(txHash string) (*ResponseData, error) {
	return send(fmt.Sprintf(`{"id": 1, "method": "Ledger.GetTxByHash", "params":["%s"]}`, txHash), DefaultURL, DefaultClient)
}

//GetBalance get balance information by address from Ledger
func GetBalance(address string) (*ResponseData, error) {
	return send(fmt.Sprintf(`{"id":1,"method":"Ledger.GetBalance","params":["%s"]}`, address), DefaultURL, DefaultClient)
}

//GetAsset get asset information by id from Ledger
func GetAsset(id uint32) (*ResponseData, error) {
	return send(fmt.Sprintf(`{"id":1,"method":"Ledger.GetAsset","params":[%d]}`, id), DefaultURL, DefaultClient)
}

func Broadcast(tx *types.Transaction, url string, cli *http.Client) (*ResponseData, error) {
	txData := utils.Serialize(tx)
	fmt.Println("tx_hash: ", tx.Hash(), " tx_type: ", tx.GetType())

	return SendTransaction(fmt.Sprintf(`{"id":1,"method":"Transaction.Broadcast","params":["%s"]}`, hex.EncodeToString(txData)), url, cli)
}

func SendTransaction(data, url string, cli *http.Client) (*ResponseData, error) {
	return send(data, url, cli)
}

func send(data, url string, client *http.Client) (*ResponseData, error) {
	//respData := &ResponseData{Error: fmt.Sprintf("can't connect to host: %s", url)}
	var respData ResponseData
	req, _ := http.NewRequest("POST", url, bytes.NewBufferString(data))
	req.Header.Set("Content-Type", "application/json")
	if response, err := client.Do(req); err == nil {
		defer response.Body.Close()
		if body, err := ioutil.ReadAll(response.Body); err == nil {
			err = json.Unmarshal(body, &respData)
			if err != nil {
				return &respData, err
			}
		}
	} else {
		respData.Error = fmt.Sprintf("can't connect to host: %s", url)
		return &respData, err
	}

	return &respData, nil
}
