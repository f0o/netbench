package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"go.f0o.dev/netbench/interfaces"
	"go.f0o.dev/netbench/utils/logger"
	"go.f0o.dev/netbench/utils/prometheus"
	"go.f0o.dev/netbench/utils/worker"
)

var (
	flags                 interfaces.Flags = interfaces.Flags{}
	version, commit, date string           = "unknown", "unknown", "unknown"
)

func init() {
	runtime.GOMAXPROCS(1) // notably increases performance

	v := flag.Bool("version", false, "Print version and exit")

	flag.DurationVar(&flags.Duration, "duration", 15*time.Minute, "Duration of benchmark")
	flag.StringVar(&flags.Format, "format", "text", "Output format (text|json)")

	flag.BoolVar(&flags.PrometheusOpts.Enabled, "prometheus", false, "Enable Prometheus metrics server")
	flag.StringVar(&flags.PrometheusOpts.Bind, "prometheus-bind", "0.0.0.0:8080", "Address to bind Prometheus metrics server")

	flag.StringVar(&flags.WorkerOpts.Target, "target", "", fmt.Sprintf(`Target URI to benchmark (scheme://host[:port][/path])
Supported Schemes: %+v`, worker.AvailableWorkers()))
	flag.StringVar(&flags.WorkerOpts.Payload, "payload", "", "Optional base64 encoded payload to send to the target")

	flags.WorkerOpts.HTTPOpts.Headers = make(map[string]string)
	flag.StringVar(&flags.WorkerOpts.HTTPOpts.Method, "http-method", "GET", "HTTP Method to use")
	flag.Var(&flags.WorkerOpts.HTTPOpts.Headers, "http-header", "HTTP Headers to use (Header:Value)")
	flag.BoolVar(&flags.WorkerOpts.HTTPOpts.Follow, "http-follow", false, "Follow redirects")

	flag.DurationVar(&flags.WorkerOpts.NetOpts.Timeout, "net-timeout", 200*time.Millisecond, "Timeout for socket operations")

	flag.Var(&flags.ScalerOpts.Type, "scaler", `Scaler to use:
- 'curve' : adds workers in a power curve (x^y where x is the increment and y is the factor)
- 'exp'   : adds workers in a base-e exponential curve (e^x where x is the increment)
- 'linear': adds workers in a linear fashion
- 'log'   : adds workers in a natural logarithmic curve
- 'sine'  : adds and removes workers in a sine wave
- 'static': static number of workers
 (default curve)`)
	flag.DurationVar(&flags.ScalerOpts.Period, "scaler-period", time.Minute, "Time to wait between scaler adjustments")
	flag.Float64Var(&flags.ScalerOpts.Factor, "scaler-factor", 1.5, `Scaling factor for scalers:
- when using 'static' scaler uses this as the number of workers
- when using 'curve' scaler uses this as the exponent
- when using 'sine' scaler uses this as the frequency
- all other scalers use this as the multiplier
`)
	flag.IntVar(&flags.ScalerOpts.Min, "scaler-min", 0, "Minimum number of workers (does not apply to static scaler)")
	flag.IntVar(&flags.ScalerOpts.Max, "scaler-max", runtime.NumCPU()*2, "Maximum number of workers (does not apply to static scaler)")

	flag.Parse()
	if v != nil && *v {
		info, ok := debug.ReadBuildInfo()
		if ok {
			fmt.Printf("netbench %s (%s) built on %s with %s for %s/%s\n\n", version, commit[:7], date, info.GoVersion, runtime.GOOS, runtime.GOARCH)
			fmt.Printf("Using:\n")
			for _, dep := range info.Deps {
				fmt.Printf("  %s %s\n", dep.Path, dep.Version)
			}
		}
		os.Exit(0)
	}
	if flags.WorkerOpts.Target == "" {
		logger.Fatalw("Missing Target parameter, Check --help", "Flags", flags)
	}
	if flags.PrometheusOpts.Enabled {
		go prometheus.Start(flags.PrometheusOpts.Bind)
	}
}
