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

package validator

import (
	"math/big"

	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/ledger"
	"github.com/bocheninc/L0/core/types"
)

type account struct {
	balance *big.Int
	nonce   uint32
}

func newAccount(address accounts.Address, leger *ledger.Ledger) *account {
	amount, nonce, _ := leger.GetBalance(address)
	return &account{
		balance: amount,
		nonce:   nonce,
	}
}

func (a *account) updateTransactionReceiverBalance(tx *types.Transaction) {
	a.balance = a.balance.Add(a.balance, tx.Amount())
}

func (a *account) updateTransactionSenderBalance(tx *types.Transaction) {
	a.balance = a.balance.Sub(a.balance, tx.Amount())
}
