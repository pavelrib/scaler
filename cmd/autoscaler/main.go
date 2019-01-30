package main

import (
	"flag"
	"os"
	"time"

	"github.com/v3io/scaler/cmd/autoscaler/app"
	"github.com/v3io/scaler/pkg/common"

	"github.com/nuclio/errors"
)

func main() {
	kubeconfigPath := flag.String("kubeconfig-path", os.Getenv("KUBECONFIG"), "Path of kubeconfig file")
	namespace := flag.String("namespace", "", "Namespace to listen on, or * for all")
	scaleInterval := flag.Duration("scale-interval", time.Minute, "Interval to call check scale function")
	scaleWindow := flag.Duration("scale-window", time.Minute, "Time window after initial value to act upon")
	metricsInterval := flag.Duration("metrics-poll-interval", time.Minute, "Interval to poll custom metrics")
	metricsGroupKind := flag.String("metrics-group-kind", "", "Metrics resource kind")
	metricName := flag.String("metric-name", "", "Metric name from custom metrics")
	scaleThreshold := flag.Int64("scale-threshold", 0, "Maximum allowed value for metric to be considered below active")
	flag.Parse()

	*namespace = common.GetNamespace(*namespace)

	if err := app.Run(*kubeconfigPath,
		*namespace,
		*scaleInterval,
		*scaleWindow,
		*metricName,
		*scaleThreshold,
		*metricsInterval,
		*metricsGroupKind); err != nil {
		errors.PrintErrorStack(os.Stderr, err, 5)

		os.Exit(1)
	}
}
