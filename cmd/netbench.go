package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"time"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/logger"
	"go.f0o.dev/netbench/utils/prometheus"
	"go.f0o.dev/netbench/utils/scaler"
)

var (
	flags                 interfaces.Flags
	version, commit, date string
)

func init() {
	v := flag.Bool("version", false, "Print version and exit")

	flag.DurationVar(&flags.Duration, "duration", 15*time.Minute, "Duration of benchmark")
	flag.StringVar(&flags.Format, "format", "text", "Output format (text|json)")

	flag.BoolVar(&flags.PrometheusOpts.Enabled, "prometheus-enable", false, "Enable Prometheus metrics server")
	flag.StringVar(&flags.PrometheusOpts.Bind, "prometheus-bind", ":8080", "Address to bind Prometheus metrics server")

	flags.WorkerOpts.Headers = make(map[string]string)
	flag.StringVar(&flags.WorkerOpts.URL, "http-url", "", "Target URL to benchmark")
	flag.StringVar(&flags.WorkerOpts.Method, "http-method", "GET", "HTTP Method to use")
	flag.Var(&flags.WorkerOpts.Headers, "http-header", "HTTP Headers to use")
	flag.BoolVar(&flags.WorkerOpts.Follow, "http-follow", false, "Follow redirects")

	flag.StringVar(&flags.ScalerOpts.Type, "scaler-type", "curve", `Scaler to use:
- 'curve': adds workers in a power curve (x^y where x is the increment and y is the factor)
- 'exp[onential]': adds workers in a base-e exponential curve (e^x where x is the increment)
- 'linear': adds workers in a linear fashion
- 'log[arithmic]': adds workers in a natural logarithmic curve
- 'sin[e]': adds and removes workers in a sine wave
- 'static': static number of workers`)
	flag.DurationVar(&flags.ScalerOpts.Period, "scaler-period", time.Minute, "Time to wait between scaler adjustments")
	flag.Float64Var(&flags.ScalerOpts.Factor, "scaler-factor", 1.5, `Scaling factor for scalers:
- 'static' scaler uses this as the number of workers
- 'curve' scaler uses this as the exponent
- 'sine' scaler uses this as the frequency
- all other scalers use this as the multiplier`)
	flag.IntVar(&flags.ScalerOpts.Min, "scaler-min", 0, "Minimum number of workers (does not apply to static scaler)")
	flag.IntVar(&flags.ScalerOpts.Max, "scaler-max", runtime.NumCPU()*5, "Maximum number of workers (does not apply to static scaler)")

	flag.Parse()
	if v != nil && *v {
		info, ok := debug.ReadBuildInfo()
		if ok && version != "" {
			fmt.Printf("netbench %s (%s) built on %s with %s for %s/%s\n\n", version, commit[:7], date, info.GoVersion, runtime.GOOS, runtime.GOARCH)
			fmt.Printf("Using:\n")
			for _, dep := range info.Deps {
				fmt.Printf("  %s %s\n", dep.Path, dep.Version)
			}
		} else {
			fmt.Printf("netbench %s built for %s/%s\n", "unknown", runtime.GOOS, runtime.GOARCH)
		}
		os.Exit(0)
	}
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

func outputText(metrics prometheus.MetricValues) {
	fmt.Printf("Run Duration     : %+v\n", metrics.Duration)
	fmt.Printf("Total Requests   : %+v\n", metrics.RequestsTotal)
	fmt.Printf("Failed Requests  : %+v\n", metrics.RequestsFailed)
	fmt.Printf("   Runtime Error : %+v\n", metrics.RequestsError)
	fmt.Printf("   Aborted       : %+v\n", metrics.RequestsAborted)
	fmt.Printf("   Body Length   : %+v\n", metrics.RequestsBlength)
	fmt.Println("Status Codes")
	for code, count := range metrics.ResponseCodes {
		fmt.Printf("   %+v           : %+v\n", code, count)
	}
	fmt.Printf("Average Req/Sec  : %+v\n", metrics.RequestsPerSec)
	fmt.Printf("Average Latency  : %+v\n", time.Duration(metrics.ResponseTimes["0.5"]))
	fmt.Printf("    Max Latency  : %+v\n", time.Duration(metrics.ResponseTimes["1"]))
	fmt.Printf("    99%% Latency  : %+v\n", time.Duration(metrics.ResponseTimes["0.99"]))
	fmt.Printf("    90%% Latency  : %+v\n", time.Duration(metrics.ResponseTimes["0.9"]))
	fmt.Printf("    75%% Latency  : %+v\n", time.Duration(metrics.ResponseTimes["0.75"]))
	fmt.Printf("    50%% Latency  : %+v\n", time.Duration(metrics.ResponseTimes["0.5"]))
	fmt.Printf("    25%% Latency  : %+v\n", time.Duration(metrics.ResponseTimes["0.25"]))
	fmt.Printf("    Min Latency  : %+v\n", time.Duration(metrics.ResponseTimes["0"]))
}

func main() {
	logger.Info("Starting netbench")
	logger.Debugw("Starting with configuration", "Flags", flags)
	go signalHandler()

	ctx, cancel = context.WithTimeout(context.Background(), flags.Duration)
	go scaler.NewScaler(ctx, &flags.ScalerOpts, &flags.WorkerOpts).Start()
	<-ctx.Done()

	metrics := prometheus.Metrics.Get()

	switch flags.Format {
	case "text":
		outputText(metrics)
	case "json":
		json_metrics, err := json.Marshal(metrics)
		if err != nil {
			logger.Fatalw("Failed to marshal metrics", "Error", err)
		}
		fmt.Printf("%+v\n", string(json_metrics))
	}
}
