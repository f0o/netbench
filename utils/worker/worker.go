// Package worker provides the worker interface and worker implementation
package worker

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/logger"
	"go.f0o.dev/netbench/utils/prometheus"
)

// worker is the internal worker struct representing a single worker
type worker struct {
	ctx     context.Context
	client  *http.Client
	target  string
	method  string
	headers map[string]string
	blen    int
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
			req, _ := http.NewRequestWithContext(this.ctx, this.method, this.target, nil)
			for k, v := range this.headers {
				req.Header.Add(k, v)
			}
			start := time.Now()
			resp, err := this.client.Do(req)
			stop := time.Since(start)
			prometheus.Metrics.RequestsTotal.Inc()
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					logger.Debug("context canceled or deadline exceeded")
					prometheus.Metrics.RequestsAborted.Inc()
					continue
				}
				logger.Debug("net/http error: %+v", err)
				prometheus.Metrics.RequestsFailed.Inc()
				prometheus.Metrics.RequestsError.Inc()
				continue
			}
			prometheus.Metrics.GetCodeCounter(resp.StatusCode).Inc()
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				prometheus.Metrics.RequestsFailed.Inc()
				continue
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if this.blen == 0 {
				this.blen = int(0.9 * float64(len(body))) // 90% of the body length
			} else if this.blen > len(body) {
				prometheus.Metrics.RequestsFailed.Inc()
				prometheus.Metrics.RequestsBlength.Inc()
				continue
			}
			prometheus.Metrics.ResponseTimes.Observe(float64(stop))
		}
	}
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
	return &worker{
		ctx:     ctx,
		target:  worker_opts.URL,
		method:  worker_opts.Method,
		headers: worker_opts.Headers,
		client:  client,
	}
}
