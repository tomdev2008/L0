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

import "time"

type Config struct {
	IsValid        bool
	TxPoolCapacity int
	TxPoolTimeOut  time.Duration
	BlacklistDur   time.Duration
}

var config *Config

func DefaultConfig() *Config {
	return &Config{
		IsValid:        true,
		TxPoolCapacity: 200000,
		TxPoolTimeOut:  10 * time.Minute,
		BlacklistDur:   1 * time.Minute,
	}
}
