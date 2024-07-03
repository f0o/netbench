package worker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/logger"
	"go.f0o.dev/netbench/utils/prometheus"
)

type httpWorker struct {
	ctx     context.Context
	Client  *http.Client
	URL     string
	Method  string
	Headers map[string]string
	Payload []byte
	Timeout time.Duration
	blen    int
}

func init() {
	workers["http"] = HTTPWorker
	workers["https"] = HTTPWorker
}

func (httpWorker *httpWorker) Do() error {
	ctx, c := context.WithTimeout(httpWorker.ctx, httpWorker.Timeout)
	defer c()
	req, _ := http.NewRequestWithContext(ctx, httpWorker.Method, httpWorker.URL, bytes.NewReader(httpWorker.Payload))
	// req, _ := http.NewRequest(httpWorker.Method, httpWorker.URL, bytes.NewReader(httpWorker.Payload))
	for k, v := range httpWorker.Headers {
		req.Header.Add(k, v)
	}
	start := time.Now()
	resp, err := httpWorker.Client.Do(req)
	stop := time.Since(start)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logger.Trace("context canceled or deadline exceeded")
			prometheus.Metrics.RequestsAborted.Inc()
			return ctx.Err()
		}
		logger.Trace("net/http error: %+v", err)

		prometheus.Metrics.RequestsError.Inc()
		return err
	}
	prometheus.Metrics.GetCodeCounter(resp.StatusCode).Inc()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return ErrHTTPStatus
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logger.Trace("context canceled or deadline exceeded")
			prometheus.Metrics.RequestsAborted.Inc()
			return httpWorker.ctx.Err()
		}
		logger.Trace("io.ReadAll error: %+v", err)
		prometheus.Metrics.RequestsError.Inc()
		return err
	}
	if httpWorker.blen == 0 {
		httpWorker.blen = int(0.9 * float64(len(body))) // 90% of the body length
		prometheus.Metrics.ResponseBytes.Set(float64(len(body)))
	} else if httpWorker.blen > len(body) {
		prometheus.Metrics.RequestsBlength.Inc()
		return ErrDataLength
	}
	prometheus.Metrics.ResponseTimes.Observe(float64(stop))
	return nil
}

func NewHTTPWorker(ctx context.Context, opts *interfaces.HTTPOpts, payload []byte) interfaces.Worker {
	client := http.DefaultClient
	if !opts.Follow {
		client = &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	return &httpWorker{
		ctx:     ctx,
		Client:  client,
		URL:     opts.URL,
		Method:  opts.Method,
		Headers: opts.Headers,
		Payload: payload,
		Timeout: opts.Timeout,
	}
}
