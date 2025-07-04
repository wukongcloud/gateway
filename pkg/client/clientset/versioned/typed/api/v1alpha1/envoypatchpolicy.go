// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	context "context"

	apiv1alpha1 "github.com/wukongcloud/gateway/api/v1alpha1"
	scheme "github.com/wukongcloud/gateway/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// EnvoyPatchPoliciesGetter has a method to return a EnvoyPatchPolicyInterface.
// A group's client should implement this interface.
type EnvoyPatchPoliciesGetter interface {
	EnvoyPatchPolicies(namespace string) EnvoyPatchPolicyInterface
}

// EnvoyPatchPolicyInterface has methods to work with EnvoyPatchPolicy resources.
type EnvoyPatchPolicyInterface interface {
	Create(ctx context.Context, envoyPatchPolicy *apiv1alpha1.EnvoyPatchPolicy, opts v1.CreateOptions) (*apiv1alpha1.EnvoyPatchPolicy, error)
	Update(ctx context.Context, envoyPatchPolicy *apiv1alpha1.EnvoyPatchPolicy, opts v1.UpdateOptions) (*apiv1alpha1.EnvoyPatchPolicy, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, envoyPatchPolicy *apiv1alpha1.EnvoyPatchPolicy, opts v1.UpdateOptions) (*apiv1alpha1.EnvoyPatchPolicy, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*apiv1alpha1.EnvoyPatchPolicy, error)
	List(ctx context.Context, opts v1.ListOptions) (*apiv1alpha1.EnvoyPatchPolicyList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *apiv1alpha1.EnvoyPatchPolicy, err error)
	EnvoyPatchPolicyExpansion
}

// envoyPatchPolicies implements EnvoyPatchPolicyInterface
type envoyPatchPolicies struct {
	*gentype.ClientWithList[*apiv1alpha1.EnvoyPatchPolicy, *apiv1alpha1.EnvoyPatchPolicyList]
}

// newEnvoyPatchPolicies returns a EnvoyPatchPolicies
func newEnvoyPatchPolicies(c *GatewayV1alpha1Client, namespace string) *envoyPatchPolicies {
	return &envoyPatchPolicies{
		gentype.NewClientWithList[*apiv1alpha1.EnvoyPatchPolicy, *apiv1alpha1.EnvoyPatchPolicyList](
			"envoypatchpolicies",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *apiv1alpha1.EnvoyPatchPolicy { return &apiv1alpha1.EnvoyPatchPolicy{} },
			func() *apiv1alpha1.EnvoyPatchPolicyList { return &apiv1alpha1.EnvoyPatchPolicyList{} },
		),
	}
}
