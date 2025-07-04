// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package translator

import (
	"errors"
	"fmt"

	configv3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	rlv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	localrlv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	egv1a1 "github.com/wukongcloud/gateway/api/v1alpha1"
	"github.com/wukongcloud/gateway/internal/ir"
	"github.com/wukongcloud/gateway/internal/utils/ratelimit"
	"github.com/wukongcloud/gateway/internal/xds/types"
)

const (
	localRateLimitFilterStatPrefix = "http_local_rate_limiter"
	descriptorMaskedRemoteAddress  = "masked_remote_address"
	descriptorRemoteAddress        = "remote_address"
)

func init() {
	registerHTTPFilter(&localRateLimit{})
}

type localRateLimit struct{}

var _ httpFilter = &localRateLimit{}

// patchHCM builds and appends the local rate limit filter to the HTTP Connection Manager
// if applicable, and it does not already exist.
func (*localRateLimit) patchHCM(mgr *hcmv3.HttpConnectionManager, irListener *ir.HTTPListener) error {
	if mgr == nil {
		return errors.New("hcm is nil")
	}
	if irListener == nil {
		return errors.New("ir listener is nil")
	}
	if !listenerContainsLocalRateLimit(irListener) {
		return nil
	}

	// Return early if filter already exists.
	for _, httpFilter := range mgr.HttpFilters {
		if httpFilter.Name == egv1a1.EnvoyFilterLocalRateLimit.String() {
			return nil
		}
	}

	localRl := &localrlv3.LocalRateLimit{
		StatPrefix: localRateLimitFilterStatPrefix,
		MaxDynamicDescriptors: &wrapperspb.UInt32Value{
			Value: 10000,
			// Default to 10k, assuming a listener has 10k unique active users to be rate limited.
			// We can make this configurable in the API if needed.
		},
	}

	localRlAny, err := anypb.New(localRl)
	if err != nil {
		return err
	}

	// The local rate limit filter at the HTTP connection manager level is an
	// empty filter. The real configuration is done at the route level.
	// See https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/local_rate_limit_filter
	filter := &hcmv3.HttpFilter{
		Name: egv1a1.EnvoyFilterLocalRateLimit.String(),
		ConfigType: &hcmv3.HttpFilter_TypedConfig{
			TypedConfig: localRlAny,
		},
	}

	mgr.HttpFilters = append(mgr.HttpFilters, filter)
	return nil
}

func listenerContainsLocalRateLimit(irListener *ir.HTTPListener) bool {
	if irListener == nil {
		return false
	}

	for _, route := range irListener.Routes {
		if routeContainsLocalRateLimit(route) {
			return true
		}
	}

	return false
}

func routeContainsLocalRateLimit(irRoute *ir.HTTPRoute) bool {
	if irRoute == nil ||
		irRoute.Traffic == nil ||
		irRoute.Traffic.RateLimit == nil ||
		irRoute.Traffic.RateLimit.Local == nil {
		return false
	}

	return true
}

func (*localRateLimit) patchResources(*types.ResourceVersionTable,
	[]*ir.HTTPRoute,
) error {
	return nil
}

func (*localRateLimit) patchRoute(route *routev3.Route, irRoute *ir.HTTPRoute) error {
	routeAction := route.GetRoute()

	// Return early if no rate limit config exists.
	if !routeContainsLocalRateLimit(irRoute) || routeAction == nil {
		return nil
	}

	if routeAction.RateLimits != nil {
		// This should not happen since this is the only place where the rate limit
		// config is added in a route.
		return fmt.Errorf(
			"route already contains rate limit config:  %s",
			route.Name)
	}

	local := irRoute.Traffic.RateLimit.Local

	rateLimits, descriptors := buildRouteLocalRateLimits(local)
	routeAction.RateLimits = rateLimits

	filterCfg := route.GetTypedPerFilterConfig()
	if _, ok := filterCfg[egv1a1.EnvoyFilterLocalRateLimit.String()]; ok {
		// This should not happen since this is the only place where the filter
		// config is added in a route.
		return fmt.Errorf(
			"route already contains local rate limit filter config:  %s",
			route.Name)
	}

	localRl := &localrlv3.LocalRateLimit{
		StatPrefix: localRateLimitFilterStatPrefix,
		TokenBucket: &typev3.TokenBucket{
			MaxTokens: uint32(local.Default.Requests),
			TokensPerFill: &wrapperspb.UInt32Value{
				Value: uint32(local.Default.Requests),
			},
			FillInterval: ratelimit.UnitToDuration(local.Default.Unit),
		},
		FilterEnabled: &configv3.RuntimeFractionalPercent{
			DefaultValue: &typev3.FractionalPercent{
				Numerator:   100,
				Denominator: typev3.FractionalPercent_HUNDRED,
			},
		},
		FilterEnforced: &configv3.RuntimeFractionalPercent{
			DefaultValue: &typev3.FractionalPercent{
				Numerator:   100,
				Denominator: typev3.FractionalPercent_HUNDRED,
			},
		},
		Descriptors: descriptors,
		// By setting AlwaysConsumeDefaultTokenBucket to false, the descriptors
		// won't consume the default token bucket. This means that a request only
		// counts towards the default token bucket if it does not match any of the
		// descriptors.
		AlwaysConsumeDefaultTokenBucket: &wrapperspb.BoolValue{
			Value: false,
		},
	}

	localRlAny, err := anypb.New(localRl)
	if err != nil {
		return err
	}

	if filterCfg == nil {
		route.TypedPerFilterConfig = make(map[string]*anypb.Any)
	}

	route.TypedPerFilterConfig[egv1a1.EnvoyFilterLocalRateLimit.String()] = localRlAny
	return nil
}

func buildRouteLocalRateLimits(local *ir.LocalRateLimit) (
	[]*routev3.RateLimit, []*rlv3.LocalRateLimitDescriptor,
) {
	var rateLimits []*routev3.RateLimit
	var descriptors []*rlv3.LocalRateLimitDescriptor

	// Rules are ORed
	for rIdx, rule := range local.Rules {
		var rlActions []*routev3.RateLimit_Action
		var descriptorEntries []*rlv3.RateLimitDescriptor_Entry

		// HeaderMatches
		for mIdx, match := range rule.HeaderMatches {
			var action *routev3.RateLimit_Action
			var entry *rlv3.RateLimitDescriptor_Entry

			if match.Distinct {
				// For distinct matches, we only check if the header exists using the RequestHeaders action.
				descriptorKey := getRouteRuleDescriptor(rIdx, mIdx)
				action = &routev3.RateLimit_Action{
					ActionSpecifier: &routev3.RateLimit_Action_RequestHeaders_{
						RequestHeaders: &routev3.RateLimit_Action_RequestHeaders{
							HeaderName:    match.Name,
							DescriptorKey: descriptorKey,
						},
					},
				}
				// The descriptor entry value is not set for distinct matches, which means that each distinct
				// value of the matched header will be counted separately.
				entry = &rlv3.RateLimitDescriptor_Entry{
					Key: descriptorKey,
				}
			} else {
				// For exact matches, we check if there is an existing header with the matching value using the
				// HeaderValueMatch action.
				descriptorKey := getRouteRuleDescriptor(rIdx, mIdx)
				descriptorVal := getRouteRuleDescriptor(rIdx, mIdx)
				headerMatcher := &routev3.HeaderMatcher{
					Name: match.Name,
					HeaderMatchSpecifier: &routev3.HeaderMatcher_StringMatch{
						StringMatch: buildXdsStringMatcher(match),
					},
				}
				expectMatch := true
				if match.Invert != nil && *match.Invert {
					expectMatch = false
				}
				action = &routev3.RateLimit_Action{
					ActionSpecifier: &routev3.RateLimit_Action_HeaderValueMatch_{
						HeaderValueMatch: &routev3.RateLimit_Action_HeaderValueMatch{
							DescriptorKey:   descriptorKey,
							DescriptorValue: descriptorVal,
							ExpectMatch: &wrapperspb.BoolValue{
								Value: expectMatch,
							},
							Headers: []*routev3.HeaderMatcher{headerMatcher},
						},
					},
				}
				// For exact matches, the descriptor entry value is set to the generated descriptor value.
				entry = &rlv3.RateLimitDescriptor_Entry{
					Key:   descriptorKey,
					Value: descriptorVal,
				}
			}
			rlActions = append(rlActions, action)
			descriptorEntries = append(descriptorEntries, entry)
		}

		// Source IP CIDRMatch
		if rule.CIDRMatch != nil {
			// For CIDR matches, we first need to check if the source IP matches the CIDR range using
			// the MaskedRemoteAddress action.
			mra := &routev3.RateLimit_Action_MaskedRemoteAddress{}
			maskLen := &wrapperspb.UInt32Value{Value: rule.CIDRMatch.MaskLen}
			if rule.CIDRMatch.IsIPv6 {
				mra.V6PrefixMaskLen = maskLen
			} else {
				mra.V4PrefixMaskLen = maskLen
			}
			action := &routev3.RateLimit_Action{
				ActionSpecifier: &routev3.RateLimit_Action_MaskedRemoteAddress_{
					MaskedRemoteAddress: mra,
				},
			}
			entry := &rlv3.RateLimitDescriptor_Entry{
				Key:   descriptorMaskedRemoteAddress,
				Value: rule.CIDRMatch.CIDR,
			}
			descriptorEntries = append(descriptorEntries, entry)
			rlActions = append(rlActions, action)

			if rule.CIDRMatch.Distinct {
				// If the CIDRMatch is distinct, we also need to use the RemoteAddress action to get the client IP.
				action = &routev3.RateLimit_Action{
					ActionSpecifier: &routev3.RateLimit_Action_RemoteAddress_{
						RemoteAddress: &routev3.RateLimit_Action_RemoteAddress{},
					},
				}

				// If the CIDRMatch is distinct, we use the built-in remote address descriptor key without a value.
				// This means that each distinct client IP will be counted separately.
				entry = &rlv3.RateLimitDescriptor_Entry{
					Key: descriptorRemoteAddress,
				}
				descriptorEntries = append(descriptorEntries, entry)
				rlActions = append(rlActions, action)
			}
		}

		rateLimit := &routev3.RateLimit{Actions: rlActions}
		rateLimits = append(rateLimits, rateLimit)

		descriptor := &rlv3.LocalRateLimitDescriptor{
			Entries: descriptorEntries,
			TokenBucket: &typev3.TokenBucket{
				MaxTokens: uint32(rule.Limit.Requests),
				TokensPerFill: &wrapperspb.UInt32Value{
					Value: uint32(rule.Limit.Requests),
				},
				FillInterval: ratelimit.UnitToDuration(rule.Limit.Unit),
			},
		}
		descriptors = append(descriptors, descriptor)
	}

	return rateLimits, descriptors
}
