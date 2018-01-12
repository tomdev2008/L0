package vm

import (
	"time"
	"sync/atomic"
	"github.com/bocheninc/L0/components/log"
)

type VmWorker interface {
	//called for job, adn returned synchronously
	VmJob(interface{}) (interface{}, error)

	//wait for to execute the next job
	VmReady() bool
}

type VmExtendedWorker interface {
	// when the mechine is opened and closed, the will be implemented.
	VmInitialize()
	VmTerminate()
}

type VmInterruptableWorker interface {
	//called by the client that will be killed this worker
	VmInterruptable()
}

// the vm DefaultWorker
type VmDefaultWorker struct {
	job *func(interface{}) interface{}
}

func (worker *VmDefaultWorker) VmJob(data interface{}) (interface{}, error) {
	return (*worker.job)(data), nil
}

func (worker *VmDefaultWorker) VmReady() bool {
	return true
}

// the external worker
type workerWrapper struct {
	readyChan chan int
	jobChan  chan interface{}
	outputChan chan interface{}
	workerMechine uint32
	worker VmWorker
	jobCnt int
}

func (ww *workerWrapper) Open() {
	if extWorker, ok := ww.worker.(VmExtendedWorker); ok {
		extWorker.VmInitialize()
	}

	ww.readyChan = make(chan int)
	ww.jobChan = make(chan interface{})
	ww.outputChan = make(chan interface{})

	atomic.SwapUint32(&ww.workerMechine, 1)
	go ww.Loop()
}

func (ww *workerWrapper) Close() {
	log.Debugf("vm to close, jobCnt: %d", ww.jobCnt)
	close(ww.jobChan)
	atomic.SwapUint32(&ww.workerMechine, 0)
}

func (ww *workerWrapper) Loop() {
	// TODO: now wait for the next job to come through sleep
	//thread := rand.Int()
	//log.Debugf("==> ww: %+v, %p", thread, ww)
	waitNextJob := func() {
		for !ww.worker.VmReady() {
			if atomic.LoadUint32(&ww.workerMechine) == 0 {
				break
			}

			time.Sleep(5 * time.Millisecond)
		}

		ww.readyChan <- 1
	}

	waitNextJob()
	for data := range ww.jobChan {
		//ww.outputChan <- ww.worker.VmJob(data)
		ww.worker.VmJob(data)
		ww.jobCnt ++
		waitNextJob()
	}
	close(ww.readyChan)
	close(ww.outputChan)
}

func (ww *workerWrapper) Join() {
	for {
		_, readyChan := <-ww.readyChan
		_, outputChan := <-ww.outputChan
		if !readyChan && !outputChan {
			break
		}
	}

	if extWorker, ok  := ww.worker.(VmExtendedWorker); ok {
		extWorker.VmTerminate()
	}
}

func (ww *workerWrapper) Interrupt() {
	if interrupttWorker, ok := ww.worker.(VmInterruptableWorker); ok {
		interrupttWorker.VmInterruptable()
	}
}
