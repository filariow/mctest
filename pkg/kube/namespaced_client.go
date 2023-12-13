package kube

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ client.WithWatch = &NamespacedClient{}
var _ client.SubResourceClient = &subResourceClient{}

type NamespacedClient struct {
	cli client.WithWatch

	namespace string
}

func NewNamespacedClient(
	cfg *rest.Config,
	opts client.Options,
	namespace string,
) (*NamespacedClient, error) {
	cli, err := client.NewWithWatch(cfg, opts)
	if err != nil {
		return nil, err
	}

	nc := &NamespacedClient{
		cli:       cli,
		namespace: namespace,
	}
	return nc, nil
}

func (c *NamespacedClient) GetNamespace() string { return c.namespace }

// Implement client.Writer

// Create saves the object obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c *NamespacedClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	obj.SetNamespace(c.namespace)
	return c.cli.Create(ctx, obj, opts...)
}

// Delete deletes the given obj from Kubernetes cluster.
func (c *NamespacedClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	obj.SetNamespace(c.namespace)
	return c.cli.Delete(ctx, obj, opts...)
}

// Update updates the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c *NamespacedClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	obj.SetNamespace(c.namespace)
	return c.cli.Update(ctx, obj, opts...)
}

// Patch patches the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c *NamespacedClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	obj.SetNamespace(c.namespace)
	return c.cli.Patch(ctx, obj, patch, opts...)
}

// DeleteAllOf deletes all objects of the given type matching the given options.
func (c *NamespacedClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	obj.SetNamespace(c.namespace)
	return c.cli.DeleteAllOf(ctx, obj, opts...)
}

// Implement client.Reader

// Get retrieves an obj for the given object key from the Kubernetes Cluster.
// obj must be a struct pointer so that obj can be updated with the response
// returned by the Server.
func (c *NamespacedClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	key.Namespace = c.namespace
	return c.cli.Get(ctx, key, obj, opts...)
}

type namespacedListOptions struct {
	namespace string
}

// ApplyToList applies this configuration to the given list options.
func (o *namespacedListOptions) ApplyToList(l *client.ListOptions) {
	l.Namespace = o.namespace
}

// List retrieves list of objects for a given namespace and list options. On a
// successful call, Items field in the list will be populated with the
// result returned from the server.
func (c *NamespacedClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	opts = append(opts, &namespacedListOptions{namespace: c.namespace})
	return c.cli.List(ctx, list, opts...)
}

// Implement Watch

func (c *NamespacedClient) Watch(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) (watch.Interface, error) {
	return c.cli.Watch(ctx, obj, opts...)
}

// other interfaces
func (c *NamespacedClient) Status() client.SubResourceWriter {
	return c.SubResource("status")
}

// SubResourceClientConstructor returns a subresource client for the named subResource. Known
// upstream subResources usages are:
//
//   - ServiceAccount token creation:
//     sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     token := &authenticationv1.TokenRequest{}
//     c.SubResourceClient("token").Create(ctx, sa, token)
//
//   - Pod eviction creation:
//     pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     c.SubResourceClient("eviction").Create(ctx, pod, &policyv1.Eviction{})
//
//   - Pod binding creation:
//     pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     binding := &corev1.Binding{Target: corev1.ObjectReference{Name: "my-node"}}
//     c.SubResourceClient("binding").Create(ctx, pod, binding)
//
//   - CertificateSigningRequest approval:
//     csr := &certificatesv1.CertificateSigningRequest{
//     ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"},
//     Status: certificatesv1.CertificateSigningRequestStatus{
//     Conditions: []certificatesv1.[]CertificateSigningRequestCondition{{
//     Type: certificatesv1.CertificateApproved,
//     Status: corev1.ConditionTrue,
//     }},
//     },
//     }
//     c.SubResourceClient("approval").Update(ctx, csr)
//
//   - Scale retrieval:
//     dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     scale := &autoscalingv1.Scale{}
//     c.SubResourceClient("scale").Get(ctx, dep, scale)
//
//   - Scale update:
//     dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     scale := &autoscalingv1.Scale{Spec: autoscalingv1.ScaleSpec{Replicas: 2}}
//     c.SubResourceClient("scale").Update(ctx, dep, client.WithSubResourceBody(scale))
func (c *NamespacedClient) SubResource(subResource string) client.SubResourceClient {
	return &subResourceClient{client: c.cli.SubResource(subResource)}
}

// Scheme returns the scheme this client is using.
func (c *NamespacedClient) Scheme() *runtime.Scheme {
	return c.cli.Scheme()
}

// RESTMapper returns the rest this client is using.
func (c *NamespacedClient) RESTMapper() meta.RESTMapper {
	return c.cli.RESTMapper()
}

// GroupVersionKindFor returns the GroupVersionKind for the given object.
func (c *NamespacedClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return c.cli.GroupVersionKindFor(obj)
}

// IsObjectNamespaced returns true if the GroupVersionKind of the object is namespaced.
func (c *NamespacedClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return c.cli.IsObjectNamespaced(obj)
}

// type subResourceClient
type subResourceClient struct {
	client client.SubResourceClient

	namespace string
}

func (c *subResourceClient) Get(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceGetOption) error {
	obj.SetNamespace(c.namespace)
	return c.client.Get(ctx, obj, subResource, opts...)
}

// Create saves the subResource object in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c *subResourceClient) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	obj.SetNamespace(c.namespace)
	return c.client.Create(ctx, obj, subResource, opts...)
}

// Update updates the fields corresponding to the status subresource for the
// given obj. obj must be a struct pointer so that obj can be updated
// with the content returned by the Server.
func (c *subResourceClient) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	obj.SetNamespace(c.namespace)
	return c.client.Update(ctx, obj, opts...)
}

// Patch patches the given object's subresource. obj must be a struct
// pointer so that obj can be updated with the content returned by the
// Server.
func (c *subResourceClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	obj.SetNamespace(c.namespace)
	return c.client.Patch(ctx, obj, patch, opts...)
}
