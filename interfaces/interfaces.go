package interfaces

import (
	"fmt"
	"strings"
	"time"
)

type Flags struct {
	PrometheusOpts
	WorkerOpts
	ScalerOpts

	Duration time.Duration
	Format   string
}

type ScalerOpts struct {
	Type   string
	Period time.Duration
	Factor float64
	Min    int
	Max    int
}

type WorkerOpts struct {
	URL     string
	Method  string
	Headers HTTPHeaders
}

type HTTPHeaders map[string]string

func (this *HTTPHeaders) String() string {
	return fmt.Sprintf("%v", *this)
}

func (this *HTTPHeaders) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid header: %s", value)
	}

	k := strings.TrimSpace(parts[0])
	v := strings.TrimSpace(parts[1])

	(*this)[k] = v

	return nil
}

type PrometheusOpts struct {
	Enabled bool
	Bind    string
}

type Worker interface {
	Do() error
}

type Scaler interface {
	Start() error
}
