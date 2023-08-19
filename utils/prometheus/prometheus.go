package prometheus

// TODO: documentation

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"go.f0o.dev/netbench/utils/logger"
)

type metrics struct {
	mutex           sync.Mutex
	RequestsTotal   prometheus.Counter
	RequestsFailed  prometheus.Counter
	RequestsError   prometheus.Counter
	RequestsBlength prometheus.Counter
	RequestsAborted prometheus.Counter
	ResponseTimes   prometheus.Summary
	ResponseCodes   map[int]prometheus.Counter
	ResponseBytes   prometheus.Gauge
	Workers         prometheus.Gauge
	Start           time.Time
	Tolerance       float64
}

type MetricValues struct {
	RequestsTotal   float64            `json:"requests_total"`
	RequestsFailed  float64            `json:"requests_failed"`
	RequestsError   float64            `json:"requests_error"`
	RequestsBlength float64            `json:"requests_failed_bodylength"`
	RequestsAborted float64            `json:"requests_aborted"`
	ResponseTimes   map[string]float64 `json:"response_times"`
	ResponseCodes   map[string]float64 `json:"response_codes"`
	ResponseBytes   float64            `json:"response_bytes"`
	Workers         float64            `json:"workers"`
	Duration        time.Duration      `json:"duration"`
	RequestsPerSec  float64            `json:"requests_per_sec"`
	Tolerance       float64            `json:"tolerance"`
}

var Metrics metrics

var SkipSanityCheck bool = false

func (metrics *metrics) Get() MetricValues {
	d := time.Since(metrics.Start)
	m := MetricValues{
		RequestsTotal:   *getCounterValue(metrics.RequestsTotal),
		RequestsFailed:  *getCounterValue(metrics.RequestsFailed),
		RequestsError:   *getCounterValue(metrics.RequestsError),
		RequestsBlength: *getCounterValue(metrics.RequestsBlength),
		RequestsAborted: *getCounterValue(metrics.RequestsAborted),
		ResponseTimes:   getSummaryValue(metrics.ResponseTimes),
		ResponseCodes:   metrics.GetCodes(),
		ResponseBytes:   *getGaugeValue(metrics.ResponseBytes),
		Workers:         *getGaugeValue(metrics.Workers),
		Duration:        d,
		Tolerance:       metrics.Tolerance,
	}
	m.RequestsPerSec = m.RequestsTotal / m.Duration.Seconds()
	if !SkipSanityCheck {
		m.sanityCheck()
	}
	return m
}

func (metricvalues *MetricValues) sanityCheck() {
	var insane bool
	var diff float64
	total_2xx := 0.0
	total_nxx := 0.0
	for k, v := range metricvalues.ResponseCodes {
		i, _ := strconv.Atoi(k)
		if i >= 200 && i < 300 {
			total_2xx += v
		} else {
			total_nxx += v
		}
	}
	total_err := metricvalues.RequestsError + metricvalues.RequestsBlength + metricvalues.RequestsAborted + total_nxx
	tolerance := metricvalues.Tolerance * metricvalues.RequestsTotal
	if metricvalues.RequestsFailed != total_err {
		diff = math.Abs(metricvalues.RequestsFailed - total_err)
		if diff > tolerance {
			logger.Warnw("Total Failed Requests does not match", "Have", metricvalues.RequestsFailed, "Want", total_err, "Diff", diff, "Tolerance", metricvalues.Tolerance)
			insane = true
		} else {
			logger.Debugw("Total Failed Requests does not match but is within tolerance", "Have", metricvalues.RequestsFailed, "Want", total_err, "Diff", diff, "Tolerance", metricvalues.Tolerance)
		}
	}
	if metricvalues.RequestsTotal != total_err+total_2xx {
		diff = math.Abs(metricvalues.RequestsTotal - (total_err + total_2xx))
		if diff > tolerance {
			logger.Warnw("Total Requests does not match", "Have", metricvalues.RequestsTotal, "Want", total_err+total_2xx, "Diff", diff, "Tolerance", metricvalues.Tolerance)
			insane = true
		} else {
			logger.Debugw("Total Requests does not match but is within tolerance", "Have", metricvalues.RequestsTotal, "Want", total_err+total_2xx, "Diff", diff, "Tolerance", metricvalues.Tolerance)
		}

	}
	if insane {
		logger.Warn("Metrics are insane; This could be caused by extreme scaling or a bug in netbench. Interpreting results may be difficult.")
	}
}

func (metrics *metrics) GetCodes() map[string]float64 {
	r := make(map[string]float64)
	for i, c := range metrics.ResponseCodes {
		r[strconv.Itoa(i)] = *getCounterValue(c)
	}
	return r
}

func (metrics *metrics) GetCodeCounter(code int) prometheus.Counter {
	if metrics.ResponseCodes[code] == nil {
		metrics.mutex.Lock()
		defer metrics.mutex.Unlock()
		if metrics.ResponseCodes[code] == nil {
			metrics.ResponseCodes[code] = promauto.NewCounter(prometheus.CounterOpts{
				Name:        "netbench_response_codes",
				Help:        "requests_code",
				ConstLabels: map[string]string{"code": strconv.Itoa(code)},
			})
		}
	}
	return metrics.ResponseCodes[code]
}

func (metrics *metrics) SetTolerance(tolerance float64) {
	if tolerance < 0.0 {
		logger.Warnw("Tolerance less than 0.0 (=0%); setting to 0.0", "Tolerance", tolerance)
		tolerance = 0.0
	} else if tolerance >= 1.0 {
		logger.Warnw("Tolerance greater than or equal to 1.0 (=100%); disabling metrics sanity checks", "Tolerance", tolerance)
		SkipSanityCheck = true
		return
	}

	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()
	metrics.Tolerance = tolerance
}

func getGaugeValue(g prometheus.Gauge) *float64 {
	m := dto.Metric{}
	err := g.Write(&m)
	if err != nil {
		logger.Warnw("Could not get Metric", "Metric", g)
		return nil
	}
	return m.GetGauge().Value
}

func getCounterValue(c prometheus.Counter) *float64 {
	m := dto.Metric{}
	err := c.Write(&m)
	if err != nil {
		logger.Warnw("Could not get Metric", "Metric", c)
		return nil
	}
	return m.GetCounter().Value
}

func getSummaryValue(s prometheus.Summary) map[string]float64 {
	m := dto.Metric{}
	err := s.Write(&m)
	if err != nil {
		logger.Warnw("Could not get Metric", "Metric", s)
		return nil
	}
	q := m.GetSummary().Quantile
	r := make(map[string]float64)
	for _, v := range q {
		k := strconv.FormatFloat(v.GetQuantile(), 'f', -1, 64)
		vv := v.GetValue()
		if math.IsNaN(vv) {
			vv = -1
		}
		r[k] = vv
	}
	return r
}

func init() {
	Metrics = metrics{
		Start: time.Now(),
		RequestsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "netbench_requests_total",
			Help: "requests_total",
		}),
		RequestsFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "netbench_requests_failed",
			Help: "requests_failed",
		}),
		RequestsError: promauto.NewCounter(prometheus.CounterOpts{
			Name: "netbench_requests_error",
			Help: "requests_error",
		}),
		RequestsBlength: promauto.NewCounter(prometheus.CounterOpts{
			Name: "netbench_requests_failed_bodylength",
			Help: "requests_failed_bodylength",
		}),
		RequestsAborted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "netbench_requests_aborted",
			Help: "requests_aborted",
		}),
		ResponseTimes: promauto.NewSummary(prometheus.SummaryOpts{
			Name:       "netbench_response_times",
			Help:       "response_times",
			Objectives: map[float64]float64{0: 1, 0.25: 0.075, 0.5: 0.05, 0.75: 0.025, 0.9: 0.01, 0.99: 0.001, 1: 0},
		}),
		ResponseCodes: make(map[int]prometheus.Counter),
		ResponseBytes: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "netbench_response_bytes",
			Help: "response_bytes",
		}),
		Workers: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "netbench_workers",
			Help: "workers",
		}),
	}
}

func Start(bind string) {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(bind, nil)
	if err != nil {
		logger.Fatalw("Prometheus Error: %+v", err)
	}
}
