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

package config

import (
	"github.com/bocheninc/L0/core/blockchain/validator"
	"github.com/spf13/viper"
)

func ValidatorConfig(pluginDir string) *validator.Config {
	var config = validator.DefaultConfig()

	config.IsValid = getbool("validator.status", config.IsValid)
	config.BlacklistDur = getDuration("validator.blacklisttimeout", config.BlacklistDur)
	config.TxPoolCapacity = getInt("validator.txpool.capacity", config.TxPoolCapacity)
	config.TxPoolTimeOut = getDuration("validator.txpool.timeout", config.TxPoolTimeOut)
	if value := viper.GetInt("validator.txpool.txdelay"); value >= 0 {
		config.TxPoolDelay = value
	}
	config.SecurityPluginDir = pluginDir
	return config
}
