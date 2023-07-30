package worker

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"time"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/logger"
	"go.f0o.dev/netbench/utils/prometheus"
)

var (
	NetErrDataLength = errors.New("data length mismatch")
)

type netWorker struct {
	ctx     context.Context
	Type    string
	Addr    string
	Timeout time.Duration
	Payload string
	blen    int
}

func init() {
	workers["tcp"] = NetWorker
	workers["udp"] = NetWorker
	workers["unix"] = NetWorker
}

func (this *netWorker) Do() error {
	prometheus.Metrics.RequestsTotal.Inc()
	start := time.Now()
	blen, err := this.Dial()
	stop := time.Since(start)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logger.Debug("context canceled or deadline exceeded")
			prometheus.Metrics.RequestsAborted.Inc()
			return this.ctx.Err()
		}
		logger.Debugw("failed to connect to socket", "Error", err)
		prometheus.Metrics.RequestsFailed.Inc()
		prometheus.Metrics.RequestsError.Inc()
		return err
	}
	if err != nil {
		logger.Debugw("failed to read from socket", "Error", err)
		prometheus.Metrics.RequestsFailed.Inc()
		prometheus.Metrics.RequestsError.Inc()
		return err
	}
	if this.blen == 0 {
		this.blen = int(0.9 * float64(blen)) // 90% of the body length
		prometheus.Metrics.ResponseBytes.Set(float64(blen))
	} else if this.blen > blen {
		prometheus.Metrics.RequestsFailed.Inc()
		prometheus.Metrics.RequestsBlength.Inc()
		return NetErrDataLength
	}
	prometheus.Metrics.GetCodeCounter(200).Inc()
	prometheus.Metrics.ResponseTimes.Observe(float64(stop))
	return nil
}

func (this *netWorker) Dial() (int, error) {
	// var err error
	dailer := &net.Dialer{}
	conn, err := dailer.DialContext(this.ctx, this.Type, this.Addr)
	if err != nil {
		return -1, err
	}
	conn.SetDeadline(time.Now().Add(this.Timeout))
	if this.Payload != "" {
		_, err = conn.Write([]byte(this.Payload))
		if err != nil {
			logger.Debugw("failed to write to socket", "Error", err)
			return -1, err
		}
	}
	defer conn.Close()
	return this.Read(conn)
}

func (this *netWorker) Read(conn net.Conn) (int, error) {
	br := bufio.NewReader(conn)
	for {
		select {
		case <-this.ctx.Done():
			logger.Debug("context done, quitting")
			return -1, this.ctx.Err()
		default:
			_, err := br.Peek(1)
			if err == io.EOF {
				return -1, nil
			} else if err != nil {
				return -1, err
			} else {
				n, _ := br.Read(make([]byte, br.Buffered()))
				return n, nil
			}
		}
	}
}

func NewNetWorker(ctx context.Context, opts *interfaces.NetOpts, payload string) interfaces.Worker {
	return &netWorker{
		ctx:     ctx,
		Type:    opts.Type,
		Addr:    opts.Addr,
		Timeout: opts.Timeout,
		Payload: payload,
	}
}
