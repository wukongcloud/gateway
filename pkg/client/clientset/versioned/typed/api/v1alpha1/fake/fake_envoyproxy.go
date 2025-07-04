// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/wukongcloud/gateway/api/v1alpha1"
	apiv1alpha1 "github.com/wukongcloud/gateway/pkg/client/clientset/versioned/typed/api/v1alpha1"
	gentype "k8s.io/client-go/gentype"
)

// fakeEnvoyProxies implements EnvoyProxyInterface
type fakeEnvoyProxies struct {
	*gentype.FakeClientWithList[*v1alpha1.EnvoyProxy, *v1alpha1.EnvoyProxyList]
	Fake *FakeGatewayV1alpha1
}

func newFakeEnvoyProxies(fake *FakeGatewayV1alpha1, namespace string) apiv1alpha1.EnvoyProxyInterface {
	return &fakeEnvoyProxies{
		gentype.NewFakeClientWithList[*v1alpha1.EnvoyProxy, *v1alpha1.EnvoyProxyList](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("envoyproxies"),
			v1alpha1.SchemeGroupVersion.WithKind("EnvoyProxy"),
			func() *v1alpha1.EnvoyProxy { return &v1alpha1.EnvoyProxy{} },
			func() *v1alpha1.EnvoyProxyList { return &v1alpha1.EnvoyProxyList{} },
			func(dst, src *v1alpha1.EnvoyProxyList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.EnvoyProxyList) []*v1alpha1.EnvoyProxy { return gentype.ToPointerSlice(list.Items) },
			func(list *v1alpha1.EnvoyProxyList, items []*v1alpha1.EnvoyProxy) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
