package prometheus

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
}

var Metrics metrics

func (metrics *metrics) Get() MetricValues {
	d := time.Since(metrics.Start)
	logger.Debug("Flushing Metrics")
	time.Sleep(150 * time.Millisecond)
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
	}
	m.RequestsPerSec = m.RequestsTotal / m.Duration.Seconds()

	m.sanityCheck()

	return m
}

func (metricvalues *MetricValues) sanityCheck() {
	total_err := metricvalues.RequestsError + metricvalues.RequestsBlength + metricvalues.RequestsAborted
	total_2xx := 0.0
	for i := 200; i < 300; i++ {
		total_2xx += metricvalues.ResponseCodes[strconv.Itoa(i)]
	}

	if metricvalues.RequestsFailed != total_err {
		logger.Warnw("Total Failed Requests does not match", "Total Failed", metricvalues.RequestsFailed, "Total Errors", total_err)
	}

	if metricvalues.RequestsTotal != metricvalues.RequestsFailed+total_2xx {
		logger.Warnw("Total Requests does not match", "Total Requests", metricvalues.RequestsTotal, "Total Failed", total_err, "Total 2xx", total_2xx)
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

func getGaugeValue(g prometheus.Gauge) *float64 {
	m := dto.Metric{}
	g.Write(&m)
	return m.GetGauge().Value
}

func getCounterValue(c prometheus.Counter) *float64 {
	m := dto.Metric{}
	c.Write(&m)
	return m.GetCounter().Value
}

func getSummaryValue(s prometheus.Summary) map[string]float64 {
	m := dto.Metric{}
	s.Write(&m)
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

func Start(bind string) error {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(bind, nil)
	if err != nil {
		logger.Fatalw("Prometheus Error: %+v", err)
	}
	return err
}
