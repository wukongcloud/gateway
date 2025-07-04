// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

//go:build e2e

package tests

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	"sigs.k8s.io/gateway-api/conformance/utils/http"
	"sigs.k8s.io/gateway-api/conformance/utils/kubernetes"
	"sigs.k8s.io/gateway-api/conformance/utils/roundtripper"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
	"sigs.k8s.io/gateway-api/conformance/utils/tlog"

	"github.com/wukongcloud/gateway/internal/gatewayapi"
	"github.com/wukongcloud/gateway/internal/gatewayapi/resource"
	"github.com/wukongcloud/gateway/test/utils/prometheus"
)

func init() {
	ConformanceTests = append(ConformanceTests, RateLimitCIDRMatchTest)
	ConformanceTests = append(ConformanceTests, RateLimitHeaderMatchTest)
	ConformanceTests = append(ConformanceTests, GlobalRateLimitHeaderInvertMatchTest)
	ConformanceTests = append(ConformanceTests, RateLimitHeadersDisabled)
	ConformanceTests = append(ConformanceTests, RateLimitBasedJwtClaimsTest)
	ConformanceTests = append(ConformanceTests, RateLimitMultipleListenersTest)
	ConformanceTests = append(ConformanceTests, RateLimitHeadersAndCIDRMatchTest)
	ConformanceTests = append(ConformanceTests, UsageRateLimitTest)
	ConformanceTests = append(ConformanceTests, RateLimitGlobalSharedCidrMatchTest)
	ConformanceTests = append(ConformanceTests, RateLimitGlobalSharedGatewayHeaderMatchTest)
	ConformanceTests = append(ConformanceTests, RateLimitGlobalMergeTest)
}

var RateLimitCIDRMatchTest = suite.ConformanceTest{
	ShortName:   "RateLimitCIDRMatch",
	Description: "Limit all requests that match CIDR",
	Manifests:   []string{"testdata/ratelimit-cidr-match.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		t.Run("block all ips", func(t *testing.T) {
			ns := "gateway-conformance-infra"
			routeNN := types.NamespacedName{Name: "cidr-ratelimit", Namespace: ns}
			gwNN := types.NamespacedName{Name: "same-namespace", Namespace: ns}
			gwAddr := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), routeNN)
			ratelimitHeader := make(map[string]string)
			expectOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path: "/",
				},
				Response: http.Response{
					StatusCode: 200,
					Headers:    ratelimitHeader,
				},
				Namespace: ns,
			}
			expectOkResp.Response.Headers["X-Ratelimit-Limit"] = "3, 3;w=3600"
			expectOkReq := http.MakeRequest(t, &expectOkResp, gwAddr, "HTTP", "http")

			expectLimitResp := http.ExpectedResponse{
				Request: http.Request{
					Path: "/",
				},
				Response: http.Response{
					StatusCode: 429,
				},
				Namespace: ns,
			}
			expectLimitReq := http.MakeRequest(t, &expectLimitResp, gwAddr, "HTTP", "http")

			// should just send exactly 4 requests, and expect 429

			// keep sending requests till get 200 first, that will cost one 200
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, expectOkResp)

			// fire the rest of requests
			if err := GotExactExpectedResponse(t, 2, suite.RoundTripper, expectOkReq, expectOkResp); err != nil {
				t.Errorf("failed to get expected response for the first three requests: %v", err)
			}
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectLimitReq, expectLimitResp); err != nil {
				t.Errorf("failed to get expected response for the last (fourth) request: %v", err)
			}
			// make sure that metric worked as expected.
			if err := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, time.Minute, true, func(ctx context.Context) (done bool, err error) {
				v, err := prometheus.QueryPrometheus(suite.Client, `ratelimit_service_rate_limit_over_limit{key2="masked_remote_address_0_0_0_0/0"}`)
				if err != nil {
					tlog.Logf(t, "failed to query prometheus: %v", err)
					return false, err
				}
				if v != nil {
					tlog.Logf(t, "got expected value: %v", v)
					return true, nil
				}
				return false, nil
			}); err != nil {
				t.Errorf("failed to get expected response for the last (fourth) request: %v", err)
			}
		})
	},
}

var RateLimitHeaderMatchTest = suite.ConformanceTest{
	ShortName:   "RateLimitHeaderMatch",
	Description: "Limit all requests that match headers",
	Manifests:   []string{"testdata/ratelimit-header-match.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		ns := "gateway-conformance-infra"
		routeNN := types.NamespacedName{Name: "header-ratelimit", Namespace: ns}
		gwNN := types.NamespacedName{Name: "same-namespace", Namespace: ns}
		gwAddr := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), routeNN)

		t.Run("all matched headers can got limited", func(t *testing.T) {
			requestHeaders := map[string]string{
				"x-user-id":  "one",
				"x-user-org": "acme",
			}

			ratelimitHeader := make(map[string]string)
			expectOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/get",
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 200,
					Headers:    ratelimitHeader,
				},
				Namespace: ns,
			}
			expectOkResp.Response.Headers["X-Ratelimit-Limit"] = "3, 3;w=3600"
			expectOkReq := http.MakeRequest(t, &expectOkResp, gwAddr, "HTTP", "http")

			expectLimitResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/get",
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 429,
				},
				Namespace: ns,
			}
			expectLimitReq := http.MakeRequest(t, &expectLimitResp, gwAddr, "HTTP", "http")

			// should just send exactly 4 requests, and expect 429

			// keep sending requests till get 200 first, that will cost one 200
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, expectOkResp)

			// fire the rest of the requests
			if err := GotExactExpectedResponse(t, 2, suite.RoundTripper, expectOkReq, expectOkResp); err != nil {
				t.Errorf("failed to get expected response for the first three requests: %v", err)
			}
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectLimitReq, expectLimitResp); err != nil {
				t.Errorf("failed to get expected response for the last (fourth) request: %v", err)
			}
		})

		t.Run("only one matched header cannot got limited", func(t *testing.T) {
			requestHeaders := map[string]string{
				"x-user-id": "one",
			}

			// it does not require any rate limit header, since this request never be rate limited.
			expectOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/get",
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 200,
				},
				Namespace: ns,
			}
			expectOkReq := http.MakeRequest(t, &expectOkResp, gwAddr, "HTTP", "http")

			// send exactly 4 requests, and still expect 200

			// keep sending requests till get 200 first, that will cost one 200
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, expectOkResp)

			// fire the rest of the requests
			if err := GotExactExpectedResponse(t, 3, suite.RoundTripper, expectOkReq, expectOkResp); err != nil {
				t.Errorf("failed to get expected responses for the request: %v", err)
			}
		})
	},
}

var GlobalRateLimitHeaderInvertMatchTest = suite.ConformanceTest{
	ShortName:   "GlobalRateLimitHeaderInvertMatch",
	Description: "Limit all requests that match distinct headers except for which invert is set to true",
	Manifests:   []string{"testdata/ratelimit-header-invert-match-global.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		ns := "gateway-conformance-infra"
		routeNN := types.NamespacedName{Name: "header-ratelimit", Namespace: ns}
		gwNN := types.NamespacedName{Name: "same-namespace", Namespace: ns}
		gwAddr := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), routeNN)

		t.Run("all matched headers got limited", func(t *testing.T) {
			requestHeaders := map[string]string{
				"x-user-name": "username",
			}

			ratelimitHeader := make(map[string]string)
			expectOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/get",
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 200,
					Headers:    ratelimitHeader,
				},
				Namespace: ns,
			}
			expectOkResp.Response.Headers["X-Ratelimit-Limit"] = "3, 3;w=3600"
			expectOkReq := http.MakeRequest(t, &expectOkResp, gwAddr, "HTTP", "http")

			expectLimitResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/get",
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 429,
				},
				Namespace: ns,
			}
			expectLimitReq := http.MakeRequest(t, &expectLimitResp, gwAddr, "HTTP", "http")

			// should just send exactly 4 requests, and expect 429

			// keep sending requests till get 200 first, that will cost one 200
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, expectOkResp)

			// fire the rest of the requests
			if err := GotExactExpectedResponse(t, 2, suite.RoundTripper, expectOkReq, expectOkResp); err != nil {
				t.Errorf("failed to get expected response for the first three requests: %v", err)
			}
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectLimitReq, expectLimitResp); err != nil {
				t.Errorf("failed to get expected response for the last (fourth) request: %v", err)
			}
		})

		t.Run("if header matched with invert will not get limited", func(t *testing.T) {
			requestHeaders := map[string]string{
				"x-user-name": "admin",
			}

			// it does not require any rate limit header, since this request never be rate limited.
			expectOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/get",
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 200,
				},
				Namespace: ns,
			}
			expectOkReq := http.MakeRequest(t, &expectOkResp, gwAddr, "HTTP", "http")

			// send exactly 4 requests, and still expect 200

			// keep sending requests till get 200 first, that will cost one 200
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, expectOkResp)

			// fire the rest of the requests
			if err := GotExactExpectedResponse(t, 3, suite.RoundTripper, expectOkReq, expectOkResp); err != nil {
				t.Errorf("failed to get expected responses for the request: %v", err)
			}
		})
	},
}

var RateLimitHeadersDisabled = suite.ConformanceTest{
	ShortName:   "RateLimitHeadersDisabled",
	Description: "Disable rate limit headers",
	Manifests:   []string{"testdata/ratelimit-headers-disabled.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		ns := "gateway-conformance-infra"
		routeNN := types.NamespacedName{Name: "ratelimit-headers-disabled", Namespace: ns}
		gwNN := types.NamespacedName{Name: "same-namespace", Namespace: ns}
		gwAddr := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), routeNN)

		t.Run("all matched headers can get limited", func(t *testing.T) {
			requestHeaders := map[string]string{
				"x-user-id":  "one",
				"x-user-org": "acme",
			}

			ratelimitHeader := make(map[string]string)
			expectOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/get",
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 200,
					Headers:    ratelimitHeader,
				},
				Namespace: ns,
			}
			// expectOkResp.Response.Headers["X-Ratelimit-Limit"] is not defined because we disabled it.
			expectOkReq := http.MakeRequest(t, &expectOkResp, gwAddr, "HTTP", "http")

			expectLimitResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/get",
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 429,
				},
				Namespace: ns,
			}
			expectLimitReq := http.MakeRequest(t, &expectLimitResp, gwAddr, "HTTP", "http")

			// should just send exactly 4 requests, and expect 429

			// keep sending requests till get 200 first, that will cost one 200
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, expectOkResp)

			// fire the rest of the requests
			if err := GotExactExpectedResponse(t, 2, suite.RoundTripper, expectOkReq, expectOkResp); err != nil {
				t.Errorf("failed to get expected response for the first three requests: %v", err)
			}
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectLimitReq, expectLimitResp); err != nil {
				t.Errorf("failed to get expected response for the last (fourth) request: %v", err)
			}
		})

		t.Run("only one matched header cannot got limited", func(t *testing.T) {
			requestHeaders := map[string]string{
				"x-user-id": "one",
			}

			// it does not require any rate limit header, since this request never be rate limited.
			expectOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/get",
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 200,
				},
				Namespace: ns,
			}
			expectOkReq := http.MakeRequest(t, &expectOkResp, gwAddr, "HTTP", "http")

			// send exactly 4 requests, and still expect 200

			// keep sending requests till get 200 first, that will cost one 200
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, expectOkResp)

			// fire the rest of the requests
			if err := GotExactExpectedResponse(t, 3, suite.RoundTripper, expectOkReq, expectOkResp); err != nil {
				t.Errorf("failed to get expected responses for the request: %v", err)
			}
		})
	},
}

var RateLimitBasedJwtClaimsTest = suite.ConformanceTest{
	ShortName:   "RateLimitBasedJwtClaims",
	Description: "Limit based jwt claims",
	Manifests:   []string{"testdata/ratelimit-based-jwt-claims.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		t.Run("ratelimit based on jwt claims", func(t *testing.T) {
			ns := "gateway-conformance-infra"
			routeNN := types.NamespacedName{Name: "http-ratelimit-based-jwt-claims", Namespace: ns}
			gwNN := types.NamespacedName{Name: "same-namespace", Namespace: ns}
			gwAddr := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), routeNN)

			preCount, err := OverLimitCount(suite)
			require.NoError(t, err)

			expectOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path: "/foo",
				},
				Response: http.Response{
					StatusCode: 200,
				},
				Namespace: ns,
			}

			expectLimitResp := http.ExpectedResponse{
				Request: http.Request{
					Path: "/foo",
				},
				Response: http.Response{
					StatusCode: 429,
				},
				Namespace: ns,
			}

			// Just to construct the request that carries a jwt token that can be limited
			ratelimitHeader := make(map[string]string)
			TokenHeader := make(map[string]string)
			JwtOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/foo",
					Headers: TokenHeader,
				},
				ExpectedRequest: &http.ExpectedRequest{
					Request: http.Request{
						Path: "/foo",
					},
				},
				Response: http.Response{
					StatusCode: 200,
					Headers:    ratelimitHeader,
				},
				Namespace: ns,
			}
			JwtOkResp.Request.Headers["Authorization"] = "Bearer " + "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.NHVaYe26MbtOYhSKkoKYdFVomg4i8ZJd8_-RU8VNbftc4TSMb4bXP3l3YlNWACwyXPGffz5aXHc6lty1Y2t4SWRqGteragsVdZufDn5BlnJl9pdR_kdVFUsra2rWKEofkZeIC4yWytE58sMIihvo9H1ScmmVwBcQP6XETqYd0aSHp1gOa9RdUPDvoXQ5oqygTqVtxaDr6wUFKrKItgBMzWIdNZ6y7O9E0DhEPTbE9rfBo6KTFsHAZnMg4k68CDp2woYIaXbmYTWcvbzIuHO7_37GT79XdIwkm95QJ7hYC9RiwrV7mesbY4PAahERJawntho0my942XheVLmGwLMBkQ"
			JwtOkResp.Response.Headers["X-Ratelimit-Limit"] = "3, 3;w=3600"

			JwtReq := http.MakeRequest(t, &JwtOkResp, gwAddr, "HTTP", "http")

			// Just to construct the request that carries a jwt token that can not be limited
			DifTokenHeader := make(map[string]string)
			difJwtOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/foo",
					Headers: DifTokenHeader,
				},
				Response: http.Response{
					StatusCode: 200,
				},
				Namespace: ns,
			}
			difJwtOkResp.Request.Headers["Authorization"] = "Bearer " + "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlRvbSIsImFkbWluIjp0cnVlLCJpYXQiOjE1MTYyMzkwMjJ9.kyzDDSo7XpweSPU1lxoI9IHzhTBrRNlnmcW9lmCbloZELShg-8isBx4AFoM4unXZTHpS_Y24y0gmd4nDQxgUE-CgjVSnGCb0Xhy3WO1gm9iChoKDyyQ3kHp98EmKxTyxKG2X9GyKcDFNBDjH12OBD7TcJUaBEvLf6Jw1SG2A7FakUPWeK04DQ916-ROylzI6qKyaZ0OpfYIbijvyAQxlQRxxs2XHlAkLdJhfVcUqJBwsFTbwHYARC-WNgd2_etAk1GWdwwZ_NoTmRzZAMryrYJpHY9KPlbnZ93Ye3o9h2viBQ_XRb7JBkWnAGYO4_KswpJWE_7ROUVj8iOJo2jfY6w"

			difJwtReq := http.MakeRequest(t, &difJwtOkResp, gwAddr, "HTTP", "http")

			// make sure the gateway is available
			OkResp := http.ExpectedResponse{
				Request: http.Request{
					Path: "/bar",
				},
				Response: http.Response{
					StatusCode: 200,
				},
				Namespace: ns,
			}

			// keep sending requests till get 200 first to make sure the gateway is available
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, OkResp)

			// should just send exactly 4 requests, and expect 429

			// keep sending requests till get 200 first, that will cost one 200
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, JwtOkResp)

			// fire the rest of requests
			if err := GotExactExpectedResponse(t, 2, suite.RoundTripper, JwtReq, JwtOkResp); err != nil {
				t.Errorf("failed to get expected response at third request: %v", err)
			}
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, JwtReq, expectLimitResp); err != nil {
				t.Errorf("failed to get expected response at the fourth request: %v", err)
			}

			// Carrying different jwt claims will not be limited
			if err := GotExactExpectedResponse(t, 4, suite.RoundTripper, difJwtReq, expectOkResp); err != nil {
				t.Errorf("failed to get expected response for the request with a different jwt: %v", err)
			}

			// make sure the request with no token is rejected
			noTokenResp := http.ExpectedResponse{
				Request: http.Request{
					Path: "/foo",
				},
				Response: http.Response{
					StatusCode: 401,
				},
				Namespace: ns,
			}
			noTokenReq := http.MakeRequest(t, &noTokenResp, gwAddr, "HTTP", "http")
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, noTokenReq, noTokenResp); err != nil {
				t.Errorf("failed to get expected response: %v", err)
			}

			err = wait.PollUntilContextTimeout(context.TODO(), time.Second, 1*time.Minute, true, func(_ context.Context) (bool, error) {
				curCount, err := OverLimitCount(suite)
				if err != nil {
					return false, err
				}
				return curCount > preCount, nil
			})
			require.NoError(t, err)
		})
	},
}

var RateLimitMultipleListenersTest = suite.ConformanceTest{
	ShortName:   "RateLimitMultipleListeners",
	Description: "Limit requests on multiple listeners",
	Manifests:   []string{"testdata/ratelimit-multiple-listeners.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		t.Run("block all ips on listener 80 and 8080", func(t *testing.T) {
			ns := "gateway-conformance-infra"
			routeNN := types.NamespacedName{Name: "cidr-ratelimit", Namespace: ns}
			gwNN := types.NamespacedName{Name: "eg-rate-limit", Namespace: ns}
			gwAddr := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), routeNN)
			gwIP, _, err := net.SplitHostPort(gwAddr)
			require.NoError(t, err)

			gwPorts := []string{"80", "8080"}
			for _, port := range gwPorts {
				gwAddr = net.JoinHostPort(gwIP, port)

				ratelimitHeader := make(map[string]string)
				expectOkResp := http.ExpectedResponse{
					Request: http.Request{
						Path: "/",
					},
					Response: http.Response{
						StatusCode: 200,
						Headers:    ratelimitHeader,
					},
					Namespace: ns,
				}
				expectOkResp.Response.Headers["X-Ratelimit-Limit"] = "3, 3;w=3600"
				expectOkReq := http.MakeRequest(t, &expectOkResp, gwAddr, "HTTP", "http")

				expectLimitResp := http.ExpectedResponse{
					Request: http.Request{
						Path: "/",
					},
					Response: http.Response{
						StatusCode: 429,
					},
					Namespace: ns,
				}
				expectLimitReq := http.MakeRequest(t, &expectLimitResp, gwAddr, "HTTP", "http")

				// should just send exactly 4 requests, and expect 429

				// keep sending requests till get 200 first, that will cost one 200
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, expectOkResp)

				// fire the rest of requests
				if err := GotExactExpectedResponse(t, 2, suite.RoundTripper, expectOkReq, expectOkResp); err != nil {
					t.Errorf("failed to get expected response for the first three requests: %v", err)
				}
				if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectLimitReq, expectLimitResp); err != nil {
					t.Errorf("failed to get expected response for the last (fourth) request: %v", err)
				}
			}
		})
	},
}

var RateLimitHeadersAndCIDRMatchTest = suite.ConformanceTest{
	ShortName:   "RateLimitHeadersAndCIDRMatch",
	Description: "Limit requests on rule that has both headers and cidr matches",
	Manifests:   []string{"testdata/ratelimit-headers-and-cidr-match.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		ns := "gateway-conformance-infra"
		routeNN := types.NamespacedName{Name: "header-and-cidr-ratelimit", Namespace: ns}
		gwNN := types.NamespacedName{Name: "same-namespace", Namespace: ns}
		gwAddr := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), routeNN)

		t.Run("all matched both headers and cidr can got limited", func(t *testing.T) {
			if IPFamily == "ipv6" {
				t.Skip("Skipping test as IP_FAMILY is IPv6")
			}

			requestHeaders := map[string]string{
				"x-user-id":  "one",
				"x-user-org": "acme",
			}

			ratelimitHeader := make(map[string]string)
			expectOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/get",
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 200,
					Headers:    ratelimitHeader,
				},
				Namespace: ns,
			}
			expectOkResp.Response.Headers["X-Ratelimit-Limit"] = "3, 3;w=3600"
			expectOkReq := http.MakeRequest(t, &expectOkResp, gwAddr, "HTTP", "http")

			expectLimitResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/get",
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 429,
				},
				Namespace: ns,
			}
			expectLimitReq := http.MakeRequest(t, &expectLimitResp, gwAddr, "HTTP", "http")

			// should just send exactly 4 requests, and expect 429

			// keep sending requests till get 200 first, that will cost one 200
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, expectOkResp)

			// fire the rest of the requests
			if err := GotExactExpectedResponse(t, 2, suite.RoundTripper, expectOkReq, expectOkResp); err != nil {
				t.Errorf("failed to get expected response for the first three requests: %v", err)
			}
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectLimitReq, expectLimitResp); err != nil {
				t.Errorf("failed to get expected response for the last (fourth) request: %v", err)
			}
		})

		t.Run("only partly matched headers cannot got limited", func(t *testing.T) {
			requestHeaders := map[string]string{
				"x-user-id": "one",
			}

			// it does not require any rate limit header, since this request never be rate limited.
			expectOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/get",
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 200,
				},
				Namespace: ns,
			}
			expectOkReq := http.MakeRequest(t, &expectOkResp, gwAddr, "HTTP", "http")

			// send exactly 4 requests, and still expect 200

			// keep sending requests till get 200 first, that will cost one 200
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, expectOkResp)

			// fire the rest of the requests
			if err := GotExactExpectedResponse(t, 3, suite.RoundTripper, expectOkReq, expectOkResp); err != nil {
				t.Errorf("failed to get expected responses for the request: %v", err)
			}
		})

		t.Run("only matched cidr cannot got limited", func(t *testing.T) {
			// it does not require any rate limit header, since this request never be rate limited.
			expectOkResp := http.ExpectedResponse{
				Request: http.Request{
					Path: "/get",
				},
				Response: http.Response{
					StatusCode: 200,
				},
				Namespace: ns,
			}
			expectOkReq := http.MakeRequest(t, &expectOkResp, gwAddr, "HTTP", "http")

			// send exactly 4 requests, and still expect 200

			// keep sending requests till get 200 first, that will cost one 200
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, expectOkResp)

			// fire the rest of the requests
			if err := GotExactExpectedResponse(t, 3, suite.RoundTripper, expectOkReq, expectOkResp); err != nil {
				t.Errorf("failed to get expected responses for the request: %v", err)
			}
		})
	},
}

var UsageRateLimitTest = suite.ConformanceTest{
	ShortName:   "UsageRateLimit",
	Description: "Perform usage-based rate limit based on response content",
	Manifests:   []string{"testdata/ext-proc-service.yaml", "testdata/ratelimit-usage-ratelimit.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		ns := "gateway-conformance-infra"
		routeNN := types.NamespacedName{Name: "usage-rate-limit", Namespace: ns}
		gwNN := types.NamespacedName{Name: "same-namespace", Namespace: ns}
		gwAddr := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), routeNN)

		// Waiting for the extproc service to be ready.
		ancestorRef := gwapiv1a2.ParentReference{
			Group:     gatewayapi.GroupPtr(gwapiv1.GroupName),
			Kind:      gatewayapi.KindPtr(resource.KindGateway),
			Namespace: gatewayapi.NamespacePtr(gwNN.Namespace),
			Name:      gwapiv1.ObjectName(gwNN.Name),
		}
		EnvoyExtensionPolicyMustBeAccepted(t, suite.Client, types.NamespacedName{Name: "usage-rate-limit", Namespace: ns}, suite.ControllerName, ancestorRef)
		podReady := corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}
		// Wait for the grpc ext auth service pod to be ready
		WaitForPods(t, suite.Client, ns, map[string]string{"app": "grpc-ext-proc"}, corev1.PodRunning, podReady)

		requestHeaders := map[string]string{"x-user-id": "one"}

		ratelimitHeader := make(map[string]string)
		expectOkResp := http.ExpectedResponse{
			Request: http.Request{
				Path:    "/get",
				Headers: requestHeaders,
			},
			Response:  http.Response{StatusCode: 200, Headers: ratelimitHeader},
			Namespace: ns,
		}
		expectOkResp.Response.Headers["X-Ratelimit-Limit"] = "21, 21;w=3600"
		expectOkReq := http.MakeRequest(t, &expectOkResp, gwAddr, "HTTP", "http")

		expectLimitResp := http.ExpectedResponse{
			Request: http.Request{
				Path:    "/get",
				Headers: requestHeaders,
			},
			Response: http.Response{
				StatusCode: 429,
			},
			Namespace: ns,
		}
		expectLimitReq := http.MakeRequest(t, &expectLimitResp, gwAddr, "HTTP", "http")

		// Keep sending requests till get 200 first, that will cost 10 usage.
		http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr, expectOkResp)

		// The next two request will be fine as the limit is set to 21.
		if err := GotExactExpectedResponse(t, 2, suite.RoundTripper, expectOkReq, expectOkResp); err != nil {
			t.Errorf("failed to get expected response for the first three requests: %v", err)
		}
		// At this point, the budget must be zero (21 -> 11 -> 1 -> 0), so the next request will be limited.
		if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectLimitReq, expectLimitResp); err != nil {
			t.Errorf("failed to get expected response for the last (fourth) request: %v", err)
		}
	},
}

var RateLimitGlobalSharedCidrMatchTest = suite.ConformanceTest{
	ShortName:   "RateLimitGlobalSharedCidrMatch",
	Description: "Limit all requests that match CIDR across multiple routes with a shared rate limit",
	Manifests:   []string{"testdata/ratelimit-global-shared-cidr-match.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		t.Run("block all ips with shared rate limit across routes with different paths", func(t *testing.T) {
			ns := "gateway-conformance-infra"
			route1NN := types.NamespacedName{Name: "cidr-ratelimit-1", Namespace: ns}
			route2NN := types.NamespacedName{Name: "cidr-ratelimit-2", Namespace: ns}
			gwNN := types.NamespacedName{Name: "same-namespace", Namespace: ns}

			// Get gateway address for the first route
			gwAddr1 := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), route1NN)

			// Get gateway address for the second route
			gwAddr2 := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), route2NN)

			ratelimitHeader := make(map[string]string)
			expectOkResp1 := http.ExpectedResponse{
				Request: http.Request{
					Path: "/foo", // First route path
				},
				Response: http.Response{
					StatusCode: 200,
					Headers:    ratelimitHeader,
				},
				Namespace: ns,
			}
			expectOkResp1.Response.Headers["X-Ratelimit-Limit"] = "3, 3;w=3600"

			expectOkResp2 := http.ExpectedResponse{
				Request: http.Request{
					Path: "/bar", // Second route path
				},
				Response: http.Response{
					StatusCode: 200,
					Headers:    ratelimitHeader,
				},
				Namespace: ns,
			}
			expectOkResp2.Response.Headers["X-Ratelimit-Limit"] = "3, 3;w=3600"

			expectLimitResp := http.ExpectedResponse{
				Request: http.Request{
					Path: "/bar", // Path for testing the limit on the second route
				},
				Response: http.Response{
					StatusCode: 429,
				},
				Namespace: ns,
			}

			// Create requests for the first route (path: /foo)
			expectOkReq1 := http.MakeRequest(t, &expectOkResp1, gwAddr1, "HTTP", "http")

			// Create requests for the second route (path: /bar)
			expectOkReq2 := http.MakeRequest(t, &expectOkResp2, gwAddr2, "HTTP", "http")
			expectLimitReq2 := http.MakeRequest(t, &expectLimitResp, gwAddr2, "HTTP", "http")

			// Ensure the first route is available
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr1, expectOkResp1)

			// Send 1 more request to the first route with /foo path (total: 2 requests)
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectOkReq1, expectOkResp1); err != nil {
				t.Errorf("failed to get expected response for the request to first route (/foo): %v", err)
			}

			// Send a request to the second route with /bar path (total: 3 requests)
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectOkReq2, expectOkResp2); err != nil {
				t.Errorf("failed to get expected response for the request to second route (/bar): %v", err)
			}

			// At this point, 3 requests have been sent in total (2 to /foo, 1 to /bar)
			// Since the rate limit is shared and set to 3, the next request should be rate limited
			// even though it's going to a different path
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectLimitReq2, expectLimitResp); err != nil {
				t.Errorf("failed to get expected rate limit response for the second request to /bar: %v", err)
			}

			// Make sure that metric worked as expected.
			if err := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, time.Minute, true, func(ctx context.Context) (done bool, err error) {
				v, err := prometheus.QueryPrometheus(suite.Client, `ratelimit_service_rate_limit_over_limit{key2="masked_remote_address_0_0_0_0/0"}`)
				if err != nil {
					tlog.Logf(t, "failed to query prometheus: %v", err)
					return false, err
				}
				if v != nil {
					tlog.Logf(t, "got expected value: %v", v)
					return true, nil
				}
				return false, nil
			}); err != nil {
				t.Errorf("failed to get expected metric for rate limit: %v", err)
			}
		})
	},
}

var RateLimitGlobalSharedGatewayHeaderMatchTest = suite.ConformanceTest{
	ShortName:   "RateLimitGlobalSharedGatewayHeaderMatch",
	Description: "Limit all requests with matching headers across multiple routes with a shared rate limit",
	Manifests:   []string{"testdata/ratelimit-global-shared-gateway-header-match.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		t.Run("rate limit requests with shared header limit across routes with different paths", func(t *testing.T) {
			ns := "gateway-conformance-infra"
			route1NN := types.NamespacedName{Name: "header-ratelimit-1", Namespace: ns}
			route2NN := types.NamespacedName{Name: "header-ratelimit-2", Namespace: ns}
			gwNN := types.NamespacedName{Name: "eg-rate-limit", Namespace: ns}

			// Get gateway address for the first route
			gwAddr1 := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), route1NN)

			// Get gateway address for the second route
			gwAddr2 := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), route2NN)

			// Define headers that will trigger the rate limit
			requestHeaders := map[string]string{
				"x-user-id": "one",
			}

			ratelimitHeader := make(map[string]string)
			expectOkResp1 := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/foo", // First route path
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 200,
					Headers:    ratelimitHeader,
				},
				Namespace: ns,
			}
			expectOkResp1.Response.Headers["X-Ratelimit-Limit"] = "3, 3;w=3600"

			expectOkResp2 := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/bar", // Second route path
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 200,
					Headers:    ratelimitHeader,
				},
				Namespace: ns,
			}
			expectOkResp2.Response.Headers["X-Ratelimit-Limit"] = "3, 3;w=3600"

			expectLimitResp := http.ExpectedResponse{
				Request: http.Request{
					Path:    "/bar", // Path for testing the limit on the second route
					Headers: requestHeaders,
				},
				Response: http.Response{
					StatusCode: 429,
				},
				Namespace: ns,
			}

			// Create requests for the first route (path: /foo)
			expectOkReq1 := http.MakeRequest(t, &expectOkResp1, gwAddr1, "HTTP", "http")

			// Create requests for the second route (path: /bar)
			expectOkReq2 := http.MakeRequest(t, &expectOkResp2, gwAddr2, "HTTP", "http")
			expectLimitReq2 := http.MakeRequest(t, &expectLimitResp, gwAddr2, "HTTP", "http")

			// Ensure the first route is available
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr1, expectOkResp1)

			// Send 1 more request to the first route with /foo path (total: 2 requests)
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectOkReq1, expectOkResp1); err != nil {
				t.Errorf("failed to get expected response for the request to first route (/foo): %v", err)
			}

			// Send a request to the second route with /bar path (total: 3 requests)
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectOkReq2, expectOkResp2); err != nil {
				t.Errorf("failed to get expected response for the request to second route (/bar): %v", err)
			}

			// At this point, 3 requests have been sent in total (2 to /foo, 1 to /bar)
			// Since the rate limit is shared and set to 3, the next request should be rate limited
			// even though it's going to a different path
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, expectLimitReq2, expectLimitResp); err != nil {
				t.Errorf("failed to get expected rate limit response for the second request to /bar: %v", err)
			}

			// Make sure that metric worked as expected.
			if err := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, time.Minute, true, func(ctx context.Context) (done bool, err error) {
				v, err := prometheus.QueryPrometheus(suite.Client, `ratelimit_service_rate_limit_over_limit{key2="header_x-user-id_one"}`)
				if err != nil {
					tlog.Logf(t, "failed to query prometheus: %v", err)
					return false, err
				}
				if v != nil {
					tlog.Logf(t, "got expected value: %v", v)
					return true, nil
				}
				return false, nil
			}); err != nil {
				t.Errorf("failed to get expected metric for rate limit: %v", err)
			}
		})
	},
}

var RateLimitGlobalMergeTest = suite.ConformanceTest{
	ShortName:   "RateLimitGlobalMergeTest",
	Description: "Limit requests with matching headers across multiple routes, verifying both shared and unshared rate limit behaviors",
	Manifests:   []string{"testdata/ratelimit-global-shared-and-unshared-header-match.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		ns := "gateway-conformance-infra"
		route1NN := types.NamespacedName{Name: "header-ratelimit-1", Namespace: ns}
		route2NN := types.NamespacedName{Name: "header-ratelimit-2", Namespace: ns}
		gwNN := types.NamespacedName{Name: "eg-rate-limit", Namespace: ns}

		gwAddr1 := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), route1NN)
		gwAddr2 := kubernetes.GatewayAndHTTPRoutesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, suite.ControllerName, kubernetes.NewGatewayRef(gwNN), route2NN)

		t.Run("shared_route_policy_x-user-id=one", func(t *testing.T) {
			headers := map[string]string{"x-user-id": "one"}

			expectOk1 := http.ExpectedResponse{Request: http.Request{Path: "/bar", Headers: headers}, Response: http.Response{StatusCode: 200}, Namespace: ns}
			expectOk2 := http.ExpectedResponse{Request: http.Request{Path: "/foo", Headers: headers}, Response: http.Response{StatusCode: 200}, Namespace: ns}
			expectLimit := http.ExpectedResponse{Request: http.Request{Path: "/foo", Headers: headers}, Response: http.Response{StatusCode: 429}, Namespace: ns}

			require.Eventually(t, func() bool {
				_, cRes, err := suite.RoundTripper.CaptureRoundTrip(http.MakeRequest(t, &expectOk1, gwAddr2, "HTTP", "http"))
				if err != nil {
					return false
				}
				vals := cRes.Headers["X-Ratelimit-Limit"]
				return len(vals) > 0 && vals[0] == "3, 3;w=3600"
			}, suite.TimeoutConfig.MaxTimeToConsistency, suite.TimeoutConfig.RequestTimeout, "rate limit header not yet present")

			for _, expect := range []http.ExpectedResponse{expectOk1, expectOk2} {
				if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &expect, gwAddr2, "HTTP", "http"), expect); err != nil {
					t.Errorf("expected 200 response: %v", err)
				}
			}
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &expectLimit, gwAddr1, "HTTP", "http"), expectLimit); err != nil {
				t.Errorf("expected 429 response: %v", err)
			}
		})

		t.Run("unshared_route_policy_x-user-id=two", func(t *testing.T) {
			headers := map[string]string{"x-user-id": "two"}

			okFoo := http.ExpectedResponse{Request: http.Request{Path: "/foo", Headers: headers}, Response: http.Response{StatusCode: 200}, Namespace: ns}
			limitFoo := http.ExpectedResponse{Request: http.Request{Path: "/foo", Headers: headers}, Response: http.Response{StatusCode: 429}, Namespace: ns}
			okBar := http.ExpectedResponse{Request: http.Request{Path: "/bar", Headers: headers}, Response: http.Response{StatusCode: 200}, Namespace: ns}
			limitBar := http.ExpectedResponse{Request: http.Request{Path: "/bar", Headers: headers}, Response: http.Response{StatusCode: 429}, Namespace: ns}

			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr1, okFoo)

			for i := 0; i < 2; i++ {
				if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &okFoo, gwAddr1, "HTTP", "http"), okFoo); err != nil {
					t.Errorf("foo request #%d failed: %v", i+1, err)
				}
			}
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &limitFoo, gwAddr1, "HTTP", "http"), limitFoo); err != nil {
				t.Errorf("expected 429 on 4th foo: %v", err)
			}

			for i := 0; i < 3; i++ {
				if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &okBar, gwAddr2, "HTTP", "http"), okBar); err != nil {
					t.Errorf("bar request #%d failed: %v", i+1, err)
				}
			}
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &limitBar, gwAddr2, "HTTP", "http"), limitBar); err != nil {
				t.Errorf("expected 429 on 4th bar: %v", err)
			}
		})

		t.Run("shared_gateway_policy_x-user-id=three", func(t *testing.T) {
			headers := map[string]string{"x-user-id": "three"}

			ok1 := http.ExpectedResponse{Request: http.Request{Path: "/bar", Headers: headers}, Response: http.Response{StatusCode: 200}, Namespace: ns}
			ok2 := http.ExpectedResponse{Request: http.Request{Path: "/foo", Headers: headers}, Response: http.Response{StatusCode: 200}, Namespace: ns}
			limit := http.ExpectedResponse{Request: http.Request{Path: "/foo", Headers: headers}, Response: http.Response{StatusCode: 429}, Namespace: ns}

			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr2, ok1)

			for _, expect := range []http.ExpectedResponse{ok1, ok2} {
				if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &expect, gwAddr2, "HTTP", "http"), expect); err != nil {
					t.Errorf("expected 200 response: %v", err)
				}
			}
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &limit, gwAddr1, "HTTP", "http"), limit); err != nil {
				t.Errorf("expected 429 response: %v", err)
			}
		})

		t.Run("unshared_gateway_policy__x-user-id=four", func(t *testing.T) {
			headers := map[string]string{"x-user-id": "four"}

			okFoo := http.ExpectedResponse{Request: http.Request{Path: "/foo", Headers: headers}, Response: http.Response{StatusCode: 200}, Namespace: ns}
			limitFoo := http.ExpectedResponse{Request: http.Request{Path: "/foo", Headers: headers}, Response: http.Response{StatusCode: 429}, Namespace: ns}
			okBar := http.ExpectedResponse{Request: http.Request{Path: "/bar", Headers: headers}, Response: http.Response{StatusCode: 200}, Namespace: ns}
			limitBar := http.ExpectedResponse{Request: http.Request{Path: "/bar", Headers: headers}, Response: http.Response{StatusCode: 429}, Namespace: ns}

			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr1, okFoo)

			for i := 0; i < 2; i++ {
				if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &okFoo, gwAddr1, "HTTP", "http"), okFoo); err != nil {
					t.Errorf("foo request #%d failed: %v", i+1, err)
				}
			}
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &limitFoo, gwAddr1, "HTTP", "http"), limitFoo); err != nil {
				t.Errorf("expected 429 on 4th foo: %v", err)
			}

			for i := 0; i < 3; i++ {
				if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &okBar, gwAddr2, "HTTP", "http"), okBar); err != nil {
					t.Errorf("bar request #%d failed: %v", i+1, err)
				}
			}
			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &limitBar, gwAddr2, "HTTP", "http"), limitBar); err != nil {
				t.Errorf("expected 429 on 4th bar: %v", err)
			}
		})

		t.Run("shared_no_client_selectors", func(t *testing.T) {
			ok1 := http.ExpectedResponse{Request: http.Request{Path: "/bar"}, Response: http.Response{StatusCode: 200}, Namespace: ns}
			limit := http.ExpectedResponse{Request: http.Request{Path: "/bar"}, Response: http.Response{StatusCode: 429}, Namespace: ns}

			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, gwAddr1, ok1)

			if err := GotExactExpectedResponse(t, 1, suite.RoundTripper, http.MakeRequest(t, &limit, gwAddr1, "HTTP", "http"), limit); err != nil {
				t.Errorf("expected 429 for third request: %v", err)
			}
		})
	},
}

func GotExactExpectedResponse(t *testing.T, n int, r roundtripper.RoundTripper, req roundtripper.Request, resp http.ExpectedResponse) error {
	for i := 0; i < n; i++ {
		cReq, cRes, err := r.CaptureRoundTrip(req)
		if err != nil {
			return err
		}

		if err = http.CompareRequest(t, &req, cReq, cRes, resp); err != nil {
			return err
		}
	}
	return nil
}
