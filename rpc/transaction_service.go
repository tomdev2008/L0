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

package rpc

import (
	"errors"
	"math/big"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/notify"
	//"github.com/bocheninc/L0/core/notify"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/base/log"
)

type IBroadcast interface {
	Relay(inv types.IInventory)
	QueryContract(tx *types.Transaction) ([]byte, error)
}

type Transaction struct {
	pmHander IBroadcast
}

type TransactionCreateArgs struct {
	FromChain string
	ToChain   string
	Recipient string
	Nonce     uint32
	AssetID   uint32
	Amount    int64
	Fee       int64
	TxType    uint32
	PayLoad   interface{}
}

type PayLoad struct {
	ContractCode   string
	ContractAddr   string
	ContractParams []string
}

type ContractQueryArgs struct {
	ContractAddr   string
	ContractParams []string
}

func NewTransaction(pmHandler IBroadcast) *Transaction {
	return &Transaction{pmHander: pmHandler}
}

func (t *Transaction) Create(args *TransactionCreateArgs, reply *string) error {
	fromChain := coordinate.HexToChainCoordinate(args.FromChain)
	toChain := coordinate.HexToChainCoordinate(args.ToChain)
	nonce := args.Nonce
	recipient := accounts.HexToAddress(args.Recipient)
	sender := accounts.Address{}
	assetID := args.AssetID
	amount := big.NewInt(args.Amount)
	fee := big.NewInt(args.Fee)
	tx := types.NewTransaction(fromChain, toChain, args.TxType, nonce, sender, recipient, assetID, amount, fee, utils.CurrentTimestamp())

	switch tx.GetType() {
	case types.TypeJSContractInit:
		fallthrough
	case types.TypeLuaContractInit:
		fallthrough
	case types.TypeContractInvoke:
		if args.PayLoad == nil {
			return errors.New("contract transaction payload must not be nil")
		}
		contractSpec := new(types.ContractSpec)
		payLoad := args.PayLoad.(map[string]interface{})

		if contractCode, ok := payLoad["ContractCode"]; ok {
			contractSpec.ContractCode = utils.HexToBytes(contractCode.(string))
		}
		if contractAddr, ok := payLoad["ContractAddr"]; ok {
			contractSpec.ContractAddr = utils.HexToBytes(contractAddr.(string))
		}
		if contractParams, ok := payLoad["ContractParams"]; ok {
			for _, v := range contractParams.([]interface{}) {
				contractSpec.ContractParams = append(contractSpec.ContractParams, v.(string))
			}
		}
		tx.WithPayload(utils.Serialize(contractSpec))
	default:
		if args.PayLoad != nil {
			tx.WithPayload([]byte(args.PayLoad.(string)))
		}
	}
	*reply = utils.BytesToHex(tx.Serialize())
	return nil
}

type BroadcastReply struct {
	Result           *string     `json:"result"`
	ContractAddr     *string     `json:"contractAddr"`
	TransactionHash  crypto.Hash `json:"transactionHash"`
	AssetID          int         `json:"assetID"`
	SenderBalance    *big.Int    `json:"senderBalance"`
	RecipientBalance *big.Int    `json:"recipientBalance"`
}

func (t *Transaction) Broadcast(txHex string, reply *BroadcastReply) error {
	//startTime := time.Now()

	if len(txHex) < 1 {
		return errors.New("Invalid Params: len(txSerializeData) must be >0 ")
	}

	tx := new(types.Transaction)

	err := tx.Deserialize(utils.HexToBytes(txHex))
	if err != nil {
		return err
	}

	if tx.Data.Amount == nil || tx.Data.Fee == nil || tx.Data.Signature == nil {
		return errors.New("Invalid Hex")
	}

	if tx.Amount().Sign() < 0 {
		return errors.New("Invalid Amount in Tx, Amount must be >0")
	}

	if tx.Fee() == nil || tx.Fee().Sign() < 0 {
		return errors.New("Invalid Fee in Tx, Fee must bigger than 0")
	}

	_, err = tx.Verfiy()
	if err != nil {
		return errors.New("Invalid Tx, varify the signature of Tx failed")
	}

	var (
		errMsg  error
		balance *types.Balance
	)
	ch := make(chan struct{}, 1)
	notify.Register(tx.Hash(), 0, nil, nil, func(arg interface{}) {
		ch <- struct{}{}
		switch arg.(type) {
		case error:
			errMsg = arg.(error)
		case *types.Balance:
			balance = arg.(*types.Balance)
		}
	})

	t.pmHander.Relay(tx)
	<-ch
	if errMsg != nil {
		return errMsg
	}

	if tp := tx.GetType(); tp == types.TypeLuaContractInit || tp == types.TypeJSContractInit || tp == types.TypeContractInvoke || tp == types.TypeSecurity {
		contractSpec := new(types.ContractSpec)
		utils.Deserialize(tx.Payload, contractSpec)
		contractAddr := utils.BytesToHex(contractSpec.ContractAddr)
		*reply = BroadcastReply{ContractAddr: &contractAddr, TransactionHash: tx.Hash()}
	} else {
		*reply = BroadcastReply{TransactionHash: tx.Hash(), SenderBalance: balance.Sender, RecipientBalance: balance.Recipient, AssetID: int(balance.ID)}
	}
	return nil
}

//Query contract query
func (t *Transaction) Query(args *ContractQueryArgs, reply *string) error {
	var contractAddress []byte
	if len(args.ContractAddr) > 0 {
		if args.ContractAddr[0:2] == "0x" {
			args.ContractAddr = args.ContractAddr[2:]
		}

		contractAddress = utils.HexToBytes(args.ContractAddr)
		if len(contractAddress) != 20 && len(contractAddress) != 22 {
			log.Errorf("contract address[%s] is illegal", args.ContractAddr)
			return errors.New("contract address is illegal")
		}
	}

	contractSpec := new(types.ContractSpec)
	contractSpec.ContractAddr = contractAddress
	contractSpec.ContractParams = args.ContractParams
	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(params.ChainID),
		coordinate.NewChainCoordinate(params.ChainID),
		types.TypeContractQuery,
		uint32(0),
		accounts.Address{},
		accounts.NewAddress(contractSpec.ContractAddr),
		uint32(0),
		big.NewInt(0),
		big.NewInt(0),
		uint32(time.Now().Unix()),
	)
	tx.Payload = utils.Serialize(contractSpec)

	result, err := t.pmHander.QueryContract(tx)
	if err != nil {
		return err
	}

	*reply = string(result)

	return nil

}
