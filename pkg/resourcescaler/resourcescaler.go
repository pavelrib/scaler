package main

import (
	"os"

	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
	"github.com/v3io/scaler-types"

	"github.com/nuclio/zap"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
)

type AppResourceScaler struct {
	logger        logger.Logger
	helmClient    *helm.Client
	kubeClientSet kubernetes.Interface
}

func New() scaler_types.ResourceScaler {
	helmClient := helm.NewClient()
	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "Failed getting cluster kube config")
	}
	rLogger, err := nucliozap.NewNuclioZap("autoscaler", "console", os.Stdout, os.Stderr, nucliozap.DebugLevel)


	return &AppResourceScaler{
		logger:        rLogger,
		helmClient:    helmClient,
		kubeClientSet: kubernetes.NewForConfig(kubeconfig),
	}
}

// if last int parameter is 0 -> helm del --purge. if 1 -> helm install
func (s *AppResourceScaler) SetScale(namespace string, resource scaler_types.Resource, scaling int) error {

	// get deployment by resource name
	deployment, err := s.kubeClientSet.AppsV1beta1().Deployments(namespace).Get(string(resource), meta_v1.GetOptions{})
	if err != nil {
		s.logger.WarnWith("Failure during retrieval of deployment", "resource_name", string(resource))
		return errors.Wrap(err, "Failed getting deployment instance")
	}

	int32scaling := int32(scaling)
	deployment.Spec.Replicas = &int32scaling
	_, err = s.kubeClientSet.AppsV1beta1().Deployments(namespace).Update(deployment)
	if err != nil {
		s.logger.WarnWith("Failure during update of deployment", "resource_name", string(resource))
		return errors.Wrap(err, "Failed updating deployment instance")
	}

	return nil
}

func (s *AppResourceScaler) GetResources(namespace string) ([]scaler_types.Resource, error) {
	resources := make([]scaler_types.Resource, 0)

	listNamespace := helm.ReleaseListNamespace(namespace)
	allStatuses := helm.ReleaseListStatuses(allReleaseStatuses())
	listedReleases, err := s.helmClient.ListReleases(listNamespace, allStatuses)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get listed releases")
	}

	// return the names of all releases
	for _, releaseInst := range listedReleases.GetReleases() {
		resources = append(resources, scaler_types.Resource(releaseInst.Name))
	}

	return resources, nil
}

func (s *AppResourceScaler) GetConfig() (*scaler_types.ResourceScalerConfig, error) {
	return nil, nil
}

func allReleaseStatuses() []release.Status_Code {
	statusCodes := make([]release.Status_Code, 0)
	for statusCodeValue := range release.Status_Code_name {
		statusCodes = append(statusCodes, release.Status_Code(statusCodeValue))
	}
	return statusCodes
}
