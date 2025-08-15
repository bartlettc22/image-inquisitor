package callbackworker

import (
	"sync"

	workertypes "github.com/bartlettc22/image-inquisitor/internal/worker/types"
)

type WorkerPool struct {
	done     chan struct{}
	taskChan chan Task
}

type WorkerPoolConfig struct {

	// Default: 3
	NumWorkers int

	// Default: 10000
	TaskChanSize int
}

func NewWorkerPool(c *WorkerPoolConfig) *WorkerPool {

	if c.NumWorkers == 0 {
		c.NumWorkers = workertypes.DefaultNumWorkers
	}

	if c.TaskChanSize == 0 {
		c.TaskChanSize = workertypes.DefaultTaskChanSize
	}

	taskChan := make(chan Task, c.TaskChanSize)
	wg := &sync.WaitGroup{}

	for i := 1; i <= c.NumWorkers; i++ {
		wg.Add(1)
		go Worker(i, taskChan, wg)
	}

	pool := &WorkerPool{
		done:     make(chan struct{}),
		taskChan: taskChan,
	}

	// Start a goroutine to close results channel after all workers finish
	// This is done in a goroutine so we can process results as they come in
	// before all the tasks have been completed
	go func() {
		wg.Wait()
		close(pool.done)
	}()

	return pool
}

func (w *WorkerPool) AddTask(task Task) {
	w.taskChan <- task
}

func (w *WorkerPool) Done() {
	close(w.taskChan)
}

func (w *WorkerPool) Wait() {
	<-w.done
}
