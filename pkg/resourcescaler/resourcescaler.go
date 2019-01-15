package resourcescaler

import (
	"os"
	"strconv"
	"time"

	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
	"github.com/v3io/scaler/pkg"

	"k8s.io/helm/pkg/helm"
)

type ResourceScaler struct {
	helmClient *helm.Client
	namespace  string
}

func New() *ResourceScaler {
	namespace := getenv("RESOURCE_NAMESPACE", "default-tenant")

	helmClient := helm.NewClient()

	return &ResourceScaler{
		helmClient: helmClient,
		namespace:  namespace,
	}
}

func (r *ResourceScaler) SetScale(logger.Logger, string, scaler.Resource, int) error {
	// if last int parameter is 0 -> helm del --purge. if 1 -> helm install
	resources, _ := r.GetResources()

	listNamespace := helm.ReleaseListNamespace(r.namespace)
	listFilter := helm.ReleaseListFilter(string(resources[0]))
	listedReleases, err := r.helmClient.ListReleases(listNamespace, listFilter)
	if err != nil {
		return errors.Wrap(err, "Failed to get listed releases")
	}

	return nil
}

func (r *ResourceScaler) GetResources() ([]scaler.Resource, error) {
	return []scaler.Resource{"jupyter"}, nil
}

func (r *ResourceScaler) GetConfig() (*scaler.ResourceScalerConfig, error) {

	// Autoscaler options definition
	scaleInterval, err := time.ParseDuration(os.Getenv("AUTOSCALER_SCALE_INTERVAL"))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse autoscaler scale interval")
	}

	scaleWindow, err := time.ParseDuration(os.Getenv("AUTOSCALER_SCALE_WINDOW"))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse autoscaler scale window")
	}

	threshold, err := strconv.Atoi(os.Getenv("AUTOSCALER_THRESHOLD"))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse autoscaler threshold")
	}

	autoscalerOptions := scaler.AutoScalerOptions{
		Namespace:     r.namespace,
		ScaleInterval: scaleInterval,
		ScaleWindow:   scaleWindow,
		MetricName:    os.Getenv("AUTOSCALER_METRIC_NAME"),
		Threshold:     int64(threshold),
	}

	// Poller options definitions
	pollerMetricInterval, err := time.ParseDuration(os.Getenv("POLLER_METRIC_INTERVAL"))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse poller metric interval")
	}

	pollerOptions := scaler.PollerOptions{
		MetricInterval: pollerMetricInterval,
		MetricName:     os.Getenv("POLLER_METRIC_NAME"),
		Namespace:      r.namespace,
	}

	// DLX options definition
	targetPort, err := strconv.Atoi(os.Getenv("DLX_TARGET_PORT"))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse dlx target port")
	}

	dlxOptions := scaler.DLXOptions{
		Namespace:        r.namespace,
		TargetNameHeader: os.Getenv("DLX_TARGET_NAME_HEADER"),
		TargetPathHeader: os.Getenv("DLX_TARGET_PATH_HEADER"),
		TargetPort:       targetPort,
		ListenAddress:    os.Getenv("DLX_LISTEN_ADDRESS"),
	}

	// Now combine everything
	return &scaler.ResourceScalerConfig{
		KubeconfigPath:    os.Getenv("KUBECONFIG_PATH"),
		AutoScalerOptions: autoscalerOptions,
		PollerOptions:     pollerOptions,
		DLXOptions:        dlxOptions,
	}, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
