package layout

import (
	"hiTools/layout/job"
	"sync"
)

type JobPoolService struct{}

var pool *job.Pool
var once sync.Once

func getInstance(maxPoolTotal, poolLength int) *job.Pool {
	once.Do(func() {
		pool = job.NewPool(maxPoolTotal, poolLength)
		pool.Run()
	})
	return pool
}

func (service *JobPoolService) GetPool() *job.Pool {
	maxPoolTotal := 20
	poolLength := 3
	return getInstance(maxPoolTotal, poolLength)
}
