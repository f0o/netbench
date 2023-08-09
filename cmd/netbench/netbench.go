package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"go.f0o.dev/netbench/utils/logger"
	"go.f0o.dev/netbench/utils/prometheus"
	"go.f0o.dev/netbench/utils/scaler"
	"go.uber.org/automaxprocs/maxprocs"
)

var ctx context.Context
var cancel context.CancelFunc

func init() {
	maxprocs.Set(maxprocs.Logger(logger.Debug))
}

func signalHandler() {
	c := make(chan os.Signal, 1)
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
	fmt.Printf("Response Bytes   : %+v\n", metrics.ResponseBytes)
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
	scaler := scaler.NewScaler(ctx, &flags.ScalerOpts, &flags.WorkerOpts)
	go scaler.Start()
	<-ctx.Done()
	<-scaler.Wait()

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
