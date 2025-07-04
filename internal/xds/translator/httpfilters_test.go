// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package translator

import (
	"testing"

	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"

	egv1a1 "github.com/wukongcloud/gateway/api/v1alpha1"
)

func Test_sortHTTPFilters(t *testing.T) {
	tests := []struct {
		name        string
		filters     []*hcmv3.HttpFilter
		filterOrder []egv1a1.FilterPosition
		want        []*hcmv3.HttpFilter
	}{
		{
			name: "sort filters",
			filters: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(wellknown.HealthCheck),
				httpFilterForTest(egv1a1.EnvoyFilterBuffer),
			},
			want: []*hcmv3.HttpFilter{
				httpFilterForTest(wellknown.HealthCheck),
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterBuffer),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
			},
		},
		{
			name: "custom filter order-singleton filter",
			filters: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
			},
			filterOrder: []egv1a1.FilterPosition{
				{
					Name:  egv1a1.EnvoyFilterFault,
					After: ptr.To(egv1a1.EnvoyFilterCORS),
				},
				{
					Name:   egv1a1.EnvoyFilterRateLimit,
					Before: ptr.To(egv1a1.EnvoyFilterJWTAuthn),
				},
			},
			want: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
			},
		},
		{
			name: "custom filter order-singleton-before-multipleton",
			filters: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
			},
			filterOrder: []egv1a1.FilterPosition{
				{
					Name:   egv1a1.EnvoyFilterRateLimit,
					Before: ptr.To(egv1a1.EnvoyFilterWasm),
				},
			},
			want: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
			},
		},
		{
			name: "custom filter order-singleton-after-multipleton",
			filters: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
			},
			filterOrder: []egv1a1.FilterPosition{
				{
					Name:  egv1a1.EnvoyFilterJWTAuthn,
					After: ptr.To(egv1a1.EnvoyFilterWasm),
				},
			},
			want: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
			},
		},
		{
			name: "custom filter order-multipleton-before-singleton",
			filters: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
			},
			filterOrder: []egv1a1.FilterPosition{
				{
					Name:   egv1a1.EnvoyFilterWasm,
					Before: ptr.To(egv1a1.EnvoyFilterJWTAuthn),
				},
			},
			want: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
			},
		},
		{
			name: "custom filter order-multipleton-after-singleton",
			filters: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
			},
			filterOrder: []egv1a1.FilterPosition{
				{
					Name:  egv1a1.EnvoyFilterWasm,
					After: ptr.To(egv1a1.EnvoyFilterRateLimit),
				},
			},
			want: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
			},
		},
		{
			name: "custom filter order-multipleton-before-multipleton",
			filters: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
			},
			filterOrder: []egv1a1.FilterPosition{
				{
					Name:   egv1a1.EnvoyFilterWasm,
					Before: ptr.To(egv1a1.EnvoyFilterExtProc),
				},
			},
			want: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
			},
		},
		{
			name: "custom filter order-multipleton-after-multipleton",
			filters: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
			},
			filterOrder: []egv1a1.FilterPosition{
				{
					Name:  egv1a1.EnvoyFilterExtProc,
					After: ptr.To(egv1a1.EnvoyFilterWasm),
				},
			},
			want: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
			},
		},
		{
			name: "custom filter order-complex-ordering",
			filters: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
			},
			filterOrder: []egv1a1.FilterPosition{
				{
					Name:   egv1a1.EnvoyFilterLocalRateLimit,
					Before: ptr.To(egv1a1.EnvoyFilterJWTAuthn),
				},
				{
					Name:  egv1a1.EnvoyFilterLocalRateLimit,
					After: ptr.To(egv1a1.EnvoyFilterCORS),
				},
				{
					Name:   egv1a1.EnvoyFilterWasm,
					Before: ptr.To(egv1a1.EnvoyFilterOAuth2),
				},
				{
					Name:   egv1a1.EnvoyFilterExtProc,
					Before: ptr.To(egv1a1.EnvoyFilterWasm),
				},
			},
			want: []*hcmv3.HttpFilter{
				httpFilterForTest(egv1a1.EnvoyFilterFault),
				httpFilterForTest(egv1a1.EnvoyFilterCORS),
				httpFilterForTest(egv1a1.EnvoyFilterLocalRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterExtAuthz + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterBasicAuth),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterExtProc + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/0"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/1"),
				httpFilterForTest(egv1a1.EnvoyFilterWasm + "/envoyextensionpolicy/default/policy-for-http-route-1/2"),
				httpFilterForTest(egv1a1.EnvoyFilterOAuth2 + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterJWTAuthn),
				httpFilterForTest(egv1a1.EnvoyFilterRBAC + "/securitypolicy/default/policy-for-http-route-1"),
				httpFilterForTest(egv1a1.EnvoyFilterRateLimit),
				httpFilterForTest(egv1a1.EnvoyFilterRouter),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortHTTPFilters(tt.filters, tt.filterOrder)
			assert.Equalf(t, tt.want, result, "sortHTTPFilters(%v)", tt.filters)
		})
	}
}

func httpFilterForTest(name egv1a1.EnvoyFilter) *hcmv3.HttpFilter {
	return &hcmv3.HttpFilter{
		Name: string(name),
	}
}
