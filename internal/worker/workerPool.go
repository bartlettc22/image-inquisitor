package worker

import (
	"sync"

	workertypes "github.com/bartlettc22/image-inquisitor/internal/worker/types"
)

type EphemeralWorkerPool struct {
	taskChan   chan Task
	resultChan chan Result
}

type EphemeralWorkerPoolConfig struct {

	// Default: 3
	NumWorkers int

	// Default: 10000
	TaskChanSize int

	// Default: 10000
	ResultChanSize int
}

func NewEphemeralWorkerPool(c EphemeralWorkerPoolConfig) *EphemeralWorkerPool {

	if c.NumWorkers == 0 {
		c.NumWorkers = workertypes.DefaultNumWorkers
	}

	if c.TaskChanSize == 0 {
		c.TaskChanSize = workertypes.DefaultTaskChanSize
	}

	if c.ResultChanSize == 0 {
		c.ResultChanSize = workertypes.DefaultResultChanSize
	}

	taskChan := make(chan Task, c.TaskChanSize)
	resultChan := make(chan Result, c.ResultChanSize)
	wg := &sync.WaitGroup{}

	for i := 0; i < c.NumWorkers; i++ {
		wg.Add(1)
		go Worker(i, taskChan, resultChan, wg)
	}

	// Start a goroutine to close results channel after all workers finish
	// This is done in a goroutine so we can process results as they come in
	// before all the tasks have been completed
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	return &EphemeralWorkerPool{
		taskChan:   taskChan,
		resultChan: resultChan,
	}
}

func (w *EphemeralWorkerPool) AddTask(task Task) {
	w.taskChan <- task
}

func (w *EphemeralWorkerPool) Done() {
	close(w.taskChan)
}

func (w *EphemeralWorkerPool) ResultChan() chan Result {
	return w.resultChan
}
