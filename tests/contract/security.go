package main

import (
	"encoding/json"
	"fmt"
	"math/big"

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
go build -buildmode=plugin -o security.so security.go
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

	// sender
	if len(mtData.Account.FromUser) == 0 {
		return fmt.Errorf("invalid sender account")
	}

	dbData, err := getter("account." + mtData.Account.FromUser)
	if err != nil {
		return err
	}

	if len(dbData) == 0 {
		return fmt.Errorf("can't find sender account data")
	}

	var accData accountData
	err = json.Unmarshal(dbData, &accData)
	if err != nil {
		return err
	}

	if tx.Sender().String() != accData.Address {
		return fmt.Errorf("sender address does not match, %s vs %s",
			tx.Sender().String(), accData.Address)
	}

	if accData.Frozened {
		return fmt.Errorf("sender account is frozened")
	}

	// recipient
	if len(mtData.Account.ToUser) == 0 {
		return fmt.Errorf("invalid recipient account")
	}

	dbData, err = getter("account." + mtData.Account.ToUser)
	if err != nil {
		return err
	}

	if len(dbData) == 0 {
		return fmt.Errorf("can't find recipient account data")
	}

	err = json.Unmarshal(dbData, &accData)
	if err != nil {
		return err
	}

	if tx.Recipient().String() != accData.Address {
		return fmt.Errorf("recipient address does not match, %s vs %s",
			tx.Recipient().String(), accData.Address)
	}

	if accData.Frozened {
		return fmt.Errorf("recipient account is frozened")
	}

	// amount
	amount := tx.Amount()

	limitData, err := getter("singleTransactionLimit")
	if err != nil {
		return err
	}

	if len(limitData) > 0 {
		limitAmount := big.NewInt(0).SetBytes(limitData)
		if amount.Cmp(limitAmount) > 0 {
			return fmt.Errorf("amount is more than single limit of %d", limitAmount.Int64())
		}
	}

	limitData, err = getter("dailyTransactionLimit")
	if err != nil {
		return err
	}

	if len(limitData) > 0 {
		limitAmount := big.NewInt(0).SetBytes(limitData)
		if amount.Cmp(limitAmount) > 0 {
			return fmt.Errorf("amount is more than daily limit of %d", limitAmount.Int64())
		}
	}

	return nil
}
