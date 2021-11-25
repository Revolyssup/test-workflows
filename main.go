// Copyright 2019 Layer5.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/layer5io/meshery-traefik-mesh/traefik"
	"github.com/layer5io/meshery-traefik-mesh/traefik/oam"
	"github.com/layer5io/meshkit/logger"
	"github.com/layer5io/meshkit/utils/manifests"
	"gopkg.in/yaml.v2"

	// "github.com/layer5io/meshkit/tracing"
	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-adapter-library/api/grpc"
	"github.com/layer5io/meshery-traefik-mesh/internal/config"
	configprovider "github.com/layer5io/meshkit/config/provider"
	smp "github.com/layer5io/service-mesh-performance/spec"
)

var (
	serviceName = "traefik-mesh-adaptor"
	version     = "none"
	gitsha      = "none"
)

func init() {
	// Create the config path if it doesn't exists as the entire adapter
	// expects that directory to exists, which may or may not be true
	if err := os.MkdirAll(path.Join(config.RootPath(), "bin"), 0750); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// main is the entrypoint of the adaptor
func main() {
	// Initialize Logger instance
	log, err := logger.New(serviceName, logger.Options{
		Format:     logger.SyslogLogFormat,
		DebugLevel: isDebug(),
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = os.Setenv("KUBECONFIG", path.Join(
		config.KubeConfig[configprovider.FilePath],
		fmt.Sprintf("%s.%s", config.KubeConfig[configprovider.FileName], config.KubeConfig[configprovider.FileType])),
	)

	if err != nil {
		// Fail silently
		log.Warn(err)
	}

	fmt.Println("HELOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO")

	// Initialize application specific configs and dependencies
	// App and request config
	cfg, err := config.New(configprovider.ViperKey)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	service := &grpc.Service{}
	err = cfg.GetObject(adapter.ServerKey, service)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	kubeconfigHandler, err := config.NewKubeconfigBuilder(configprovider.ViperKey)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// // Initialize Tracing instance
	// tracer, err := tracing.New(service.Name, service.TraceURL)
	// if err != nil {
	//      log.Err("Tracing Init Failed", err.Error())
	//      os.Exit(1)
	// }

	// Initialize Handler intance
	handler := traefik.New(cfg, log, kubeconfigHandler)
	handler = adapter.AddLogger(log, handler)

	service.Handler = handler
	service.Channel = make(chan interface{}, 10)
	service.StartedAt = time.Now()
	service.Version = version
	service.GitSHA = gitsha

	fmt.Println("HELOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO")
	go registerCapabilities(service.Port, log)        //Registering static capabilities
	go registerDynamicCapabilities(service.Port, log) //Registering latest capabilities periodically

	// Server Initialization
	log.Info("Adaptor Listening at port: ", service.Port)
	err = grpc.Start(service, nil)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func isDebug() bool {
	return os.Getenv("DEBUG") == "true"
}

func mesheryServerAddress() string {
	meshReg := os.Getenv("MESHERY_SERVER")

	if meshReg != "" {
		if strings.HasPrefix(meshReg, "http") {
			return meshReg
		}

		return "http://" + meshReg
	}

	return "http://localhost:9081"
}

func serviceAddress() string {
	svcAddr := os.Getenv("SERVICE_ADDR")

	if svcAddr != "" {
		return svcAddr
	}

	return "mesherylocal.layer5.io"
}

func registerCapabilities(port string, log logger.Handler) {
	log.Info("Registering static capabilities...")
	// Register workloads
	if err := oam.RegisterWorkloads(mesheryServerAddress(), serviceAddress()+":"+port); err != nil {
		log.Info(err.Error())
	}
	// Register traits
	if err := oam.RegisterTraits(mesheryServerAddress(), serviceAddress()+":"+port); err != nil {
		log.Info(err.Error())
	}
	log.Info("Registeration of static capabilities completed")
}
func registerDynamicCapabilities(port string, log logger.Handler) {
	registerWorkloads(port, log)
	//Start the ticker
	const reRegisterAfter = 24
	ticker := time.NewTicker(reRegisterAfter * time.Hour)
	for {
		<-ticker.C
		registerWorkloads(port, log)
	}

}

func registerWorkloads(port string, log logger.Handler) {
	appVersion, chartVersion, err := getLatestValidAppVersionAndChartVersion()
	if err != nil {
		log.Info(err)
		return
	}
	log.Info("Registering latest workload components for version ", appVersion, " and chart version ", chartVersion)
	// Register workloads
	if err := adapter.RegisterWorkLoadsDynamically(mesheryServerAddress(), serviceAddress()+":"+port, &adapter.DynamicComponentsConfig{
		TimeoutInMinutes: 60,
		URL:              "https://helm.traefik.io/traefik/traefik-" + chartVersion + ".tgz",
		GenerationMethod: adapter.HelmCHARTS,
		Config: manifests.Config{
			Name:        smp.ServiceMesh_Type_name[int32(smp.ServiceMesh_TRAEFIK_MESH)],
			MeshVersion: appVersion,
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
		},
		Operation: config.TraefikMeshOperation,
	}); err != nil {
		log.Info(err.Error())
		return
	}
	log.Info("Latest workload components successfully registered.")
}

// returns latest valid appversion and chartversion
func getLatestValidAppVersionAndChartVersion() (string, string, error) {
	res, err := http.Get("https://helm.traefik.io/traefik/index.yaml")
	if err != nil {
		return "", "", config.ErrGetLatestReleases(err)
	}
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", "", config.ErrGetLatestReleases(err)
	}
	var h helmIndex
	err = yaml.Unmarshal(content, &h)
	if err != nil {
		return "", "", config.ErrGetLatestReleases(err)
	}

	return h.Entries["traefik"][0].AppVersion, h.Entries["traefik"][0].Version, nil
}

// Below structs are helper structs to unmarshall and extract certain fields from helm chart's index.yaml
type helmIndex struct {
	Entries map[string][]data `yaml:"entries"`
}

type data struct {
	AppVersion string `yaml:"appVersion"`
	Version    string `yaml:"version"`
}
