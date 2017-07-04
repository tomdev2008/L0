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
	"github.com/bocheninc/L0/components/log"
)

// stateCacheNode
type stateCacheNode struct {
	level    int
	index    int
	parent   *stateCacheNode
	children []*stateCacheNode
	sum      crypto.Hash
	latest   bool
	units    *stateCacheUnits
}

func newStateCacheNode(index int, parent *stateCacheNode) *stateCacheNode {
	level := 1
	if parent != nil {
		level = parent.level + 1
	}

	return &stateCacheNode{
		level:    level,
		index:    index,
		parent:   parent,
		children: make([]*stateCacheNode, cacheBranchNumber),
	}
}

func (n *stateCacheNode) valid() bool {
	if n == nil {
		return false
	}

	if !n.isLeaf() {
		for _, c := range n.children {
			if c.valid() {
				return true
			}
		}
		return false
	}
	return (n.units != nil) && (n.units.len() > 0)
}

func (n *stateCacheNode) addLeaf(unit *stateCacheUnit, localBucketID int) *stateCacheNode {
	if unit == nil {
		return nil
	}

	if n.isLeaf() {
		if n.units == nil {
			n.units = newStateCacheUnits()
		}
		n.units.set(unit)
		n.latest = false
		return n
	}

	index, localBucketID := nodeIndexByLevel(n.level+1, localBucketID)
	if index >= cacheBranchNumber {
		log.Errorf("stateCacheNode.addLeaf, invalid index: %d, branch number: %d", index, cacheBranchNumber)
		return nil
	}

	c := n.children[index]
	if c == nil {
		c = newStateCacheNode(index, n)
		n.children[index] = c
	}
	n.latest = false
	return c.addLeaf(unit, localBucketID)
}

func (n *stateCacheNode) set(unit *stateCacheUnit) {
	if unit == nil {
		return
	}

	if !n.isLeaf() {
		return
	}

	if n.units == nil {
		n.units = newStateCacheUnits()
	}
	n.units.set(unit)
	n.dirty()
	return
}

func (n *stateCacheNode) remove(key []byte) {
	if len(key) == 0 {
		return
	}

	if !n.isLeaf() {
		return
	}

	if n.units != nil {
		n.units.remove(key)
		if n.valid() {
			n.dirty()
		} else {
			n.removeFromParent()
		}
	} else {
		n.removeFromParent()
	}
}

func (n *stateCacheNode) removeChild(index int) {
	if index < 0 || len(n.children) <= index {
		return
	}

	if n.children[index] == nil {
		return
	}

	n.children[index] = nil
	if n.valid() {
		n.dirty()
	} else {
		n.removeFromParent()
	}
}

func (n *stateCacheNode) removeFromParent() {
	if n.parent != nil {
		n.parent.removeChild(n.index)
	}
}

func (n *stateCacheNode) hash() []byte {
	if n.latest {
		return n.sum[:]
	}

	if !n.isLeaf() {
		var data []byte
		for _, c := range n.children {
			if c != nil {
				data = append(data, c.hash()...)
			}
		}

		if len(data) == crypto.HashSize {
			copy(n.sum[:], data)
		} else {
			n.sum = crypto.Sha256(data)
		}
	} else {
		copy(n.sum[:], n.units.hash())
	}
	n.latest = true
	return n.sum[:]
}

func (n *stateCacheNode) dirty() {
	if n.latest {
		n.latest = false
		if n.parent != nil {
			n.parent.dirty()
		}
	}
}

func (n *stateCacheNode) isLeaf() bool {
	return int(n.level) >= cacheLevelNumber
}

func (n *stateCacheNode) childrenNumber() int {
	if !n.isLeaf() {
		count := 0
		for _, c := range n.children {
			if c.valid() {
				count++
			}
		}
		return count
	}

	if n.units != nil {
		return n.units.len()
	}
	return 0
}
