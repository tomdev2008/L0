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

package state_hash

import (
	"bytes"

	"github.com/bocheninc/L0/components/crypto"
)

// stateCacheUnit
type stateCacheUnit struct {
	key    []byte
	value  []byte
	sum    crypto.Hash
	latest bool
}

func newStateCacheUnit(key, value []byte) *stateCacheUnit {
	return &stateCacheUnit{
		key:   key,
		value: value,
	}
}

func (u *stateCacheUnit) set(value []byte) {
	if !bytes.Equal(value, u.value) {
		u.value = value
		u.latest = false
	}
}

func (u *stateCacheUnit) hash() []byte {
	if u.latest {
		return u.sum[:]
	}

	var data []byte
	data = append(data, u.key...)
	data = append(data, u.value...)
	u.sum = crypto.Sha256(data)
	u.latest = true
	return u.sum[:]
}
