// Package traefik - Common operations for the adapter
package traefik

import (
	"context"
	"fmt"

	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-adapter-library/common"
	meshkitCfg "github.com/layer5io/meshkit/config"
	"github.com/layer5io/meshery-adapter-library/status"
	internalConfig "github.com/layer5io/meshery-traefik-mesh/internal/config"
	"github.com/layer5io/meshery-traefik-mesh/traefik/oam"
	"github.com/layer5io/meshkit/logger"
	"github.com/layer5io/meshkit/models/oam/core/v1alpha1"
)

const (
	// SMIManifest is the manifest.yaml file for smi conformance tool
	SMIManifest = "https://raw.githubusercontent.com/layer5io/learn-layer5/master/smi-conformance/manifest.yml"
)

// Mesh represents the traefik-mesh adapter and embeds adapter.Adapter
type Mesh struct {
	adapter.Adapter // Type Embedded
}

// New initializes treafik-mesh handler.
func New(c meshkitCfg.Handler, l logger.Handler, kc meshkitCfg.Handler) adapter.Handler {
	return &Mesh{
		Adapter: adapter.Adapter{
			Config:            c,
			Log:               l,
			KubeconfigHandler: kc,
		},
	}
}

// ApplyOperation applies the operation on traefik mesh
func (mesh *Mesh) ApplyOperation(ctx context.Context, opReq adapter.OperationRequest) error {
	operations := make(adapter.Operations)
	err := mesh.Config.GetObject(adapter.OperationsKey, &operations)
	if err != nil {
		return err
	}

	e := &adapter.Event{
		Operationid: opReq.OperationID,
		Summary:     status.Deploying,
		Details:     "Operation is not supported",
	}

	switch opReq.OperationName {
	case internalConfig.TraefikMeshOperation:
		go func(hh *Mesh, ee *adapter.Event) {
			version := string(operations[opReq.OperationName].Versions[0])
			stat, err := hh.installTraefikMesh(opReq.IsDeleteOperation, version, opReq.Namespace)
			if err != nil {
				e.Summary = fmt.Sprintf("Error while %s Traefik service mesh", stat)
				e.Details = err.Error()
				hh.StreamErr(e, err)
				return
			}
			ee.Summary = fmt.Sprintf("Traefik service mesh %s successfully", stat)
			ee.Details = fmt.Sprintf("The Traefik service mesh is now %s.", stat)
			hh.StreamInfo(e)
		}(mesh, e)
	case common.BookInfoOperation, common.HTTPBinOperation, common.ImageHubOperation, common.EmojiVotoOperation:
		go func(hh *Mesh, ee *adapter.Event) {
			appName := operations[opReq.OperationName].AdditionalProperties[common.ServiceName]
			stat, err := hh.installSampleApp(opReq.Namespace, opReq.IsDeleteOperation, operations[opReq.OperationName].Templates)
			if err != nil {
				e.Summary = fmt.Sprintf("Error while %s %s application", stat, appName)
				e.Details = err.Error()
				hh.StreamErr(e, err)
				return
			}
			ee.Summary = fmt.Sprintf("%s application %s successfully", appName, stat)
			ee.Details = fmt.Sprintf("The %s application is now %s.", appName, stat)
			hh.StreamInfo(e)
		}(mesh, e)
	case common.CustomOperation:
		go func(hh *Mesh, ee *adapter.Event) {
			stat, err := hh.applyCustomOperation(opReq.Namespace, opReq.CustomBody, opReq.IsDeleteOperation)
			if err != nil {
				e.Summary = fmt.Sprintf("Error while %s custom operation", stat)
				e.Details = err.Error()
				hh.StreamErr(e, err)
				return
			}
			ee.Summary = fmt.Sprintf("Manifest %s successfully", status.Deployed)
			ee.Details = ""
			hh.StreamInfo(e)
		}(mesh, e)
	case common.SmiConformanceOperation:
		go func(hh *Mesh, ee *adapter.Event) {
			name := operations[opReq.OperationName].Description
			_, err := hh.RunSMITest(adapter.SMITestOptions{
				Ctx:         context.TODO(),
				OperationID: ee.Operationid,
				Manifest:    SMIManifest,
				Namespace:   "meshery",
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
			})
			if err != nil {
				e.Summary = fmt.Sprintf("Error while %s %s test", status.Running, name)
				e.Details = err.Error()
				hh.StreamErr(e, err)
				return
			}
			ee.Summary = fmt.Sprintf("%s test %s successfully", name, status.Completed)
			ee.Details = ""
			hh.StreamInfo(e)
		}(mesh, e)
	default:
		mesh.StreamErr(e, ErrOpInvalid)
	}

	return nil
}

// ProcessOAM will handles the grpc invocation for handling OAM objects
func (mesh *Mesh) ProcessOAM(ctx context.Context, oamReq adapter.OAMRequest) (string, error) {
	var comps []v1alpha1.Component
	for _, acomp := range oamReq.OamComps {
		comp, err := oam.ParseApplicationComponent(acomp)
		if err != nil {
			mesh.Log.Error(ErrParseOAMComponent)
			continue
		}

		comps = append(comps, comp)
	}

	config, err := oam.ParseApplicationConfiguration(oamReq.OamConfig)
	if err != nil {
		mesh.Log.Error(ErrParseOAMConfig)
	}

	// If operation is delete then first HandleConfiguration and then handle the deployment
	if oamReq.DeleteOp {
		// Process configuration
		msg2, err := mesh.HandleApplicationConfiguration(config, oamReq.DeleteOp)
		if err != nil {
			return msg2, ErrProcessOAM(err)
		}

		// Process components
		msg1, err := mesh.HandleComponents(comps, oamReq.DeleteOp)
		if err != nil {
			return msg1 + "\n" + msg2, ErrProcessOAM(err)
		}

		return msg1 + "\n" + msg2, nil
	}

	// Process components
	msg1, err := mesh.HandleComponents(comps, oamReq.DeleteOp)
	if err != nil {
		return msg1, ErrProcessOAM(err)
	}

	// Process configuration
	msg2, err := mesh.HandleApplicationConfiguration(config, oamReq.DeleteOp)
	if err != nil {
		return msg1 + "\n" + msg2, ErrProcessOAM(err)
	}

	return msg1 + "\n" + msg2, nil
}
