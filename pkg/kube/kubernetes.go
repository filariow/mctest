package kube

import (
	"context"
	"fmt"
	"log"
	"strings"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ Client = &Kubernetes{}
var _ Client = &NamespacedKubernetes{}

type Client interface {
	client.WithWatch

	Clientset() (*kubernetes.Clientset, error)
	DiscoveryClient() discovery.CachedDiscoveryInterface
	ClientOptions() client.Options
	RESTConfig() *rest.Config

	DeleteAndWait(ctx context.Context, u unstructured.Unstructured, opts client.DeleteOption) error
	ParseResources(ctx context.Context, spec string) ([]unstructured.Unstructured, error)
	WaitForDeletionOfResourceUnstructured(ctx context.Context, u unstructured.Unstructured) error
	WatchForEventOnResourceUnstructured(ctx context.Context, u unstructured.Unstructured, check func(e watch.Event) (bool, error)) error
	// CreateNamespaceWithLabels(ctx context.Context, namespace string, labels map[string]string) (*corev1.Namespace, error)

	Livez(ctx context.Context) ([]byte, error)
	Healthz(ctx context.Context) ([]byte, error)
}

type Kubernetes struct {
	client.WithWatch

	cfg             *rest.Config
	clientOptions   client.Options
	discoveryClient discovery.CachedDiscoveryInterface
	mapper          *restmapper.DeferredDiscoveryRESTMapper
}

type NamespacedKubernetes struct {
	Kubernetes

	Namespace string
}

func NewNamespaced(cfg *rest.Config, opts client.Options, namespace string) (*NamespacedKubernetes, error) {
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	discoveryClient := cacheddiscovery.NewMemCacheClient(cli.Discovery())
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)

	crcli, err := NewNamespacedClient(cfg, opts, namespace)
	if err != nil {
		return nil, err
	}
	return &NamespacedKubernetes{
		Kubernetes: Kubernetes{
			WithWatch:       crcli,
			mapper:          mapper,
			cfg:             cfg,
			clientOptions:   opts,
			discoveryClient: discoveryClient,
		},
		Namespace: namespace,
	}, nil
}

func New(cfg *rest.Config, opts client.Options) (*Kubernetes, error) {
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	discoveryClient := cacheddiscovery.NewMemCacheClient(cli.Discovery())
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	crcli, err := client.NewWithWatch(cfg, opts)
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		WithWatch:       crcli,
		mapper:          mapper,
		cfg:             cfg,
		clientOptions:   opts,
		discoveryClient: discoveryClient,
	}, nil
}

func (k *NamespacedKubernetes) ParseResources(ctx context.Context, spec string) ([]unstructured.Unstructured, error) {
	uu, err := k.Kubernetes.ParseResources(ctx, spec)
	if err != nil {
		return nil, err
	}

	for _, u := range uu {
		u.SetNamespace(k.Namespace)
	}
	return uu, nil
}

func (k *NamespacedKubernetes) WaitForDeletionOfResourceUnstructured(ctx context.Context, u unstructured.Unstructured) error {
	lu := u.DeepCopy()
	lu.SetNamespace(k.Namespace)
	return k.Kubernetes.WaitForDeletionOfResourceUnstructured(ctx, *lu)
}

func (k *NamespacedKubernetes) WatchForEventOnResourceUnstructured(ctx context.Context, u unstructured.Unstructured, check func(e watch.Event) (bool, error)) error {
	lu := u.DeepCopy()
	lu.SetNamespace(k.Namespace)
	return k.Kubernetes.WatchForEventOnResourceUnstructured(ctx, *lu, check)
}

func (k *Kubernetes) Clientset() (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(k.RESTConfig())
}

func (k *Kubernetes) DiscoveryClient() discovery.CachedDiscoveryInterface {
	return k.discoveryClient
}

func (k *Kubernetes) RESTConfig() *rest.Config {
	return &(*k.cfg)
}

func (k *Kubernetes) ClientOptions() client.Options {
	return k.clientOptions
}

func (k *Kubernetes) Livez(ctx context.Context) ([]byte, error) {
	return k.DiscoveryClient().RESTClient().Get().AbsPath("/livez").DoRaw(ctx)
}

func (k *Kubernetes) Healthz(ctx context.Context) ([]byte, error) {
	return k.DiscoveryClient().RESTClient().Get().AbsPath("/healthz").DoRaw(ctx)
}

func (k *Kubernetes) ParseResources(ctx context.Context, spec string) ([]unstructured.Unstructured, error) {
	uu := []unstructured.Unstructured{}
	decoder := yamlutil.NewYAMLOrJSONDecoder(strings.NewReader(spec), 100)
	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			break
		}

		obj, _, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, rawObj.Raw)
		}

		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, err
		}

		unstructuredObj := unstructured.Unstructured{Object: unstructuredMap}
		uu = append(uu, unstructuredObj)
	}

	return uu, nil
}

// steps
func (k *Kubernetes) WatchResourceUnstructured(ctx context.Context, u unstructured.Unstructured) (watch.Interface, error) {
	fs, err := fields.ParseSelector(fmt.Sprintf("metadata.name=%s,metadata.namespace=%s", u.GetName(), u.GetNamespace()))
	if err != nil {
		return nil, err
	}

	o := &client.ListOptions{
		FieldSelector: fs,
	}
	return k.Watch(ctx, &u, o)
}

func (k *Kubernetes) DeleteAndWait(ctx context.Context, u unstructured.Unstructured, opts client.DeleteOption) error {
	r := u.DeepCopy()
	t := types.NamespacedName{Namespace: u.GetNamespace(), Name: u.GetName()}
	if err := k.Get(ctx, t, r, &client.GetOptions{}); err != nil {
		return err
	}

	we := make(chan error, 1)
	go func() {
		if err := k.WatchForEventOnResourceUnstructured(ctx, *r, func(e watch.Event) (bool, error) {
			return e.Type == watch.Deleted, nil
		}); err != nil {
			we <- err
		}
		defer close(we)
	}()

	if err := k.Delete(ctx, r, opts); err != nil {
		return err
	}

	return <-we
}

func (k *Kubernetes) WaitForDeletionOfResourceUnstructured(ctx context.Context, u unstructured.Unstructured) error {
	return k.WatchForEventOnResourceUnstructured(ctx, u, func(e watch.Event) (bool, error) {
		return e.Type == watch.Deleted, nil
	})
}

func (k *Kubernetes) WatchForEventOnResourceUnstructured(ctx context.Context, u unstructured.Unstructured, check func(e watch.Event) (bool, error)) error {
	m, err := k.mapper.RESTMapping(u.GroupVersionKind().GroupKind(), u.GroupVersionKind().Version)
	if err != nil {
		return err
	}

	// watch resource
	w, err := k.WatchResourceUnstructured(ctx, u)
	if err != nil {
		return err
	}
	defer w.Stop()

	log.Printf("watching events on resource %s/%s", u.GetNamespace(), u.GetName())

	// check for event to happen
	for {
		e, ok := <-w.ResultChan()
		if !ok {
			err := kerrors.NewNotFound(m.Resource.GroupResource(), u.GetName())
			log.Printf("watch result chan is closed, returning not found: %v", err)
			return err
		}

		log.Printf("received %v event on resource %s/%s: %v", e.Type, u.GetName(), u.GetNamespace(), u.Object)
		ok, err := check(e)
		if err != nil {
			return err
		}

		if ok {
			break
		}
	}

	return nil
}
