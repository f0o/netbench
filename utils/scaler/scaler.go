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

type workerctx struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// Internal Scaler Struct representing the scaler
type scaler struct {
	ctx        context.Context
	interval   time.Duration
	increment  float64
	workers    []workerctx
	scaler     interfaces.ScalerType
	min, max   float64
	factor     float64
	workeropts *interfaces.WorkerOpts
	wait       chan struct{}
}

func (scaler *scaler) Wait() chan struct{} {
	return scaler.wait
}

// Start starts the scaler
// It will invoke the scaler function every interval to scale the workers
// It will stop when the context is canceled
func (scaler *scaler) Start() error {
	fn := scaler.getScalerFunc()
	d := time.NewTicker(scaler.interval)
	defer d.Stop()
	scaler.scale(fn)
	for {
		select {
		case <-scaler.ctx.Done():
			scaler.Stop()
			return scaler.ctx.Err()
		case <-d.C:
			scaler.scale(fn)
		}
	}
}

// Stop stops the scaler
func (scaler *scaler) Stop() {
	logger.Debug("Stopping scaler; Stopping workers")
	for len(scaler.workers) > 0 {
		scaler.workers[0].cancel()
		<-scaler.workers[0].ctx.Done()
		scaler.workers = scaler.workers[1:]
	}
	logger.Debug("Stopped scaler")
	scaler.wait <- struct{}{}
	close(scaler.wait)
}

// scale scales the workers
func (scaler *scaler) scale(fn func() float64) {
	scaler.increment++
	old := len(scaler.workers)
	target := math.Round(math.Min(math.Max(math.Abs(fn()), scaler.min), scaler.max))
	logger.Debugw("scaling", "target", target, "old", old)
	for w := float64(len(scaler.workers)); w < target; w++ {
		scaler.spawn()
	}
	for w := float64(len(scaler.workers)); w > target; w-- {
		scaler.despawn()
	}
	if old != len(scaler.workers) {
		prometheus.Metrics.Workers.Set(float64(len(scaler.workers)))
		logger.Info("Scaled to %d workers", len(scaler.workers))
	}
}

// spawn spawns a worker
func (scaler *scaler) spawn() {
	wc, wf := context.WithCancel(scaler.ctx)
	scaler.workers = append(scaler.workers, workerctx{
		cancel: wf,
		ctx:    wc})
	ww := worker.NewWorker(wc, scaler.workeropts)
	go ww.Do()
}

// despawn despawns a worker
func (scaler *scaler) despawn() {
	if len(scaler.workers) == 0 {
		return
	}
	go scaler.workers[0].cancel()
	scaler.workers = scaler.workers[1:]
}

// setScalerFunc returns the scaler function
func (scaler *scaler) getScalerFunc() func() float64 {
	switch scaler.scaler {
	case interfaces.CurveScaler:
		return func() float64 {
			return math.Pow(scaler.increment, scaler.factor)
		}
	case interfaces.ExponentialScaler:
		return func() float64 {
			return math.Exp(scaler.increment) * scaler.factor
		}
	case interfaces.LinearScaler:
		return func() float64 {
			return scaler.increment * scaler.factor
		}
	case interfaces.LogarithmicScaler:
		return func() float64 {
			return math.Log(scaler.increment) * scaler.factor
		}
	case interfaces.StaticScaler:
		return func() float64 {
			return scaler.factor
		}
	case interfaces.SineScaler:
		return func() float64 {
			return math.Sin(scaler.increment/scaler.factor) * scaler.max
		}
	}
	logger.Fatalw("invalid scaler type", "scaler", scaler.scaler)
	return nil
}

// NewScaler returns a new scaler based on the context
func NewScaler(ctx context.Context, scaler_opts *interfaces.ScalerOpts, worker_opts *interfaces.WorkerOpts) interfaces.Scaler {
	min := float64(scaler_opts.Min)
	max := float64(scaler_opts.Max)
	if scaler_opts.Type == interfaces.StaticScaler {
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
		wait:       make(chan struct{}),
	}
}
