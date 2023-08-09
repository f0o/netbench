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
	Sync    bool
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

func (httpheaders *HTTPHeaders) String() string {
	return fmt.Sprintf("%v", *httpheaders)
}

func (httpheaders *HTTPHeaders) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid header: %s", value)
	}

	k := strings.TrimSpace(parts[0])
	v := strings.TrimSpace(parts[1])

	(*httpheaders)[k] = v

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
	Stop()
	Wait() chan struct{}
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

func (scalertype *ScalerType) String() string {
	switch *scalertype {
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

func (scalertype *ScalerType) Set(value string) error {
	switch value {
	case "curve":
		*scalertype = CurveScaler
	case "exp", "exponential":
		*scalertype = ExponentialScaler
	case "linear":
		*scalertype = LinearScaler
	case "log", "logarithmic":
		*scalertype = LogarithmicScaler
	case "sin", "sine":
		*scalertype = SineScaler
	case "static":
		*scalertype = StaticScaler
	default:
		return fmt.Errorf("invalid scaler type: %s", value)
	}

	return nil
}
