package scaler

import (
	"context"
	"math"
	"time"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/logger"
	"go.f0o.dev/netbench/utils/prometheus"
	"go.f0o.dev/netbench/utils/worker"
)

type scaler struct {
	ctx       context.Context
	curve     float64
	interval  time.Duration
	increment float64
	workers   []context.CancelFunc
}

func (this *scaler) Start() error {
	// this.prometheus.NewCounterFunc("workers", func() float64 {
	// 	return float64(len(this.workers))
	// })
	d := time.NewTicker(this.interval)
	this.scale()
	for {
		select {
		case <-this.ctx.Done():
			for k, w := range this.workers {
				logger.Debug("stopping worker %+v", k)
				w()
			}
			return this.ctx.Err()
		case <-d.C:
			this.scale()
		}
	}

}

func (this *scaler) scale() {
	this.increment++
	for w := float64(len(this.workers)); w < math.Round(math.Pow(this.increment, this.curve)); w++ {
		wc, wf := context.WithCancel(this.ctx)
		this.workers = append(this.workers, wf)
		ww := worker.NewWorker(wc)
		go ww.Do()
	}
	prometheus.Metrics.Workers.Set(float64(len(this.workers)))
	logger.Info("Scaled up to %d workers", len(this.workers))
}

func NewScaler(ctx context.Context) interfaces.Scaler {
	curve := ctx.Value("flags").(interfaces.Flags).Curve
	interval := ctx.Value("flags").(interfaces.Flags).Interval
	return &scaler{
		ctx:       ctx,
		increment: 0,
		curve:     curve,
		interval:  interval,
	}
}
