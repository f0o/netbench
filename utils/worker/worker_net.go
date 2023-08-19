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

type netWorker struct {
	ctx     context.Context
	Type    string
	Addr    string
	Timeout time.Duration
	Payload []byte
	blen    int
}

func init() {
	workers["tcp"] = NetWorker
	workers["udp"] = NetWorker
	workers["unix"] = NetWorker
}

func (netWorker *netWorker) Do() error {
	start := time.Now()
	blen, err := netWorker.Dial()
	stop := time.Since(start)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logger.Trace("context canceled or deadline exceeded")
			prometheus.Metrics.RequestsAborted.Inc()
			return netWorker.ctx.Err()
		}
		logger.Tracew("failed to connect to socket", "Error", err)
		prometheus.Metrics.RequestsError.Inc()
		return err
	}
	if netWorker.blen == 0 {
		netWorker.blen = int(0.9 * float64(blen)) // 90% of the body length
		prometheus.Metrics.ResponseBytes.Set(float64(blen))
	} else if netWorker.blen > blen {
		prometheus.Metrics.RequestsBlength.Inc()
		return ErrDataLength
	}
	prometheus.Metrics.GetCodeCounter(200).Inc()
	prometheus.Metrics.ResponseTimes.Observe(float64(stop))
	return nil
}

func (netWorker *netWorker) Dial() (int, error) {
	// var err error
	dailer := &net.Dialer{}
	conn, err := dailer.DialContext(netWorker.ctx, netWorker.Type, netWorker.Addr)
	if err != nil {
		return -1, err
	}
	err = conn.SetDeadline(time.Now().Add(netWorker.Timeout))
	if err != nil {
		return -1, err
	}
	if netWorker.Payload != nil {
		_, err = conn.Write(netWorker.Payload)
		if err != nil {
			logger.Tracew("failed to write to socket", "Error", err)
			return -1, err
		}
	}
	defer conn.Close()
	return netWorker.Read(conn)
}

func (netWorker *netWorker) Read(conn net.Conn) (int, error) {
	br := bufio.NewReader(conn)
	for {
		select {
		case <-netWorker.ctx.Done():
			logger.Trace("context done, quitting")
			return -1, netWorker.ctx.Err()
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

func NewNetWorker(ctx context.Context, opts *interfaces.NetOpts, payload []byte) interfaces.Worker {
	return &netWorker{
		ctx:     ctx,
		Type:    opts.Type,
		Addr:    opts.Addr,
		Timeout: opts.Timeout,
		Payload: payload,
	}
}
