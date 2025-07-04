// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package config

import (
	"errors"
	"os"

	"k8s.io/apimachinery/pkg/runtime/serializer"

	egv1a1 "github.com/wukongcloud/gateway/api/v1alpha1"
	"github.com/wukongcloud/gateway/internal/envoygateway"
)

func Decode(cfgPath string) (*egv1a1.EnvoyGateway, error) {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	// Decode the config file.
	decoder := serializer.NewCodecFactory(envoygateway.GetScheme()).UniversalDeserializer()
	obj, gvk, err := decoder.Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}

	// Figure out the resource type from the Group|Version|Kind.
	if gvk.Group != egv1a1.GroupVersion.Group ||
		gvk.Version != egv1a1.GroupVersion.Version ||
		gvk.Kind != egv1a1.KindEnvoyGateway {
		return nil, errors.New("failed to decode unmatched resource type")
	}

	// Attempt to cast the object.
	eg, ok := obj.(*egv1a1.EnvoyGateway)
	if !ok {
		return nil, errors.New("failed to convert object to EnvoyGateway type")
	}

	return eg, nil
}
