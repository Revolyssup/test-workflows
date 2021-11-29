package config

import (
	"strings"

	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-adapter-library/meshes"
	smp "github.com/layer5io/service-mesh-performance/spec"
)

var (
	OSMOperation          = strings.ToLower(smp.ServiceMesh_OPEN_SERVICE_MESH.Enum().String())
	OSMBookStoreOperation = "osm_bookstore_app"
	ServiceName           = "service_name"
)

func getOperations(op adapter.Operations) adapter.Operations {
	versions, _ := getLatestReleaseNames(3)

	op[OSMOperation] = &adapter.Operation{
		Type:        int32(meshes.OpCategory_INSTALL),
		Description: "Open Service Mesh",
		Versions:    versions,
		Templates: []adapter.Template{
			"templates/open_service_mesh.yaml",
		},
	}

	op[OSMBookStoreOperation] = &adapter.Operation{
		Type:        int32(meshes.OpCategory_SAMPLE_APPLICATION),
		Description: "Bookstore Application",
		Versions:    versions,
		Templates: []adapter.Template{
			"https://raw.githubusercontent.com/openservicemesh/osm/release-v0.6/docs/example/manifests/apps/bookbuyer.yaml",
			"https://raw.githubusercontent.com/openservicemesh/osm/release-v0.6/docs/example/manifests/apps/bookstore-v1.yaml",
			"https://raw.githubusercontent.com/openservicemesh/osm/release-v0.6/docs/example/manifests/apps/bookthief.yaml",
			"https://raw.githubusercontent.com/openservicemesh/osm/release-v0.6/docs/example/manifests/apps/bookwarehouse.yaml",
			"https://raw.githubusercontent.com/openservicemesh/osm/release-v0.6/docs/example/manifests/apps/traffic-split.yaml",
			"file://templates/osm-bookstore-traffic-access-v1.yaml",
		},
		AdditionalProperties: map[string]string{
			ServiceName: "bookstore",
		},
	}

	return op
}
