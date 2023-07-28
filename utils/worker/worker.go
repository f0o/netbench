// Package worker provides the worker interface and worker implementation
package worker

import (
	"context"
	"net/http"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/logger"
)

// worker is the internal worker struct representing a single worker
type worker struct {
	ctx     context.Context
	t       interfaces.WorkerType
	httopts interfaces.HTTPOpts
	blen    int
}

// Do performs the actual work
// it returns an error if the context is canceled or deadline exceeded
func (this *worker) Do() error {
	fn := this.getWorkerFunc()
	for {
		select {
		case <-this.ctx.Done():
			logger.Debug("context done, quitting")
			return this.ctx.Err()
		default:
			fn()
		}
	}
}

// getWorkerFunc returns the worker function
func (this *worker) getWorkerFunc() func() error {
	switch this.t {
	case interfaces.HTTPWorker:
		return this.DoHTTP
	}
	return nil
}

// NewWorker returns a new worker based on the context
// it uses the context to get the target, method, headers and follow flags
// it returns a worker interface
func NewWorker(ctx context.Context, worker_opts *interfaces.WorkerOpts) interfaces.Worker {
	client := http.DefaultClient
	if !worker_opts.Follow {
		client = &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	worker_opts.HTTPOpts.Client = client
	return &worker{
		ctx:     ctx,
		httopts: worker_opts.HTTPOpts,
		t:       worker_opts.Type,
	}
}
