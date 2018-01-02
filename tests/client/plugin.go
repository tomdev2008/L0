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

	"github.com/bocheninc/L0/core/types"
)

type accountData struct {
	FromUser string `json:"from_user"`
	ToUser   string `json:"to_user"`
	Address  string `json:"addr"`
	UID      string `json:"uid"`
	Frozened bool   `json:"frozened"`
}

type metaData struct {
	Account accountData `json:"account"`
}

/*
Verify:
部署即生效的安全合约。

安全合约对交易做如下验证：
- 验证转出地址和转出账户是否对应
- 验证转入地址和转入账户是否对应
- 验证转出地址是否可转出
- 验证转入地址是否可转入
- 验证转出金额是否超出单笔转账限额
- 验证转出金额是否超出单日转出限额

build:
go build -buildmode=plugin -o plugin.so plugin.go
*/
func Verify(tx *types.Transaction, getter func(key string) ([]byte, error)) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	if tx.GetType() == types.TypeIssue ||
		tx.GetType() == types.TypeIssueUpdate ||
		tx.GetType() == types.TypeJSContractInit ||
		tx.GetType() == types.TypeLuaContractInit ||
		tx.GetType() == types.TypeContractInvoke ||
		tx.GetType() == types.TypeSecurity {
		return nil
	}

	if getter == nil {
		return fmt.Errorf("global state getter is nil")
	}

	var mtData metaData
	err := json.Unmarshal(tx.Meta, &mtData)
	if err != nil {
		return err
	}
	//fmt.Println("mtData: ", string(tx.Meta), mtData)

	// sender
	if len(mtData.Account.FromUser) == 0 {
		return fmt.Errorf("invalid sender account")
	}

	dbData, err := getter("account." + mtData.Account.FromUser)
	if err != nil {
		return err
	}
	//fmt.Println("account fromuser: "+mtData.Account.FromUser, " result :", string(dbData), err)

	if len(dbData) == 0 {
		return fmt.Errorf("can't find sender account data")
	}

	var accData accountData
	err = json.Unmarshal(dbData, &accData)
	if err != nil {
		return err
	}
	//fmt.Println("from accData: ", string(dbData), accData)
	if tx.Sender().String() != accData.Address {
		return fmt.Errorf("sender address does not match, %s vs %s",
			tx.Sender().String(), accData.Address)
	}

	if accData.Frozened {
		return fmt.Errorf("sender account is frozened")
	}

	// recipient
	// if len(mtData.Account.ToUser) == 0 {
	// 	return fmt.Errorf("invalid recipient account")
	// }

	// dbData, err = getter("account." + mtData.Account.ToUser)
	// if err != nil {
	// 	return err
	// }

	// //fmt.Println("account touser :"+mtData.Account.ToUser, " result :", string(dbData), err)

	// if len(dbData) == 0 {
	// 	return fmt.Errorf("can't find recipient account data")
	// }

	// err = json.Unmarshal(dbData, &accData)
	// if err != nil {
	// 	return err
	// }
	// //fmt.Println("to accData: ", string(dbData), accData)

	// if tx.Recipient().String() != accData.Address {
	// 	return fmt.Errorf("recipient address does not match, %s vs %s",
	// 		tx.Recipient().String(), accData.Address)
	// }

	// if accData.Frozened {
	// 	return fmt.Errorf("recipient account is frozened")
	// }

	// amount
	// amount := tx.Amount()

	// limitData, err := getter("singleTransactionLimit")
	// if err != nil {
	// 	return err
	// }

	// //fmt.Println("limitData: ", string(limitData), err)

	// if len(limitData) > 0 {
	// 	limitAmount := big.NewInt(0).SetBytes(limitData)
	// 	if amount.Cmp(limitAmount) > 0 {
	// 		return fmt.Errorf("amount is more than single limit of %d", limitAmount.Int64())
	// 	}
	// }

	// limitData, err = getter("dailyTransactionLimit")
	// if err != nil {
	// 	return err
	// }

	// if len(limitData) > 0 {
	// 	limitAmount := big.NewInt(0).SetBytes(limitData)
	// 	if amount.Cmp(limitAmount) > 0 {
	// 		return fmt.Errorf("amount is more than daily limit of %d", limitAmount.Int64())
	// 	}
	// }

	return nil
}
