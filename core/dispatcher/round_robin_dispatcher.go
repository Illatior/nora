package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"load-testing/config"
	"load-testing/core/worker"
	"sync/atomic"
	"time"
)

type roundRobinDispatcher struct {
	done              chan bool
	errorHandlingDone chan bool

	workers       []worker.Worker
	currentWorker uint64

	rps uint64

	errChan chan error
}

func NewRoundRobinDispatcher(cfg config.LoadTestConfig) Dispatcher {
	return &roundRobinDispatcher{
		done:              make(chan bool),
		errorHandlingDone: make(chan bool),
		workers:           make([]worker.Worker, 0),
		currentWorker:     0,
		rps:               cfg.RequestsPerSecond,
		errChan:           make(chan error, 1),
	}
}

func (d *roundRobinDispatcher) Dispatch(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return d.processErrors()
	})

	tickInterval := time.Second / time.Duration(d.rps)
	ticker := time.NewTicker(tickInterval)

	g.Go(func() error {
		for {
			select {
			case <-d.done:
				for _, worker := range d.workers {
					worker.Stop()
				}
				return nil
			case <-ticker.C:
				indx, err := d.nextIndex()
				if err != nil {
					panic(err)
				}

				go func(indx int) {
					err := d.workers[indx].Run()
					if err != nil {
						d.errChan <- err
					}
				}(indx)
			}
		}
	})

	return g.Wait()
}

func (d *roundRobinDispatcher) AddWorker(id string, worker *worker.Worker) error {
	d.workers = append(d.workers, *worker)

	return nil
}

func (d *roundRobinDispatcher) Shutdown() error {
	d.done <- true
	d.errorHandlingDone <- true
	return nil
}

func (d *roundRobinDispatcher) processErrors() error {
	for {
		select {
		case <-d.errorHandlingDone:
			return nil
		case err := <-d.errChan:
			fmt.Println(err)
		}
	}
}

func (d *roundRobinDispatcher) nextIndex() (int, error) {
	if len(d.workers) == 0 {
		return 0, errors.New("Workers are not ready yet!")
	}

	return int(atomic.AddUint64(&d.currentWorker, uint64(1)) % uint64(len(d.workers))), nil
}
