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
	Type   ScalerType
	Period time.Duration
	Factor float64
	Min    int
	Max    int
}

type WorkerOpts struct {
	HTTPOpts
	NetOpts

	Payload string
	Target  string
}

type NetOpts struct {
	Addr    string
	Type    string
	Timeout time.Duration
}

type HTTPOpts struct {
	URL     string
	Method  string
	Headers HTTPHeaders
	Follow  bool
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

type ScalerType int

const (
	CurveScaler ScalerType = iota
	ExponentialScaler
	LinearScaler
	LogarithmicScaler
	SineScaler
	StaticScaler
)

func (this *ScalerType) String() string {
	switch *this {
	case CurveScaler:
		return "curve"
	case ExponentialScaler:
		return "exponential"
	case LinearScaler:
		return "linear"
	case LogarithmicScaler:
		return "logarithmic"
	case SineScaler:
		return "sine"
	case StaticScaler:
		return "static"
	}
	return "unknown"
}

func (this *ScalerType) Set(value string) error {
	switch value {
	case "curve":
		*this = CurveScaler
	case "exp", "exponential":
		*this = ExponentialScaler
	case "linear":
		*this = LinearScaler
	case "log", "logarithmic":
		*this = LogarithmicScaler
	case "sin", "sine":
		*this = SineScaler
	case "static":
		*this = StaticScaler
	default:
		return fmt.Errorf("invalid scaler type: %s", value)
	}

	return nil
}
