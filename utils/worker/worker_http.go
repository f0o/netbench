package worker

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"go.f0o.dev/netbench/utils/logger"
	"go.f0o.dev/netbench/utils/prometheus"
)

var (
	HTTPErrBodyLength = errors.New("body length mismatch")
	HTTPErrStatus     = errors.New("status code mismatch")
)

func (this *worker) DoHTTP() error {
	req, _ := http.NewRequestWithContext(this.ctx, this.httopts.Method, this.httopts.URL, nil)
	for k, v := range this.httopts.Headers {
		req.Header.Add(k, v)
	}
	start := time.Now()
	resp, err := this.httopts.Client.Do(req)
	stop := time.Since(start)
	prometheus.Metrics.RequestsTotal.Inc()
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logger.Debug("context canceled or deadline exceeded")
			prometheus.Metrics.RequestsAborted.Inc()
			return this.ctx.Err()
		}
		logger.Debug("net/http error: %+v", err)
		prometheus.Metrics.RequestsFailed.Inc()
		prometheus.Metrics.RequestsError.Inc()
		return err
	}
	prometheus.Metrics.GetCodeCounter(resp.StatusCode).Inc()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		prometheus.Metrics.RequestsFailed.Inc()
		return HTTPErrStatus
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if this.blen == 0 {
		this.blen = int(0.9 * float64(len(body))) // 90% of the body length
	} else if this.blen > len(body) {
		prometheus.Metrics.RequestsFailed.Inc()
		prometheus.Metrics.RequestsBlength.Inc()
		return HTTPErrBodyLength
	}
	prometheus.Metrics.ResponseTimes.Observe(float64(stop))
	return nil
}
