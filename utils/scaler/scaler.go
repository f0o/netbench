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
	interval  time.Duration
	increment float64
	workers   []context.CancelFunc
	scaler    string
	min, max  float64
	factor    float64
}

func (this *scaler) Start() error {
	fn := this.setScalerFunc()
	d := time.NewTicker(this.interval)
	defer d.Stop()
	this.scale(fn)
	for {
		select {
		case <-this.ctx.Done():
			for k, w := range this.workers {
				logger.Debug("stopping worker %+v", k)
				w()
			}
			return this.ctx.Err()
		case <-d.C:
			this.scale(fn)
		}
	}
}

func (this *scaler) scale(fn func() float64) {
	this.increment++
	old := len(this.workers)
	target := math.Min(math.Max(math.Round(math.Abs(fn())), this.min), this.max)
	for w := float64(len(this.workers)); w < target; w++ {
		this.spawn()
	}
	for w := float64(len(this.workers)); w > target; w-- {
		this.despawn()
	}
	if old != len(this.workers) {
		prometheus.Metrics.Workers.Set(float64(len(this.workers)))
		logger.Info("Scaled to %d workers", len(this.workers))
	}
}

func (this *scaler) spawn() {
	wc, wf := context.WithCancel(this.ctx)
	this.workers = append(this.workers, wf)
	ww := worker.NewWorker(wc)
	go ww.Do()
}

func (this *scaler) despawn() {
	if len(this.workers) == 0 {
		return
	}
	go this.workers[0]()
	this.workers = this.workers[1:]
}

func (this *scaler) setScalerFunc() func() float64 {
	switch this.scaler {
	case "curve":
		return func() float64 {
			return math.Pow(this.increment, this.factor)
		}
	case "exponential", "exp":
		return func() float64 {
			return math.Exp(this.increment) * this.factor
		}
	case "linear":
		return func() float64 {
			return this.increment * this.factor
		}
	case "log":
		return func() float64 {
			this.increment++
			return math.Log(this.increment) * this.factor
		}
	case "static":
		return func() float64 {
			return this.factor
		}
	case "sin", "sine":
		return func() float64 {
			return math.Sin(this.increment/this.factor) * this.max
		}
	}
	logger.Fatalw("invalid scaler type", "scaler", this.scaler)
	return nil
}
func NewScaler(ctx context.Context) interfaces.Scaler {
	interval := ctx.Value("flags").(interfaces.Flags).ScalerOpts.Period
	scaler_type := ctx.Value("flags").(interfaces.Flags).ScalerOpts.Type
	min := float64(ctx.Value("flags").(interfaces.Flags).ScalerOpts.Min)
	max := float64(ctx.Value("flags").(interfaces.Flags).ScalerOpts.Max)
	factor := float64(ctx.Value("flags").(interfaces.Flags).ScalerOpts.Factor)
	if scaler_type == "static" {
		min = factor
		max = factor
	}
	return &scaler{
		ctx:       ctx,
		scaler:    scaler_type,
		increment: 0,
		interval:  interval,
		min:       min,
		max:       max,
		factor:    factor,
	}
}
