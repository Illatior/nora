package executor

import (
	"context"
	"github.com/illatior/nora/core/metric"
	"github.com/illatior/nora/core/task"
	"sync"
)

type simpleExecutor struct {
	tasks []task.Task
}

func New() Executor {
	return &simpleExecutor{
		tasks: make([]task.Task, 0),
	}
}

func (e *simpleExecutor) AddTask(task task.Task) {
	e.tasks = append(e.tasks, task)
}

func (e *simpleExecutor) ScheduleExecution(ctx context.Context, ticks <-chan interface{}, results chan<- *metric.Result) {
	var wg sync.WaitGroup

	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			for _, t := range e.tasks {

				wg.Add(1)
				go func(t task.Task) {
					defer wg.Done()
					results <- t.Run(childCtx)
				}(t)
			}
		}
	}
}
