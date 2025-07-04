// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package kubernetes

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/wukongcloud/gateway/internal/infrastructure/kubernetes/ratelimit"
)

// CreateOrUpdateRateLimitInfra creates the managed kube rate limit infra, if it doesn't exist.
func (i *Infra) CreateOrUpdateRateLimitInfra(ctx context.Context) error {
	if err := ratelimit.Validate(ctx, i.Client.Client, i.EnvoyGateway, i.ControllerNamespace); err != nil {
		return err
	}

	// Create ratelimit infra requires the uid of owner reference.
	ownerReferenceUID := make(map[string]types.UID)
	key := types.NamespacedName{
		Namespace: i.ControllerNamespace,
		Name:      "envoy-gateway",
	}

	serviceUID, err := i.Client.GetUID(ctx, key, &corev1.Service{})
	if err != nil {
		return err
	}
	ownerReferenceUID[ratelimit.ResourceKindService] = serviceUID

	deploymentUID, err := i.Client.GetUID(ctx, key, &appsv1.Deployment{})
	if err != nil {
		return err
	}
	ownerReferenceUID[ratelimit.ResourceKindDeployment] = deploymentUID

	serviceAccountUID, err := i.Client.GetUID(ctx, key, &corev1.ServiceAccount{})
	if err != nil {
		return err
	}
	ownerReferenceUID[ratelimit.ResourceKindServiceAccount] = serviceAccountUID

	r := ratelimit.NewResourceRender(i.ControllerNamespace, i.EnvoyGateway, ownerReferenceUID)
	return i.createOrUpdate(ctx, r)
}

// DeleteRateLimitInfra removes the managed kube infra, if it doesn't exist.
func (i *Infra) DeleteRateLimitInfra(ctx context.Context) error {
	// Delete ratelimit infra do not require the uid of owner reference.
	r := ratelimit.NewResourceRender(i.ControllerNamespace, i.EnvoyGateway, nil)
	return i.delete(ctx, r)
}
