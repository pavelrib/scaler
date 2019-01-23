package resourcescaler

import (
	"fmt"

	"github.com/nuclio/errors"
	"github.com/nuclio/logger"
	"github.com/v3io/scaler/pkg"
	"k8s.io/helm/pkg/helm"
	hapi_chart3 "k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
)

type AppResourceScaler struct {
	helmClient *helm.Client
	releases   []*releaseData
}

type releaseData struct {
	chart *hapi_chart3.Chart
}

func New() scaler.ResourceScaler {
	helmClient := helm.NewClient()

	return &AppResourceScaler{
		helmClient: helmClient,
		releases:   make([]*releaseData, 0),
	}
}

// if last int parameter is 0 -> helm del --purge. if 1 -> helm install
func (r *AppResourceScaler) SetScale(logger logger.Logger, namespace string, resource scaler.Resource, scaling int) error {
	if scaling == 0 {

		// delete resource, but don't purge it, for future reference
		deleteResponse, err := r.helmClient.DeleteRelease(string(resource))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to delete release %s", string(resource)))
		}
		logger.InfoWith("Release deleted successfully",
			"release_name", resource, "delete_response", deleteResponse)
	} else {

		// list deleted releases of resource name
		listNamespace := helm.ReleaseListNamespace(namespace)
		listDeleted := helm.ReleaseListStatuses([]release.Status_Code{release.Status_DELETED, release.Status_DELETING})
		listName := helm.ReleaseListFilter(string(resource))
		listedReleases, err := r.helmClient.ListReleases(listNamespace, listDeleted, listName)
		if err != nil {
			logger.WarnWith("Failed while retrieving deleted releases for resource",
				"resource_name", string(resource))
			return errors.Wrap(err,"Failed to list deleted releases for resource")
		}

		// there should be exactly one release with this name
		releases := listedReleases.GetReleases()
		if len(releases) == 0 {
			logger.WarnWith("Could not find deleted releases for resource", "resource_name", string(resource))
			return errors.New("No deleted releases found, nothing to scale up")
		}

		if len(releases) > 1 {
			releasesNames := make([]string, 0)
			for _, releaseInst := range releases {
				releasesNames = append(releasesNames, releaseInst.Name)
			}
			logger.WarnWith("Find too many releases with same name", "resources_names", releasesNames)
			return errors.New("Too many releases to scale")
		}

		// found a single release - get its chart name/version
		releaseChart := releases[0].GetChart()
		logger.DebugWith("Retrieved chart of a deleted release",
			"chart_version", releaseChart.GetMetadata().GetName())

		// run a helm install command
		_, err = r.helmClient.InstallReleaseFromChart(releaseChart, namespace, helm.InstallReuseName(true))
		if err != nil {
			logger.WarnWith("Failed while re-installing release from chart", "release_name", string(resource))
			return errors.Wrap(err, "Failed while reinstalling")
		}
		logger.InfoWith("Release was re-installed successfully", "release_name", string(resource))
	}

	return nil
}

func (r *AppResourceScaler) GetResources(namespace string) ([]scaler.Resource, error) {
	resources := make([]scaler.Resource, 0)

	listNamespace := helm.ReleaseListNamespace(namespace)
	allStatuses := helm.ReleaseListStatuses(allReleaseStatuses())
	listedReleases, err := r.helmClient.ListReleases(listNamespace, allStatuses)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get listed releases")
	}

	// return the names of all releases
	for _, releaseInst := range listedReleases.GetReleases() {
		resources = append(resources, scaler.Resource(releaseInst.Name))
	}

	return resources, nil
}

func (r *AppResourceScaler) GetConfig() (*scaler.ResourceScalerConfig, error) {
	return nil, nil
}

func allReleaseStatuses() []release.Status_Code {
	statusCodes := make([]release.Status_Code, 0)
	for statusCodeValue := range release.Status_Code_name {
		statusCodes = append(statusCodes, release.Status_Code(statusCodeValue))
	}
	return statusCodes
}
