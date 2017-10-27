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

package state

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/bocheninc/L0/components/utils"
)

// NewBalance Create an balance object
func NewBalance() *Balance {
	return &Balance{
		Amounts: make(map[uint32]*big.Int),
	}
}

// Balance Contain all asset amounts and nonce
type Balance struct {
	Amounts map[uint32]*big.Int
	Nonce   uint32
	rw      sync.RWMutex
}

//Set Set amount of asset
func (b *Balance) Set(id uint32, amount *big.Int) {
	b.rw.Lock()
	defer b.rw.Unlock()
	b.Amounts[id] = amount
}

//Get Get amount of asset
func (b *Balance) Get(id uint32) *big.Int {
	//b.RLock()
	//defer b.RUnlock()
	if amount, ok := b.Amounts[id]; ok {
		return amount
	}
	return big.NewInt(0)
}

//Add Set amount of asset to  sum +y and return.
func (b *Balance) Add(id uint32, amount *big.Int) *big.Int {
	b.rw.Lock()
	defer b.rw.Unlock()
	tamount, ok := b.Amounts[id]
	if !ok {
		tamount = big.NewInt(0)
		b.Amounts[id] = tamount
	}
	tamount.Add(tamount, amount)
	return tamount
}

func (b *Balance) serialize() []byte {
	return utils.Serialize(b)
}

func (b *Balance) deserialize(balanceBytes []byte) {
	if err := utils.Deserialize(balanceBytes, b); err != nil {
		panic(fmt.Errorf("balance deserialize error: %s", err))
	}
}
