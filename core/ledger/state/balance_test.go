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
	"math/big"
	"testing"
)

func TestBalance(t *testing.T) {
	b := NewBalance()
	if b.Get(0).Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("expect 0 == 0")
	}

	b.Set(0, big.NewInt(int64(100)))
	if b.Get(0).Cmp(big.NewInt(int64(100))) != 0 {
		t.Fatalf("expect 100 == 100")
	}

	if b.Add(0, big.NewInt(int64(100))).Cmp(big.NewInt(int64(200))) != 0 {
		t.Fatalf("expect 200 == 200")
	}

	if b.Get(0).Cmp(big.NewInt(int64(200))) != 0 {
		t.Fatalf("expect 200 == 200")
	}

	if b.Add(0, big.NewInt(int64(-1000))).Cmp(big.NewInt(int64(-800))) != 0 {
		t.Fatalf("expect -800 == -800")
	}

	if b.Get(0).Cmp(big.NewInt(int64(-800))) != 0 {
		t.Fatalf("expect -800 == -800")
	}

}

func TestSerializeAndDeserialize(t *testing.T) {
	b := NewBalance()
	b.Set(0, big.NewInt(int64(-100)))
	b.Set(1, big.NewInt(int64(200)))
	b.Set(3, big.NewInt(int64(300)))
	balanceBytes := b.serialize()
	t.Log(balanceBytes)

	tb := NewBalance()
	tb.deserialize(balanceBytes)

	if tb.Get(0).Cmp(b.Get(0)) != 0 {
		t.Fatalf("expect 100 == 100")
	}

	if tb.Get(1).Cmp(b.Get(1)) != 0 {
		t.Fatalf("expect 200 == 200")
	}

	if tb.Get(3).Cmp(b.Get(3)) != 0 {
		t.Fatalf("expect 300 == 300")
	}
}
