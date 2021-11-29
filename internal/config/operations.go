package config

import (
	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-adapter-library/meshes"
)

var (
	ServiceName = "service_name"
)

func getOperations(dev adapter.Operations) adapter.Operations {

	versions, _ := getLatestReleaseNames(3)

	dev[KumaOperation] = &adapter.Operation{
		Type:        int32(meshes.OpCategory_INSTALL),
		Description: "Kuma Service Mesh",
		Versions:    versions,
		Templates:   adapter.NoneTemplate,
		AdditionalProperties: map[string]string{
			ServiceName: KumaOperation,
		},
	}

	return dev
}
