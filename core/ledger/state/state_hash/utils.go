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
	"fmt"
	"hash/fnv"
	"math"
	"time"

	"github.com/bocheninc/L0/components/log"
)

var (
	levelPowValues map[int]int
)

func init() {
	levelPowValues = make(map[int]int)
	for l := 1; l <= cacheLevelNumber; l++ {
		levelPowValues[l] = int(math.Pow(float64(cacheBranchNumber), float64(cacheLevelNumber-l)))
	}
}

func convertBucketID(key []byte) int {
	fnvHash := fnv.New32a()
	fnvHash.Write(key)
	sum := int(fnvHash.Sum32())
	return sum % cacheBucketNumber
}

func nodeIndexByLevel(level, localBucketID int) (int, int) {
	if factor, ok := levelPowValues[level]; ok {
		return localBucketID / factor, localBucketID % factor
	}
	return 0, 0
}

func logTimeCostF(format string, args ...interface{}) func() {
	start := time.Now()
	return func() {
		log.Infof("%s, time-cost: %s", fmt.Sprintf(format, args...), time.Since(start))
	}
}
