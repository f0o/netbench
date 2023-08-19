// Package worker provides the worker interface and worker implementation
package worker

import (
	"context"
	"encoding/base64"
	"strings"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/logger"
	"go.f0o.dev/netbench/utils/prometheus"
)

// worker is the internal worker struct representing a single worker
type worker struct {
	ctx    context.Context
	worker interfaces.Worker
	sync   bool
}

// Do performs the actual work
// it returns an error if the context is canceled or deadline exceeded
func (worker *worker) Do() error {
	var signal int = -1
	for {
		select {
		case <-worker.ctx.Done():
			logger.Trace("context done, quitting")
			if worker.sync && signal != -1 {
				syncWorkDel(signal)
			}
			return worker.ctx.Err()
		default:
			if worker.sync && signal == -1 {
				signal = syncWorkAdd()
			}
			err := worker.worker.Do()
			prometheus.Metrics.RequestsTotal.Inc()
			if err != nil {
				logger.Trace("worker error: %+v", err)
				prometheus.Metrics.RequestsFailed.Inc()
			}
			if worker.sync {
				syncWorkWait(signal)
			}
		}
	}

}

// NewWorker returns a new worker based on the context
// it uses the context to get the target, method, headers and follow flags
// it returns a worker interface
func NewWorker(ctx context.Context, worker_opts *interfaces.WorkerOpts) interfaces.Worker {
	var payload []byte
	var err error
	if worker_opts.Payload != "" {
		payload, err = base64.StdEncoding.DecodeString(worker_opts.Payload)
		if err != nil {
			logger.Fatalw("failed to decode payload", "Error", err)
			return nil
		}
	}
	t := new(WorkerType)
	err = t.Set(worker_opts.Target)
	if err != nil {
		logger.Fatalw("invalid target or unsupported scheme", "Error", err)
		return nil
	}
	var _worker interfaces.Worker
	switch *t {
	case HTTPWorker:
		worker_opts.HTTPOpts.URL = worker_opts.Target
		_worker = NewHTTPWorker(ctx, &worker_opts.HTTPOpts, payload)
	case NetWorker:
		target := strings.SplitN(worker_opts.Target, "://", 2)
		worker_opts.NetOpts.Type = target[0]
		worker_opts.NetOpts.Addr = target[1]
		_worker = NewNetWorker(ctx, &worker_opts.NetOpts, payload)
	default:
		logger.Fatalw("worker type reserved but not yet implemented", "Type", t.String())
		return nil
	}
	return &worker{
		ctx:    ctx,
		worker: _worker,
		sync:   worker_opts.Sync,
	}
}
