package node

import (
	"github.com/bocheninc/base/log"
)

type job struct {
	In   interface{}
	Exec func(interface{})
}

func worker(id int, jobs <-chan *job) {
	for {
		select {
		case j := <-jobs:
			log.Debugf("[jobs] worker %d work ...", id)
			j.Exec(j.In)
		}
	}
}

func startJobs(n int, jobs chan *job) {
	log.Infof("[jobs] start workers %d ...", n)
	for i := 1; i <= n; i++ {
		go worker(i, jobs)
	}
}
