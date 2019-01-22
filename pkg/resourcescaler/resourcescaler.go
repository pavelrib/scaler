package resourcescaler

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
	"github.com/v3io/scaler/pkg"

	"k8s.io/helm/pkg/helm"
	hapi_chart3 "k8s.io/helm/pkg/proto/hapi/chart"
)

type ResourceScaler struct {
	helmClient *helm.Client
	namespace  string
	releases   []*releaseData
}

type releaseData struct {
	chart *hapi_chart3.Chart
}

func New() *ResourceScaler {
	namespace := getenv("RESOURCE_NAMESPACE", "default-tenant")

	helmClient := helm.NewClient()

	return &ResourceScaler{
		helmClient: helmClient,
		namespace:  namespace,
		releases:   make([]*releaseData, 0),
	}
}

// if last int parameter is 0 -> helm del --purge. if 1 -> helm install
func (r *ResourceScaler) SetScale(logger logger.Logger, namespace string, resource scaler.Resource, scaling int) error {
	if scaling == 0 {
		if err := r.saveResourceData(logger, resource); err != nil {
			return errors.Wrap(err, "Failed setting resource")
		}

		deleteResponse, err := r.helmClient.DeleteRelease(string(resource))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to delete release %s", string(resource)))
		}
		logger.InfoWith("Release deleted successfully", "release_name", resource, "delete_response", deleteResponse)
	} else {
		//TBD
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

func (r *ResourceScaler) saveResourceData(logger logger.Logger, resource scaler.Resource) error {
	// clean previous data
	r.releases = r.releases[:0]

	// get required values
	listNamespace := helm.ReleaseListNamespace(r.namespace)
	listFilter := helm.ReleaseListFilter(string(resource))
	listedReleases, err := r.helmClient.ListReleases(listNamespace, listFilter)

	if err != nil {
		return errors.Wrap(err, "Failed to get listed releases")
	}

	for _, release := range listedReleases.GetReleases() {
		logger.DebugWith("Saving release data for later use", "release_name", release.Name)
		singleReleaseData := &releaseData{
			chart: release.Chart,
		}
		r.releases = append(r.releases, singleReleaseData)
	}

	return nil
}
