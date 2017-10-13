package node

type job struct {
	In   interface{}
	Exec func(interface{})
}

func worker(id int, jobs <-chan *job) {
	for {
		select {
		case j := <-jobs:
			j.Exec(j.In)
		}
	}
}

func startJobs(n int, jobs chan *job) {
	for i := 1; i <= n; i++ {
		go worker(i, jobs)
	}
}
