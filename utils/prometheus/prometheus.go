package prometheus

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

type metrics struct {
	RequestsTotal   prometheus.Counter
	RequestsFailed  prometheus.Counter
	RequestsError   prometheus.Counter
	RequestsBlength prometheus.Counter
	RequestsTime    prometheus.Summary
	RequestsCode    map[int]prometheus.Counter
	Workers         prometheus.Gauge
}

type metricValues struct {
	RequestsTotal   float64
	RequestsFailed  float64
	RequestsError   float64
	RequestsBlength float64
	RequestsTime    map[float64]float64
	RequestsCode    map[int]float64
	Workers         float64
}

var Metrics metrics

func (this *metrics) Get() metricValues {
	return metricValues{
		RequestsTotal:   *getCounterValue(this.RequestsTotal),
		RequestsFailed:  *getCounterValue(this.RequestsFailed),
		RequestsError:   *getCounterValue(this.RequestsError),
		RequestsBlength: *getCounterValue(this.RequestsBlength),
		RequestsTime:    getSummaryValue(this.RequestsTime),
		RequestsCode:    this.GetCodes(),
		Workers:         *getGaugeValue(this.Workers),
	}
}

func (this *metrics) GetCodes() map[int]float64 {
	r := make(map[int]float64)
	for i, c := range this.RequestsCode {
		r[i] = *getCounterValue(c)
	}
	return r
}

func (this *metrics) GetCodeCounter(code int) prometheus.Counter {
	if this.RequestsCode[code] == nil {
		this.RequestsCode[code] = promauto.NewCounter(prometheus.CounterOpts{
			Name:        "netbench_requests_code",
			Help:        "requests_code",
			ConstLabels: map[string]string{"code": strconv.Itoa(code)},
		})
	}
	return this.RequestsCode[code]
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

func getSummaryValue(s prometheus.Summary) map[float64]float64 {
	m := dto.Metric{}
	s.Write(&m)
	q := m.GetSummary().Quantile
	r := make(map[float64]float64)
	for _, v := range q {
		r[v.GetQuantile()] = v.GetValue()
	}
	return r
}

func init() {
	Metrics = metrics{
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
			Name: "netbench_requests_blength",
			Help: "requests_blength",
		}),
		RequestsTime: promauto.NewSummary(prometheus.SummaryOpts{
			Name:       "netbench_requests_time",
			Help:       "requests_time",
			Objectives: map[float64]float64{0: 1, 0.25: 0.075, 0.5: 0.05, 0.75: 0.025, 0.9: 0.01, 0.99: 0.001, 1: 0},
		}),
		RequestsCode: make(map[int]prometheus.Counter),
		Workers: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "netbench_workers",
			Help: "workers",
		}),
	}

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)
}
