package bsvm

import (
	"testing"
	"github.com/bocheninc/L0/nvm"
	"time"
	"fmt"
	"github.com/pborman/uuid"
	"math/rand"
	"strconv"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/types"
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
		luaWorkers[i] = NewBsWorker(nvm.DefaultConfig())
	}

	bsVm := AddNewEnv("bs", luaWorkers)

	cnt := 1
	fn := func(data interface{}) interface{} {
		//fmt.Println(data)
		cnt ++
		return nil
	}

	l0Handler := NewL0Handler()

	initccd := func(name string, txType uint32) *nvm.WorkerProc {
		uid := uuid.New()
		amount := strconv.Itoa(rand.Intn(1000))
		workerProc := &nvm.WorkerProc{
			ContractData: CreateContractDataWithFileName([]string{uid, amount, uid}, name, txType),
			PreMethod: "RealInitContract",
			L0Handler: l0Handler,
		}

		return workerProc
	}


	invokeccd := func(name string, txType uint32) *nvm.WorkerProc {
		uid := uuid.New()
		amount := strconv.Itoa(rand.Intn(1000))
		workerProc := &nvm.WorkerProc{
			ContractData: CreateContractDataWithFileName([]string{"send", uid, amount, uid}, name, txType),
			PreMethod: "RealInvokeExecute",
			L0Handler: l0Handler,
		}

		return workerProc
	}


	time.Sleep(time.Second)
	bsVm.SendWorkCleanAsync(&nvm.WorkerProcWithCallback{
		WorkProc: initccd("l0coin.lua", types.TypeLuaContractInit),
		Fn:fn,
	})
	bsVm.SendWorkCleanAsync(&nvm.WorkerProcWithCallback{
		WorkProc: initccd("l0coin.js", types.TypeJSContractInit),
		Fn:fn,
	})

	time.Sleep(time.Second)
	log.Info("==============start=================")
	fileName := "l0coin.lua"
	txType := types.TypeLuaContractInit
	startTime := time.Now()
	for i := 0; i<8; i++ {
		if i % 2 == 0 {
			fileName = "l0coin.lua"
			txType = types.TypeContractInvoke
		} else {
			fileName = "l0coin.js"
			txType = types.TypeContractInvoke
		}
		bsVm.SendWorkCleanAsync(&nvm.WorkerProcWithCallback{
			WorkProc: invokeccd(fileName, txType),
			Fn:fn,
		})
	}

	fmt.Println("WorkThread: ",workCnt, " Exec time: ", time.Now().Sub(startTime))
	log.Info("==============end=================")

	time.Sleep(5 * time.Second)
	bsVm.Close("bs")
	fmt.Println("cnt: ", cnt)
}
