// Package worker provides the worker interface and worker implementation
package worker

import (
	"context"
	"fmt"
	"strings"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/logger"
)

type WorkerType int

const (
	InvalidWorker WorkerType = iota
	HTTPWorker
	WSWorker
	GRPCWorker
	NetWorker
)

func (this *WorkerType) String() string {
	for k, v := range workers {
		if v == *this {
			return k
		}
	}
	return "unknown"
}

func (this *WorkerType) Set(value string) error {
	scheme := strings.SplitN(value, "://", 2)
	if len(scheme) != 2 || scheme[1] == "" {
		return fmt.Errorf("invalid worker target: %s", value)
	}
	for k, v := range workers {
		if k == scheme[0] {
			*this = v
			return nil
		}
	}
	return fmt.Errorf("invalid worker scheme: %s", scheme[0])
}

// worker is the internal worker struct representing a single worker
type worker struct {
	ctx    context.Context
	worker interfaces.Worker
}

// Do performs the actual work
// it returns an error if the context is canceled or deadline exceeded
func (this *worker) Do() error {
	for {
		select {
		case <-this.ctx.Done():
			logger.Debug("context done, quitting")
			return this.ctx.Err()
		default:
			this.worker.Do()
		}
	}

}

var workers = make(map[string]WorkerType)

func AvailableWorkers() []string {
	var available []string
	for k := range workers {
		available = append(available, k)
	}
	return available
}

// NewWorker returns a new worker based on the context
// it uses the context to get the target, method, headers and follow flags
// it returns a worker interface
func NewWorker(ctx context.Context, worker_opts *interfaces.WorkerOpts) interfaces.Worker {
	t := new(WorkerType)
	err := t.Set(worker_opts.Target)
	if err != nil {
		logger.Fatalw("invalid target or unsupported scheme", "Error", err)
		return nil
	}
	var _worker interfaces.Worker
	switch *t {
	case HTTPWorker:
		worker_opts.HTTPOpts.URL = worker_opts.Target
		_worker = NewHTTPWorker(ctx, &worker_opts.HTTPOpts, worker_opts.Payload)
	case NetWorker:
		target := strings.SplitN(worker_opts.Target, "://", 2)
		worker_opts.NetOpts.Type = target[0]
		worker_opts.NetOpts.Addr = target[1]
		_worker = NewNetWorker(ctx, &worker_opts.NetOpts, worker_opts.Payload)
	default:
		logger.Fatalw("worker type reserved but not yet implemented", "Type", t.String())
		return nil
	}
	return &worker{
		ctx:    ctx,
		worker: _worker,
	}
}
