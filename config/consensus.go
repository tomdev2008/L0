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
	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/consensus/consenter"
	"github.com/bocheninc/L0/core/consensus/lbft"
	"github.com/bocheninc/L0/core/consensus/noops"
)

func ConsenterOptions() *consenter.Options {
	option := consenter.NewDefaultOptions()
	option.Plugin = getString("consensus.plugin", option.Plugin)
	option.Noops = NoopsOptions()
	option.Lbft = LbftOptions()
	return option
}

func NoopsOptions() *noops.Options {
	option := noops.NewDefaultOptions()
	option.BatchSize = getInt("consensus.noops.batchSize", option.BatchSize)
	option.BatchTimeout = getDuration("consensus.noops.batchTimeout", option.BatchTimeout)
	option.BlockSize = getInt("consensus.noops.blockSize", option.BlockSize)
	option.BlockTimeout = getDuration("consensus.noops.blockTimeout", option.BlockTimeout)
	return option
}

func LbftOptions() *lbft.Options {
	option := lbft.NewDefaultOptions()
	option.Chain = getString("blockchain.chainId", option.Chain)
	option.ID = option.Chain + ":" + utils.BytesToHex(crypto.Ripemd160([]byte(getString("blockchain.nodeId", option.ID)+option.Chain)))
	option.N = getInt("consensus.lbft.N", option.N)
	option.Q = getInt("consensus.lbft.Q", option.Q)
	option.K = getInt("consensus.lbft.K", option.K)
	option.BatchSize = getInt("consensus.lbft.batchSize", option.BatchSize)
	option.BatchTimeout = getDuration("consensus.lbft.batchTimeout", option.BatchTimeout)
	option.BlockSize = getInt("consensus.lbft.blockSize", option.BlockSize)
	option.BlockTimeout = getDuration("consensus.lbft.blockTimeout", option.BlockTimeout)
	option.Request = getDuration("consensus.lbft.request", option.Request)
	option.ViewChange = getDuration("consensus.lbft.viewChange", option.ViewChange)
	option.ResendViewChange = getDuration("consensus.lbft.resendViewChange", option.ViewChange)
	option.ViewChangePeriod = getDuration("consensus.lbft.viewChangePeriod", option.ViewChangePeriod)
	return option
}
