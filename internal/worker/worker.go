package worker

import (
	"sync"
)

type Task interface {
	Run(workerID int) Result
}

type Result interface {
	Result() interface{}
	Errors() []error
}

func Worker(id int, taskChan <-chan Task, resultChan chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range taskChan {
		resultChan <- task.Run(id)
	}
}
