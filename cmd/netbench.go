package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/logger"
	"go.f0o.dev/netbench/utils/prometheus"
	"go.f0o.dev/netbench/utils/scaler"
)

var flags interfaces.Flags

func init() {
	flag.DurationVar(&flags.Duration, "duration", 15*time.Minute, "Duration of benchmark")

	flag.BoolVar(&flags.PrometheusOpts.Enabled, "prometheus-enable", false, "Enable Prometheus metrics server")
	flag.StringVar(&flags.PrometheusOpts.Bind, "prometheus-bind", ":8080", "Address to bind Prometheus metrics server")

	flags.WorkerOpts.Headers = make(map[string]string)
	flag.StringVar(&flags.WorkerOpts.URL, "http-url", "", "Target URL to benchmark")
	flag.StringVar(&flags.WorkerOpts.Method, "http-method", "GET", "HTTP Method to use")
	flag.Var(&flags.WorkerOpts.Headers, "http-header", "HTTP Headers to use")

	flag.StringVar(&flags.ScalerOpts.Type, "scaler-type", "curve", "Scaler to use")
	flag.DurationVar(&flags.ScalerOpts.Period, "scaler-period", time.Minute, "Time to wait between scaler adjustments")
	flag.Float64Var(&flags.ScalerOpts.Factor, "scaler-factor", 1.5, "Scaling factor different scalers")
	flag.IntVar(&flags.ScalerOpts.Min, "scaler-min", 0, "Minimum number of workers")
	flag.IntVar(&flags.ScalerOpts.Max, "scaler-max", runtime.NumCPU()*5, "Maximum number of workers")

	flag.Parse()
	if flags.WorkerOpts.URL == "" {
		logger.Fatalw("Missing Target parameter, Check --help", "Flags", flags)
	}
	if flags.PrometheusOpts.Enabled {
		go prometheus.Start(flags.PrometheusOpts.Bind)
	}
}

var ctx context.Context
var cancel context.CancelFunc

func signalHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	logger.Debug("signalHandler started")
	for sig := range c {
		if cancel != nil {
			logger.Warn("Signal %s Received; Canceling", sig)
			cancel()
		} else {
			logger.Fatal("Signal %s Received; Cancel is %+v - Exiting", sig, cancel)
		}
	}
}

func main() {
	logger.Infow("Starting netbench", "Flags", flags)
	go signalHandler()

	ctx, cancel = context.WithTimeout(context.WithValue(context.Background(), "flags", flags), flags.Duration)
	start := time.Now()
	go scaler.NewScaler(ctx).Start()
	<-ctx.Done()
	stop := time.Since(start)

	time.Sleep(time.Second)

	metrics := prometheus.Metrics.Get()

	fmt.Printf("Run Duration     : %+v\n", stop)
	fmt.Printf("Total Requests   : %+v\n", metrics.RequestsTotal)
	fmt.Printf("Failed Requests  : %+v\n", metrics.RequestsFailed)
	fmt.Printf("   Runtime Error : %+v\n", metrics.RequestsError)
	fmt.Printf("   Aborted       : %+v\n", metrics.RequestsAborted)
	fmt.Printf("   Body Length   : %+v\n", metrics.RequestsBlength)
	fmt.Printf("   Status Code   : %+v\n", metrics.RequestsCode)
	fmt.Printf("Average RPS      : %+v\n", metrics.RequestsTotal/stop.Seconds())
	fmt.Printf("Average Latency  : %+v\n", time.Duration(metrics.RequestsTime[0.5]))
	fmt.Printf("    Max Latency  : %+v\n", time.Duration(metrics.RequestsTime[1]))
	fmt.Printf("    99%% Latency  : %+v\n", time.Duration(metrics.RequestsTime[0.99]))
	fmt.Printf("    90%% Latency  : %+v\n", time.Duration(metrics.RequestsTime[0.9]))
	fmt.Printf("    75%% Latency  : %+v\n", time.Duration(metrics.RequestsTime[0.75]))
	fmt.Printf("    50%% Latency  : %+v\n", time.Duration(metrics.RequestsTime[0.5]))
	fmt.Printf("    25%% Latency  : %+v\n", time.Duration(metrics.RequestsTime[0.25]))
	fmt.Printf("    Min Latency  : %+v\n", time.Duration(metrics.RequestsTime[0]))
}
