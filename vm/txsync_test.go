package vm

import (
	"testing"
	"fmt"
	"time"
	"math/rand"
)

var workCnt = 10
type TestWorker struct {
	threadID int
}

func (tw *TestWorker) VmJob(data interface{}) interface{} {
	value := data.(int)
	num := rand.Intn(10)
	time.Sleep(time.Duration(num) * time.Millisecond)
	fmt.Println(tw.threadID," =.get.= ", value)
	if value != 0 {
		fmt.Println(tw.threadID, "      wait ", value)
		Txsync.Wait(value%workCnt)
	}
	fmt.Println(tw.threadID, " ===>>> ", value)

	fmt.Println(tw.threadID, "      notify ", value+1)
	Txsync.Notify((value + 1) % workCnt)

	return nil
}

func (tw *TestWorker) VmReady() bool {
	return true
}

func NewTestWorker(i int) *TestWorker {
	return &TestWorker{
		threadID: i,
	}
}
func TestWorkerFunc(t *testing.T) {
	fmt.Println("===================")
	Txsync = NewTxSync(workCnt)
	testWorkers := make([]VmWorker, workCnt)
	for i:=0; i<workCnt; i++ {
		testWorkers[i] = NewTestWorker(i)
	}
	tstVM := CreateCustomVM(testWorkers)
	tstVM.Open("test")
	for i:=0; i<5000; i++ {
		tstVM.SendWorkCleanAsync(i)
	}

	select {
	}
}