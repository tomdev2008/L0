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

// vm config load from vm.yaml

package vm

import "encoding/json"

var VMConf *Config

// Config vm config struct
type Config struct {
	LogFile                    string
	LogLevel                   string
	VMRegistrySize             int
	VMCallStackSize            int
	VMMaxMem                   int // vm maximum memory size (MB)
	ExecLimitStackDepth        int
	ExecLimitMaxOpcodeCount    int // maximum allow execute opcode count
	ExecLimitMaxRunTime        int // the contract maximum run time (millisecond)
	ExecLimitMaxScriptSize     int // contract script(lua source code) maximum size (byte)
	ExecLimitMaxStateValueSize int // the max state value size (byte)
	ExecLimitMaxStateItemCount int // the max state count in one contract
	ExecLimitMaxStateKeyLength int // max state key length
	LuaVMExeFilePath           string
	JSVMExeFilePath            string
}

// DefaultConfig default vm config
func DefaultConfig() *Config {
	return &Config{
		VMRegistrySize:             256,
		VMCallStackSize:            64,
		VMMaxMem:                   200,
		ExecLimitStackDepth:        100,
		ExecLimitMaxOpcodeCount:    10000,
		ExecLimitMaxRunTime:        1000,
		ExecLimitMaxScriptSize:     10240, //5K
		ExecLimitMaxStateValueSize: 10240, //5K
		ExecLimitMaxStateItemCount: 1000,
		ExecLimitMaxStateKeyLength: 256,
		LuaVMExeFilePath:           "bin/luavm",
		JSVMExeFilePath:            "bin/jsvm",
	}
}

func (c *Config) String() string {
	data, _ := json.Marshal(c)
	return string(data)
}

func (c *Config) SetString(src string) error {
	return json.Unmarshal([]byte(src), c)
}
