// Package scaler implements a scaler for workers
//
// The scaler is a simple interface that handles scaling of workers
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

// Internal Scaler Struct representing the scaler
type scaler struct {
	ctx        context.Context
	interval   time.Duration
	increment  float64
	workers    []context.CancelFunc
	scaler     string
	min, max   float64
	factor     float64
	workeropts *interfaces.WorkerOpts
}

// Start starts the scaler
// It will invoke the scaler function every interval to scale the workers
// It will stop when the context is canceled
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

// scale scales the workers
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

// spawn spawns a worker
func (this *scaler) spawn() {
	wc, wf := context.WithCancel(this.ctx)
	this.workers = append(this.workers, wf)
	ww := worker.NewWorker(wc, this.workeropts)
	go ww.Do()
}

// despawn despawns a worker
func (this *scaler) despawn() {
	if len(this.workers) == 0 {
		return
	}
	go this.workers[0]()
	this.workers = this.workers[1:]
}

// setScalerFunc returns the scaler function
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
	case "logarithmic", "log":
		return func() float64 {
			return math.Log(this.increment) * this.factor
		}
	case "static":
		return func() float64 {
			return this.factor
		}
	case "sine", "sin":
		return func() float64 {
			return math.Sin(this.increment/this.factor) * this.max
		}
	}
	logger.Fatalw("invalid scaler type", "scaler", this.scaler)
	return nil
}

// NewScaler returns a new scaler based on the context
func NewScaler(ctx context.Context, scaler_opts *interfaces.ScalerOpts, worker_opts *interfaces.WorkerOpts) interfaces.Scaler {
	min := float64(scaler_opts.Min)
	max := float64(scaler_opts.Max)
	if scaler_opts.Type == "static" {
		min = scaler_opts.Factor
		max = scaler_opts.Factor
	}
	return &scaler{
		ctx:        ctx,
		scaler:     scaler_opts.Type,
		increment:  0,
		interval:   scaler_opts.Period,
		min:        min,
		max:        max,
		factor:     scaler_opts.Factor,
		workeropts: worker_opts,
	}
}
