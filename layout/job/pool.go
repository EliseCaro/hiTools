package job

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const (
	FLAG_OK    Flag = 1 << iota
	FLAG_RETRY Flag = 1 << iota
)

type Pool struct {
	workMaxTotal int        //工作数
	workChan     chan *Task //工作任务

	// 等待相关
	sleepCtx        context.Context
	sleepCancelFunc context.CancelFunc
	sleepSeconds    int64
	sleepNotify     chan bool

	// 停止
	stopCtx        context.Context
	stopCancelFunc context.CancelFunc
	wg             sync.WaitGroup
	// 异常处理
	PanicHandler func(interface{})
}

func NewPool(workMaxTotal, poolLen int) *Pool {
	return &Pool{
		workMaxTotal: workMaxTotal,
		workChan:     make(chan *Task, poolLen),
		sleepNotify:  make(chan bool),
	}
}

func (w *Pool) PushTask(t *Task) {
	w.workChan <- t
}

func (w *Pool) PushTaskFunc(f TaskFunc, params ...interface{}) {
	w.workChan <- &Task{
		f:      f,
		params: params,
	}
}

func (pool *Pool) Work(id int) {
	defer func() {
		if r := recover(); r != nil {
			if pool.PanicHandler != nil {
				pool.PanicHandler(r)
			} else { // 默认处理
				log.Printf("Worker panic: %v\n", r)
			}
		}
	}()

	for {
		select {
		case <-pool.stopCtx.Done():
			pool.wg.Done()
			return
		case <-pool.sleepCtx.Done():
			time.Sleep(time.Duration(pool.sleepSeconds) * time.Second)
		case t := <-pool.workChan:
			flag := t.ExecuteWork(pool)
			if flag&FLAG_RETRY != 0 {
				pool.PushTask(t)
				fmt.Printf("work %v PushTask, pool length %v\n", id, len(pool.workChan))
			}
		}
	}
}

func (w *Pool) Run() *Pool {
	fmt.Printf("workpool run %d worker\n", w.workMaxTotal)
	w.wg.Add(w.workMaxTotal + 1)
	w.stopCtx, w.stopCancelFunc = context.WithCancel(context.Background())
	w.sleepCtx, w.sleepCancelFunc = context.WithCancel(context.Background())
	go w.sleepControl()
	for i := 0; i < w.workMaxTotal; i++ {
		go w.Work(i)
	}
	return w
}

//设置延时通知
func (w *Pool) SleepNotify(seconds int64) {
	if atomic.CompareAndSwapInt64(&w.sleepSeconds, 0, seconds) {
		w.sleepNotify <- true
	}
}

func (w *Pool) sleepControl() {
	for {
		select {
		case <-w.stopCtx.Done():
			w.wg.Done()
			return
		case <-w.sleepNotify:
			w.sleepCtx, w.sleepCancelFunc = context.WithCancel(context.Background())
			w.sleepCancelFunc()
			time.Sleep(time.Duration(w.sleepSeconds) * time.Second)
			w.sleepSeconds = 0
		}
	}
}

func (w *Pool) Stop() {
	w.stopCancelFunc()
	w.wg.Wait()
}
