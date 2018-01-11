package luavm

import (
	"testing"
	"github.com/bocheninc/L0/vm"
	"time"
	"fmt"
	"github.com/pborman/uuid"
	"math/rand"
	"strconv"
	"github.com/bocheninc/L0/components/log"
)

var VMEnv = make(map[string]*vm.VirtualMachine)
func AddNewEnv(name string, worker []vm.VmWorker) *vm.VirtualMachine {
	env := vm.CreateCustomVM(worker)
	env.Open(name)
	VMEnv[name] = env

	return env
}


func TestVMFunction(t *testing.T) {
	vm.VMConf = vm.DefaultConfig()
	workCnt := 1
	luaWorkers := make([]vm.VmWorker, workCnt)
	for i:=0; i<workCnt; i++ {
		luaWorkers[i] = NewLuaWorker(vm.DefaultConfig())
	}

	luaVm := AddNewEnv("lua", luaWorkers)

	cnt := 1
	fn := func(data interface{}) interface{} {
		//fmt.Println(data)
		cnt ++
		return nil
	}

	l0Handler := NewL0Handler()

	initccd := func() *vm.WorkerProc {
		uid := uuid.New()
		amount := strconv.Itoa(rand.Intn(1000))
		workerProc := &vm.WorkerProc{
			ContractData: CreateContractData([]string{uid, amount, uid}),
			L0Handler: l0Handler,
		}

		return workerProc
	}


	invokeccd := func() *vm.WorkerProc {
		uid := uuid.New()
		amount := strconv.Itoa(rand.Intn(1000))
		workerProc := &vm.WorkerProc{
			ContractData: CreateContractData([]string{"send", uid, amount, uid}),
			L0Handler: l0Handler,
		}

		return workerProc
	}

	time.Sleep(time.Second)
	luaVm.SendWorkCleanAsync(&vm.WorkerProcWithCallback{
		WorkProc: initccd(),
		Fn:fn,
	})
	time.Sleep(time.Second)
	log.Info("==============start=================")
	startTime := time.Now()
	for i := 0; i<8; i++ {
		luaVm.SendWorkCleanAsync(&vm.WorkerProcWithCallback{
			WorkProc: invokeccd(),
			Fn:fn,
		})
	}
	fmt.Println("WorkThread: ",workCnt, " Exec time: ", time.Now().Sub(startTime))
	log.Info("==============end=================")

	luaVm.SendWorkCleanAsync(&vm.WorkerProcWithCallback{
		WorkProc: initccd(),
		Fn:fn,
	})
	time.Sleep(5 * time.Second)
	luaVm.Close("lua")
	fmt.Println("cnt: ", cnt)
}
