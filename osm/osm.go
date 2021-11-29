// Copyright 2020 Layer5, Inc.
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

package osm

import (
	"context"

	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-osm/osm/oam"
	meshkitCfg "github.com/layer5io/meshkit/config"
	"github.com/layer5io/meshkit/logger"
	"github.com/layer5io/meshkit/models/oam/core/v1alpha1"
)

// Handler instance for this adapter
type Handler struct {
	adapter.Adapter
}

// New initializes a new handler instance
func New(config meshkitCfg.Handler, log logger.Handler, kc meshkitCfg.Handler) adapter.Handler {
	return &Handler{
		Adapter: adapter.Adapter{
			Config:            config,
			Log:               log,
			KubeconfigHandler: kc,
		},
	}
}

// ProcessOAM will handles the grpc invocation for handling OAM objects
func (h *Handler) ProcessOAM(ctx context.Context, oamReq adapter.OAMRequest) (string, error) {
	var comps []v1alpha1.Component
	for _, acomp := range oamReq.OamComps {
		comp, err := oam.ParseApplicationComponent(acomp)
		if err != nil {
			h.Log.Error(ErrParseOAMComponent)
			continue
		}

		comps = append(comps, comp)
	}

	config, err := oam.ParseApplicationConfiguration(oamReq.OamConfig)
	if err != nil {
		h.Log.Error(ErrParseOAMConfig)
	}

	// If operation is delete then first HandleConfiguration and then handle the deployment
	if oamReq.DeleteOp {
		// Process configuration
		msg2, err := h.HandleApplicationConfiguration(config, oamReq.DeleteOp)
		if err != nil {
			return msg2, ErrProcessOAM(err)
		}

		// Process components
		msg1, err := h.HandleComponents(comps, oamReq.DeleteOp)
		if err != nil {
			return msg1 + "\n" + msg2, ErrProcessOAM(err)
		}

		return msg1 + "\n" + msg2, nil
	}

	// Process components
	msg1, err := h.HandleComponents(comps, oamReq.DeleteOp)
	if err != nil {
		return msg1, ErrProcessOAM(err)
	}

	// Process configuration
	msg2, err := h.HandleApplicationConfiguration(config, oamReq.DeleteOp)
	if err != nil {
		return msg1 + "\n" + msg2, ErrProcessOAM(err)
	}

	return msg1 + "\n" + msg2, nil
}
