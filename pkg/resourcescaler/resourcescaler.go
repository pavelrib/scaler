package resourcescaler

import (
	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
	"github.com/v3io/scaler/pkg"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
)

type AppResourceScaler struct {
	helmClient    *helm.Client
	kubeClientSet kubernetes.Interface
}

func New() scaler.ResourceScaler {
	helmClient := helm.NewClient()

	return &AppResourceScaler{
		helmClient: helmClient,
		kubeClientSet: kubernetes.NewForConfig(kubeconfig),
	}
}

// if last int parameter is 0 -> helm del --purge. if 1 -> helm install
func (s *AppResourceScaler) SetScale(logger logger.Logger, namespace string, resource scaler.Resource, scaling int) error {
	// get deployment by resource name
	deployment, err := s.kubeClientSet.AppsV1beta1().Deployments(namespace).Get(string(resource), meta_v1.GetOptions{})
	if err != nil {
		logger.WarnWith("Failure during retrieval of deployment", "resource_name", string(resource))
		return errors.Wrap(err, "Failed getting deployment instance")
	}

	int32scaling := int32(scaling)
	deployment.Spec.Replicas = &int32scaling
	_, err  = s.kubeClientSet.AppsV1beta1().Deployments(namespace).Update(deployment)
	if err != nil {
		logger.WarnWith("Failure during update of deployment", "resource_name", string(resource))
		return errors.Wrap(err, "Failed updating deployment instance")
	}

	return nil
}

func (s *AppResourceScaler) GetResources(namespace string) ([]scaler.Resource, error) {
	resources := make([]scaler.Resource, 0)

	listNamespace := helm.ReleaseListNamespace(namespace)
	allStatuses := helm.ReleaseListStatuses(allReleaseStatuses())
	listedReleases, err := s.helmClient.ListReleases(listNamespace, allStatuses)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get listed releases")
	}

	// return the names of all releases
	for _, releaseInst := range listedReleases.GetReleases() {
		resources = append(resources, scaler.Resource(releaseInst.Name))
	}

	return resources, nil
}

func (s *AppResourceScaler) GetConfig() (*scaler.ResourceScalerConfig, error) {
	return nil, nil
}

func allReleaseStatuses() []release.Status_Code {
	statusCodes := make([]release.Status_Code, 0)
	for statusCodeValue := range release.Status_Code_name {
		statusCodes = append(statusCodes, release.Status_Code(statusCodeValue))
	}
	return statusCodes
}
