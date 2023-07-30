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
	Payload string
	blen    int
}

func init() {
	workers["tcp"] = NetWorker
	workers["udp"] = NetWorker
	workers["unix"] = NetWorker
}

func (this *netWorker) Do() error {
	// connect to tcp service
	start := time.Now()
	blen, err := this.Dial()
	stop := time.Since(start)
	prometheus.Metrics.RequestsTotal.Inc()
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			defer logger.Debug("context canceled or deadline exceeded")
			prometheus.Metrics.RequestsAborted.Inc()
			return this.ctx.Err()
		}
		defer logger.Debugw("failed to connect to tcp service", "Error", err)
		prometheus.Metrics.RequestsFailed.Inc()
		prometheus.Metrics.RequestsError.Inc()
		return err
	}
	if err != nil {
		defer logger.Debugw("failed to read from tcp service", "Error", err)
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
	if this.Payload != "" {
		_, err = conn.Write([]byte(this.Payload))
		if err != nil {
			defer logger.Debugw("failed to write to tcp service", "Error", err)
			return -1, err
		}
	}
	defer conn.Close()
	return this.Read(conn)
}

func (this *netWorker) Read(conn net.Conn) (int, error) {
	br := bufio.NewReader(conn)
	N := 0
	for {
		select {
		case <-this.ctx.Done():
			defer logger.Debug("context done, quitting")
			return 0, this.ctx.Err()
		default:
			_, err := br.Peek(1)
			if err == io.EOF {
				return N, nil
			} else if err != nil {
				return -1, err
			} else {
				p := br.Buffered()
				buf := make([]byte, p)
				n, _ := br.Read(buf)
				N += n
				defer logger.Debugw("read from tcp service", "Bytes", n, "Total", N)
				return N, nil
			}
		}
	}
}

func NewNetWorker(ctx context.Context, opts *interfaces.NetOpts, payload string) interfaces.Worker {
	return &netWorker{
		ctx:     ctx,
		Type:    opts.Type,
		Addr:    opts.Addr,
		Payload: payload,
	}
}
