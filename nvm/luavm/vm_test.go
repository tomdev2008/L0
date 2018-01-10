package luavm

import (
	"testing"
	"github.com/bocheninc/L0/nvm"
	"time"
	"fmt"
	"github.com/pborman/uuid"
	"math/rand"
	"strconv"
	"github.com/bocheninc/L0/components/log"
)

var VMEnv = make(map[string]*nvm.VirtualMachine)
func AddNewEnv(name string, worker []nvm.VmWorker) *nvm.VirtualMachine {
	env := nvm.CreateCustomVM(worker)
	env.Open(name)
	VMEnv[name] = env

	return env
}


func TestVMFunction(t *testing.T) {
	nvm.VMConf = nvm.DefaultConfig()
	workCnt := 1
	luaWorkers := make([]nvm.VmWorker, workCnt)
	for i:=0; i<workCnt; i++ {
		luaWorkers[i] = NewLuaWorker(nvm.DefaultConfig())
	}

	luaVm := AddNewEnv("lua", luaWorkers)

	cnt := 1
	fn := func(data interface{}) interface{} {
		//fmt.Println(data)
		cnt ++
		return nil
	}

	l0Handler := NewL0Handler()

	initccd := func() *nvm.WorkerProc {
		uid := uuid.New()
		amount := strconv.Itoa(rand.Intn(1000))
		workerProc := &nvm.WorkerProc{
			ContractData: CreateContractData([]string{uid, amount, uid}),
			PreMethod: "RealInitContract",
			L0Handler: l0Handler,
		}

		return workerProc
	}


	invokeccd := func() *nvm.WorkerProc {
		uid := uuid.New()
		amount := strconv.Itoa(rand.Intn(1000))
		workerProc := &nvm.WorkerProc{
			ContractData: CreateContractData([]string{"send", uid, amount, uid}),
			PreMethod: "RealInvokeExecute",
			L0Handler: l0Handler,
		}

		return workerProc
	}

	time.Sleep(time.Second)
	luaVm.SendWorkCleanAsync(&nvm.WorkerProcWithCallback{
		WorkProc: initccd(),
		Fn:fn,
	})
	time.Sleep(time.Second)
	log.Info("==============start=================")
	startTime := time.Now()
	for i := 0; i<8; i++ {
		luaVm.SendWorkCleanAsync(&nvm.WorkerProcWithCallback{
			WorkProc: invokeccd(),
			Fn:fn,
		})
	}
	fmt.Println("WorkThread: ",workCnt, " Exec time: ", time.Now().Sub(startTime))
	log.Info("==============end=================")

	luaVm.SendWorkCleanAsync(&nvm.WorkerProcWithCallback{
		WorkProc: initccd(),
		Fn:fn,
	})
	time.Sleep(5 * time.Second)
	luaVm.Close("lua")
	fmt.Println("cnt: ", cnt)
}
