package jsvm

import (
	"testing"
	"github.com/bocheninc/L0/vm"
	"time"
	"fmt"
	"github.com/pborman/uuid"
	"math/rand"
	"strconv"
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
	workCnt := 2
	luaWorkers := make([]vm.VmWorker, workCnt)
	for i:=0; i<workCnt; i++ {
		luaWorkers[i] = NewJsWorker(vm.DefaultConfig())
	}

	luaVm := AddNewEnv("js", luaWorkers)

	time.Sleep(time.Second)
	cnt := 1
	fn := func(data interface{}) interface{} {
		//fmt.Println(data)
		cnt ++
		return true
	}

	l0Handler := NewL0Handler()

	initccd := func() *vm.WorkerProc {
		uid := uuid.New()
		amount := strconv.Itoa(rand.Intn(1000))
		workerProc := &vm.WorkerProc{
			ContractData: CreateContractData([]string{uid, amount, uid}),
			PreMethod: "RealInitContract",
			L0Handler: l0Handler,
		}

		return workerProc
	}

	invokeccd := func() *vm.WorkerProc {
		uid := uuid.New()
		amount := strconv.Itoa(rand.Intn(1000))
		workerProc := &vm.WorkerProc{
			ContractData: CreateContractData([]string{"send", uid, amount, uid}),
			PreMethod: "RealInvokeExecute",
			L0Handler: l0Handler,
		}

		return workerProc
	}

	luaVm.SendWorkCleanAsync(&vm.WorkerProcWithCallback{
		WorkProc: initccd(),
		Fn:fn,
	})

	time.Sleep(time.Second)
	startTime := time.Now()
	for i := 0; i<8; i++ {
		invokeData := invokeccd()
		//fmt.Println("contract code: ", len(invokeData.ContractData.ContractCode))
		luaVm.SendWorkCleanAsync(&vm.WorkerProcWithCallback{
			WorkProc: invokeData,
			Fn:fn,
		})
	}


	fmt.Println("WorkThread: ",workCnt, " Exec time: ", time.Now().Sub(startTime))
	luaVm.SendWorkCleanAsync(&vm.WorkerProcWithCallback{
		WorkProc: initccd(),
		Fn:fn,
	})
	time.Sleep(2 * time.Second)
	luaVm.Close("lua")
	fmt.Println("cnt: ", cnt)
}