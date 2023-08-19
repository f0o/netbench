package scaler

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"testing"
	"time"

	"go.f0o.dev/netbench/interfaces"
	"go.uber.org/goleak"
)

var (
	rng       = rand.New(rand.NewSource(time.Now().UnixNano()))
	factor    = rng.Float64()
	testCases = []struct {
		Type   interfaces.ScalerType
		Factor float64
		Fn     func(float64) float64
	}{
		{Type: interfaces.StaticScaler, Factor: factor, Fn: func(i float64) float64 { return factor }},
		{Type: interfaces.LinearScaler, Factor: factor, Fn: func(i float64) float64 { return i * factor }},
		{Type: interfaces.ExponentialScaler, Factor: factor, Fn: func(i float64) float64 { return math.Exp(i) * factor }},
		{Type: interfaces.CurveScaler, Factor: factor, Fn: func(i float64) float64 { return math.Pow(i, factor) }},
		{Type: interfaces.SineScaler, Factor: factor, Fn: func(i float64) float64 { return math.Sin(i / factor) }},
		{Type: interfaces.LogarithmicScaler, Factor: factor, Fn: func(i float64) float64 { return math.Log(i) * factor }},
	}
)

func TestScaler(t *testing.T) {
	defer goleak.VerifyNone(t)
	for _, c := range testCases {
		t.Run(c.Type.String(), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s := scaler{
				ctx:        ctx,
				scaler:     c.Type,
				increment:  0,
				interval:   0,
				min:        1,
				max:        1,
				factor:     c.Factor,
				workeropts: &interfaces.WorkerOpts{},
			}
			fn := s.getScalerFunc()
			s.increment = rand.Float64() * 1024
			r := fn()
			v := c.Fn(s.increment)
			if r != v {
				t.Logf("Expected %+v, Got %+v", v, r)
				t.FailNow()
			}
		})
	}
}

func BenchmarkScaler(b *testing.B) {
	for _, c := range testCases {
		b.Run(c.Type.String(), func(b *testing.B) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s := scaler{
				ctx:        ctx,
				scaler:     c.Type,
				increment:  0,
				interval:   0,
				min:        1,
				max:        1,
				factor:     c.Factor,
				workeropts: &interfaces.WorkerOpts{},
			}
			fn := s.getScalerFunc()
			for i := 0; i < b.N; i++ {
				s.increment++
				fn()
			}
		})
	}
}

func TestScalerStart(t *testing.T) {
	defer goleak.VerifyNone(t)
	c := testCases[rng.Intn(len(testCases))]
	t.Run(c.Type.String(), func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		s := scaler{
			ctx:        ctx,
			scaler:     c.Type,
			increment:  0,
			interval:   time.Second,
			min:        0,
			max:        0,
			factor:     c.Factor,
			workeropts: &interfaces.WorkerOpts{},
		}
		go func() {
			err := s.Start()
			if !errors.Is(err, context.Canceled) {
				t.Logf("Expected %+v, Got %+v", context.Canceled, err)
				t.Fail()
			}
		}()
		cancel()
		<-ctx.Done()
	})
}
