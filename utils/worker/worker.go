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

type worker struct {
	ctx     context.Context
	target  string
	method  string
	headers map[string]string
	blen    int
}

func (this *worker) Do() error {
	for {
		select {
		case <-this.ctx.Done():
			return this.ctx.Err()
		default:
			req, _ := http.NewRequestWithContext(this.ctx, this.method, this.target, nil)
			for k, v := range this.headers {
				req.Header.Add(k, v)
			}
			start := time.Now()
			resp, err := http.DefaultClient.Do(req)
			stop := time.Since(start)
			prometheus.Metrics.RequestsTotal.Inc()
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					logger.Debug("context canceled or deadline exceeded")
					prometheus.Metrics.RequestsAborted.Inc()
					return err
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
				this.blen = len(body) / 2
			} else if this.blen > len(body) {
				prometheus.Metrics.RequestsFailed.Inc()
				prometheus.Metrics.RequestsBlength.Inc()
				continue
			}
			prometheus.Metrics.RequestsTime.Observe(float64(stop))
		}
	}
}

func NewWorker(ctx context.Context) interfaces.Worker {
	target := ctx.Value("flags").(interfaces.Flags).WorkerOpts.URL
	method := ctx.Value("flags").(interfaces.Flags).WorkerOpts.Method
	headers := ctx.Value("flags").(interfaces.Flags).WorkerOpts.Headers
	return &worker{
		ctx:     ctx,
		target:  target,
		method:  method,
		headers: headers,
	}
}
