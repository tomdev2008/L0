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
	"sort"

	"github.com/bocheninc/L0/components/crypto"
)

// stateCacheUnits
type stateCacheUnits struct {
	units  []*stateCacheUnit
	sum    crypto.Hash
	latest bool
}

func newStateCacheUnits() *stateCacheUnits {
	return &stateCacheUnits{}
}

func (us *stateCacheUnits) set(unit *stateCacheUnit) {
	if unit == nil {
		return
	}

	for _, u := range us.units {
		if bytes.Equal(u.key, unit.key) {
			u.set(unit.value)
			us.latest = false
			return
		}
	}
	us.units = append(us.units, unit)
	us.latest = false
}

func (us *stateCacheUnits) remove(key []byte) {
	if len(key) == 0 {
		return
	}

	for i, u := range us.units {
		if bytes.Compare(u.key, key) == 0 {
			us.units = append(us.units[:i], us.units[i+1:]...)
			us.latest = false
			return
		}
	}
}

func (us *stateCacheUnits) len() int {
	return len(us.units)
}

func (us *stateCacheUnits) hash() []byte {
	if us.latest {
		return us.sum[:]
	}

	if us.len() == 0 {
		return nil
	}

	if len(us.units) == 1 {
		copy(us.sum[:], us.units[0].hash())
		us.latest = true
		return us.sum[:]
	}

	us.sort()
	var data []byte
	for _, u := range us.units {
		data = append(data, u.hash()...)
	}
	us.sum = crypto.Sha256(data)
	us.latest = true
	return us.sum[:]
}

func (us *stateCacheUnits) sort() {
	sort.Slice(us.units, func(i, j int) bool {
		return bytes.Compare(us.units[i].key, us.units[j].key) < 0
	})
}
