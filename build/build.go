package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/layer5io/meshery-adapter-library/adapter"
	smp "github.com/layer5io/service-mesh-performance/spec"

	"github.com/layer5io/meshkit/models/oam/core/v1alpha1"
	"github.com/layer5io/meshkit/utils"
	"github.com/layer5io/meshkit/utils/manifests"
)

var DefaultGenerationMethod string
var DefaultGenerationURL string
var LatestVersion string
var workloadPath string

//Should stay here
func NewConfig(version string) manifests.Config {
	return manifests.Config{
		Name:        smp.ServiceMesh_Type_name[int32(smp.ServiceMesh_ISTIO)],
		MeshVersion: version,
		Filter: manifests.CrdFilter{
			RootFilter:    []string{"$[?(@.kind==\"CustomResourceDefinition\")]"},
			NameFilter:    []string{"$..[\"spec\"][\"names\"][\"kind\"]"},
			VersionFilter: []string{"$[0]..spec.versions[0]"},
			GroupFilter:   []string{"$[0]..spec"},
			SpecFilter:    []string{"$[0]..openAPIV3Schema.properties.spec"},
			ItrFilter:     []string{"$[?(@.spec.names.kind"},
			ItrSpecFilter: []string{"$[?(@.spec.names.kind"},
			VField:        "name",
			GField:        "group",
		},
	}
}

//in library
type StaticCompConfig struct {
	URL     string
	Method  string //Use the constants exported by package
	Path    string
	DirName string
	Config  manifests.Config
	Force   bool
}

//in library
func CreateComponents(scfg StaticCompConfig) error {
	dir := filepath.Join(scfg.Path, scfg.DirName)
	_, err := os.Stat(dir)
	if err == nil {
		if !scfg.Force {
			fmt.Println("Skipping...")
			return nil
		} else {
			err := os.RemoveAll(filepath.Join(dir, "/**"))
			if err != nil {
				return err
			}
		}
	}
	if !os.IsNotExist(err) {
		return err
	}
	var comp *manifests.Component
	switch scfg.Method {
	case adapter.Manifests:
		comp, err = manifests.GetFromManifest(scfg.URL, manifests.SERVICE_MESH, scfg.Config)
	case adapter.HelmCHARTS:
		comp, err = manifests.GetFromHelm(scfg.URL, manifests.SERVICE_MESH, scfg.Config)
	default:
		return errors.New("Invalid Method: " + scfg.Method)
	}
	if err != nil {
		return errors.New("nil components: " + err.Error())
	}
	err = os.Mkdir(dir, 0777)
	if err != nil {
		return err
	}
	for i, def := range comp.Definitions {
		schema := comp.Schemas[i]
		name := GetNameFromWorkloadDefinition([]byte(def))
		defFileName := name + "_definition.json"
		schemaFileName := name + ".meshery.layer5io.schema.json"
		err := writeToFile(filepath.Join(dir, defFileName), []byte(def))
		if err != nil {
			return err
		}
		err = writeToFile(filepath.Join(dir, schemaFileName), []byte(schema))
		if err != nil {
			return err
		}
	}
	return nil
}

//create a file with this filename and stuff the string
func writeToFile(path string, data []byte) error {
	_, err := os.Create(path)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0666)
}

func GetNameFromWorkloadDefinition(definition []byte) string {
	var wd v1alpha1.WorkloadDefinition
	err := json.Unmarshal(definition, &wd)
	if err != nil {
		return ""
	}
	return wd.Spec.DefinitionRef.Name
}

func init() {
	wd, _ := os.Getwd()
	workloadPath = filepath.Join(filepath.Clean(filepath.Join(wd, "..")), "templates", "oam", "workloads")
	versions, _ := utils.GetLatestReleaseTagsSorted("istio", "istio")
	LatestVersion = versions[len(versions)-1]
	DefaultGenerationMethod = adapter.Manifests
	DefaultGenerationURL = "https://raw.githubusercontent.com/istio/istio/" + LatestVersion + "/manifests/charts/base/crds/crd-all.gen.yaml"
}
