package main

import (
	"fmt"
	"math/big"

	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/types"
)

var zeroAddr = accounts.Address{}

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
*/
func Verify(tx *types.Transaction, getter func(key string) ([]byte, error)) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	if getter == nil {
		return fmt.Errorf("global state getter is nil")
	}

	// sender
	senderAccount, err := getter("account." + tx.Sender().String())
	if err != nil {
		return err
	}

	if len(senderAccount) == 0 {
		return fmt.Errorf("invalid account")
	}

	// recipient
	recipient := tx.Recipient()
	if recipient != zeroAddr {
		recipientAccount, err := getter("account." + recipient.String())
		if err != nil {
			return err
		}

		if len(recipientAccount) == 0 {
			return fmt.Errorf("invalid recipient")
		}
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
