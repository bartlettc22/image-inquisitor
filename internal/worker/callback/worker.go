package callbackworker

import (
	"sync"
)

type Task interface {
	Run() (interface{}, error)
	Callback(interface{}, error)
}

func Worker(id int, taskChan <-chan Task, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range taskChan {
		results, errs := task.Run()
		task.Callback(results, errs)
	}
}
