package interfaces

import (
	"fmt"
	"strings"
	"time"
)

type Flags struct {
	PrometheusOpts `json:"prometheus"`
	WorkerOpts     `json:"worker"`
	ScalerOpts     `json:"scaler"`

	Duration time.Duration `json:"duration"`
	Format   string        `json:"format"`
}

type ScalerOpts struct {
	Type     ScalerType    `json:"type"`
	Interval time.Duration `json:"interval"`
	Factor   float64       `json:"factor"`
	Min      int           `json:"min"`
	Max      int           `json:"max"`
}

type WorkerOpts struct {
	HTTPOpts `json:"http"`
	NetOpts  `json:"net"`

	Payload string `json:"payload"`
	Target  string `json:"target"`
	Sync    bool   `json:"sync"`
}

type NetOpts struct {
	Addr    string        `json:"address"`
	Type    string        `json:"type"`
	Timeout time.Duration `json:"timeout"`
}

type HTTPOpts struct {
	URL     string        `json:"url"`
	Method  string        `json:"method"`
	Headers HTTPHeaders   `json:"headers"`
	Follow  bool          `json:"follow"`
	Timeout time.Duration `json:"timeout"`
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
	Enabled   bool    `json:"enabled"`
	Bind      string  `json:"bind"`
	Tolerance float64 `json:"tolerance"`
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
