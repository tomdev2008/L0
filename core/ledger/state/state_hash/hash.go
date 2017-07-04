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
	"sync"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/log"
)

var (
	once  sync.Once
	cache *hashCache
)

// ForeacherHandler handle iterating.
type ForeacherHandler func([]byte, []byte)

// Foreacher iterates.
type Foreacher func(ForeacherHandler)

// InitHash returns stateRootHash
func InitHash(foreacher Foreacher) crypto.Hash {
	defer logTimeCostF("state_hash.InitHash")()
	if cache == nil {
		initStateCache(foreacher)
	}
	return cache.hash()
}

// UpdateHash updates state cache and returns the new stateRootHash.
func UpdateHash(writeBatchs []*db.WriteBatch) crypto.Hash {
	if cache == nil {
		panic("must init first!")
	}

	defer logTimeCostF("state_hash.UpdateHash(count=%d)", len(writeBatchs))()
	for _, writeBatch := range writeBatchs {
		switch writeBatch.Operation {
		case db.OperationPut:
			cache.set(writeBatch.Key, writeBatch.Value)
		case db.OperationDelete:
			cache.remove(writeBatch.Key)
		}
	}
	defer logTimeCostF("state_hash.UpdateHash, make hash")()
	return cache.hash()
}

func initStateCache(foreacher Foreacher) {
	if foreacher == nil {
		panic("invalid foreacher.")
	}

	once.Do(func() {
		defer logTimeCostF("state_hash.initStateCache")()
		cache = newHashCache()

		k := 0
		foreacher(func(key, value []byte) {
			cache.set(key, value)
			k++
		})
		log.Infof("state_hash.initStateCache, count: %d, leafNumber: %d", k, cache.leafNumber())
	})
}
