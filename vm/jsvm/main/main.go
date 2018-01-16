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

package main

import (
	"os"
	"syscall"

	"github.com/bocheninc/L0/vm"
	"github.com/bocheninc/L0/vm/jsvm"
	"github.com/bocheninc/base/log"
)

func main() {

	vm.VMConf = vm.DefaultConfig()
	vm.VMConf.SetString(os.Args[1])

	log.New(vm.VMConf.LogFile)
	log.SetLevel(vm.VMConf.LogLevel)

	if err := vm.CheckVmMem(vm.VMConf.VMMaxMem); err != nil {
		log.Warning(err)
	}
	var rlimit syscall.Rlimit
	rlimit.Max = uint64(vm.VMConf.VMMaxMem) * 1024 * 1024
	rlimit.Cur = uint64(rlimit.Max / 5 * 4)
	err := syscall.Setrlimit(syscall.RLIMIT_AS, &rlimit)
	if err != nil {
		log.Error("set rlimit error", err)
		return
	}

	err = jsvm.Start("js")
	if err != nil {
		log.Error("jsvm start error", err)
	} else {
		log.Info("jsvm start success!")
	}
	select {}
}
