package scaler

import (
	"context"
	"math"
	"math/rand"
	"testing"
	"time"

	"go.f0o.dev/netbench/interfaces"
)

func TestScaler(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	factor := rng.Float64()
	testCases := []struct {
		Type   string
		Factor float64
		Fn     func(float64) float64
	}{
		{Type: "static", Factor: factor, Fn: func(i float64) float64 { return factor }},
		{Type: "linear", Factor: factor, Fn: func(i float64) float64 { return i * factor }},
		{Type: "exponential", Factor: factor, Fn: func(i float64) float64 { return math.Exp(i) * factor }},
		{Type: "curve", Factor: factor, Fn: func(i float64) float64 { return math.Pow(i, factor) }},
		{Type: "sine", Factor: factor, Fn: func(i float64) float64 { return math.Sin(i / factor) }},
		{Type: "logarithmic", Factor: factor, Fn: func(i float64) float64 { return math.Log(i) * factor }},
	}
	for _, c := range testCases {
		t.Run(c.Type, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
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
			fn := s.setScalerFunc()
			for i := 0.0; i < 10; i++ {
				s.increment++
				r := fn()
				v := c.Fn(s.increment)
				if r != v {
					t.Logf("Expected %+v, Got %+v", v, r)
					t.FailNow()
				}
			}
			defer cancel()
		})
	}
}
