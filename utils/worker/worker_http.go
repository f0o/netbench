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

var (
	HTTPErrBodyLength = errors.New("body length mismatch")
	HTTPErrStatus     = errors.New("status code mismatch")
)

type httpWorker struct {
	ctx     context.Context
	Client  *http.Client
	URL     string
	Method  string
	Headers map[string]string
	Payload []byte
	blen    int
}

func init() {
	workers["http"] = HTTPWorker
	workers["https"] = HTTPWorker
}

func (this *httpWorker) Do() error {
	req, _ := http.NewRequestWithContext(this.ctx, this.Method, this.URL, bytes.NewReader(this.Payload))
	for k, v := range this.Headers {
		req.Header.Add(k, v)
	}
	prometheus.Metrics.RequestsTotal.Inc()
	start := time.Now()
	resp, err := this.Client.Do(req)
	stop := time.Since(start)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logger.Debug("context canceled or deadline exceeded")
			prometheus.Metrics.RequestsAborted.Inc()
			return this.ctx.Err()
		}
		logger.Debugw("net/http error: %+v", err)
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
	defer resp.Body.Close()
	if this.blen == 0 {
		this.blen = int(0.9 * float64(len(body))) // 90% of the body length
		prometheus.Metrics.ResponseBytes.Set(float64(len(body)))
	} else if this.blen > len(body) {
		prometheus.Metrics.RequestsFailed.Inc()
		prometheus.Metrics.RequestsBlength.Inc()
		return HTTPErrBodyLength
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
	}
}
