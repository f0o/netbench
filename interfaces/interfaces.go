package interfaces

import "time"

type Logger interface {
	Debug(string, ...interface{})
	Info(string, ...interface{})
	Warn(string, ...interface{})
	Error(string, ...interface{})
	Fatal(string, ...interface{})
	Child(...interface{}) Logger
}

type Flags struct {
	Curve    float64
	Interval time.Duration
	Target   string
	Duration time.Duration
}

type Worker interface {
	Do() error
}

type Scaler interface {
	Start() error
}

type Prometheus interface {
	Child(string) Prometheus
	NewCounter(string, ...map[string]string) Counter
	NewCounterFunc(string, func() float64)
	NewGauge(string, ...map[string]string) Gauge
	NewGaugeFunc(string, func() float64)
}

type Counter interface {
	Inc()
	Get() float64
}

type Gauge interface {
	Set(float64)
	Inc()
	Dec()
	Now()
}
