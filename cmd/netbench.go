package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/logger"
	"go.f0o.dev/netbench/utils/prometheus"
	"go.f0o.dev/netbench/utils/scaler"
)

var flags interfaces.Flags

func init() {
	flag.Float64Var(&flags.Curve, "curve", 1.5, "The exponent to apply to f(x)=x^c where x is an increment (0=none,1=linear,2=exponential,...)")
	flag.DurationVar(&flags.Interval, "interval", time.Minute, "Seconds to wait between curve increment increase")
	flag.DurationVar(&flags.Duration, "duration", 15*time.Minute, "Duration of benchmark in seconds")
	flag.StringVar(&flags.Target, "target", "", "Target URL to benchmark")
	flag.Parse()
	if flags.Target == "" {
		fmt.Println("Missing Target parameter")
		flag.PrintDefaults()
		os.Exit(2)
	}
	logger.Debug("flags: %+v\n", flags)
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
			logger.Error("Signal %s Received; Cancel is %+v - Exiting", sig, cancel)
			os.Exit(7)
		}
	}
}

func main() {
	logger.Info("Starting netbench")

	ctx, cancel = context.WithTimeout(context.WithValue(context.Background(), "flags", flags), flags.Duration)
	go signalHandler()
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
