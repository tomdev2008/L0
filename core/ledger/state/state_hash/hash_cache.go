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
	"github.com/bocheninc/L0/components/crypto"
)

// hashCache
type hashCache struct {
	rootNode *stateCacheNode
	leafs    map[int]*stateCacheNode
}

func newHashCache() *hashCache {
	return &hashCache{
		rootNode: newStateCacheNode(0, nil),
		leafs:    make(map[int]*stateCacheNode, cacheBucketNumber),
	}
}

func (c *hashCache) set(key, value []byte) {
	if len(key) == 0 {
		return
	}

	bucketID := convertBucketID(key)
	leaf := c.leafs[bucketID]
	if leaf != nil {
		leaf.set(newStateCacheUnit(key, value))
	} else {
		leaf = c.rootNode.addLeaf(newStateCacheUnit(key, value), bucketID)
		if leaf != nil {
			c.leafs[bucketID] = leaf
		}
	}
}

func (c *hashCache) remove(key []byte) {
	if len(key) == 0 {
		return
	}

	bucketID := convertBucketID(key)
	leaf := c.leafs[bucketID]
	if leaf != nil {
		leaf.remove(key)
		if !leaf.valid() {
			delete(c.leafs, bucketID)
		}
	}
}

func (c *hashCache) hash() crypto.Hash {
	return crypto.NewHash(c.rootNode.hash())
}

func (c *hashCache) leafNumber() int {
	return len(c.leafs)
}
