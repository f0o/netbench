package interfaces

import "time"

type Flags struct {
	Curve      float64
	Interval   time.Duration
	Target     string
	Duration   time.Duration
	Prometheus struct {
		Bind   string
		Enable bool
	}
}

type Worker interface {
	Do() error
}

type Scaler interface {
	Start() error
}
