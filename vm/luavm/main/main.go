/*
	Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.

	This file is part of L0

	The L0 is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	The L0 is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"fmt"
	"syscall"
	"time"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/vm"
	"github.com/bocheninc/L0/vm/luavm"
)

func main() {
	var rlimit syscall.Rlimit
	rlimit.Cur = 1024 //以字节为单位
	rlimit.Max = uint64(vm.VMConf.VMMaxMem) * 1024 * 1024
	err := syscall.Setrlimit(syscall.RLIMIT_AS, &rlimit)
	if err != nil {
		fmt.Println("set rlimit error", err)
		return
	}

	// fmt.Println("in luavm main")
	err = luavm.Start()
	if err != nil {
		log.Error("luavm start error", err)
	}

	for {
		time.Sleep(time.Second * 60)
	}
}
