package callbackworker

type callbackTask struct {
	runFn    func() (interface{}, error)
	callback func(interface{}, error)
}

// Ensure that the callbackTask implements the Task interface
var _ Task = (*callbackTask)(nil)

func NewCallbackTask(runFn func() (interface{}, error), callbackFn func(interface{}, error)) *callbackTask {
	return &callbackTask{
		runFn:    runFn,
		callback: callbackFn,
	}
}

func (t *callbackTask) Run() (interface{}, error) {
	return t.runFn()
}

func (t *callbackTask) Callback(result interface{}, errs error) {
	t.callback(result, errs)
}
