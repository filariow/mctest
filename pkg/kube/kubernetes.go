package kube

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/filariow/mctest/pkg/poll"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
}

type Kubernetes struct {
	client.WithWatch

	cfg             *rest.Config
	clientOptions   client.Options
	discoveryClient discovery.CachedDiscoveryInterface
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
	crcli, err := NewNamespacedClient(cfg, opts, namespace)
	if err != nil {
		return nil, err
	}
	return &NamespacedKubernetes{
		Kubernetes: Kubernetes{
			WithWatch:       crcli,
			cfg:             cfg,
			clientOptions:   opts,
			discoveryClient: discoveryClient,
		},
		Namespace: namespace,
	}, nil

	return nil, nil
}

func New(cfg *rest.Config, opts client.Options) (*Kubernetes, error) {
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	discoveryClient := cacheddiscovery.NewMemCacheClient(cli.Discovery())
	crcli, err := client.NewWithWatch(cfg, opts)
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		WithWatch:       crcli,
		cfg:             cfg,
		clientOptions:   opts,
		discoveryClient: discoveryClient,
	}, nil
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
func (k *Kubernetes) ResourcesAreCreated(ctx context.Context, spec string) error {
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		return k.CreateResourceUnstructured(ctx, u)
	}

	return nil
}

func (k *Kubernetes) NamespacedResourcesAreCreated(ctx context.Context, spec string) error {
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		if err := k.Create(ctx, &u, &client.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (k *Kubernetes) ResourcesAreCreatedInNamespace(ctx context.Context, namespace, spec string) error {
	return poll.Do(ctx, time.Second, func(ictx context.Context) error {
		uu, err := k.ParseResources(ictx, spec)
		if err != nil {
			return err
		}

		for _, u := range uu {
			u.SetNamespace(namespace)
			if err := k.Create(ictx, &u, &client.CreateOptions{}); err != nil {
				return err
			}
		}

		return nil
	})
}

func (k *Kubernetes) ResourcesAreUpdated(ctx context.Context, spec string) error {
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		if err := k.Update(ctx, u.DeepCopy(), &client.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (k *Kubernetes) ResourcesExistInNamespace(ctx context.Context, namespace, spec string) error {
	return k.resourcesExist(ctx, spec, &namespace)
}

func (k *Kubernetes) ResourcesExist(ctx context.Context, spec string) error {
	return k.resourcesExist(ctx, spec, nil)
}

func (k *Kubernetes) resourcesExist(ctx context.Context, spec string, ns *string) error {
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	errs := []error{}
	for _, u := range uu {
		if err := k.resourceExist(ctx, u, ns); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (k *Kubernetes) resourceExist(ctx context.Context, u unstructured.Unstructured, ns *string) error {
	ctxd, cf := context.WithTimeout(ctx, 2*time.Minute)
	defer cf()

	_, err := poll.DoR(ctxd, time.Second, func(ictx context.Context) (*unstructured.Unstructured, error) {
		lu := u.DeepCopy()
		if ns != nil {
			lu.SetNamespace(*ns)
		}

		t := types.NamespacedName{Namespace: lu.GetNamespace(), Name: lu.GetName()}
		if err := k.Get(ctx, t, lu, &client.GetOptions{}); err != nil {
			return nil, err
		}
		return lu, nil
	})
	if err != nil {
		log.Printf("error retrieving resource %v: %v", u.Object, err)
		return err
	}
	return nil
}

func (k *Kubernetes) ResourcesNotExist(ctx context.Context, spec string) error {
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		ctxd, cf := context.WithTimeout(ctx, 2*time.Minute)
		_, err = poll.DoR(ctxd, time.Second, func(ictx context.Context) (*unstructured.Unstructured, error) {
			lctx, lcf := context.WithTimeout(ictx, 1*time.Minute)
			defer lcf()

			t := types.NamespacedName{Namespace: u.GetNamespace(), Name: u.GetName()}
			lu := u.DeepCopy()
			if err := k.Get(lctx, t, lu, &client.GetOptions{}); err != nil {
				if kerrors.IsNotFound(err) {
					return nil, nil
				}
			}
			return nil, err
		})
		cf()

		if err != nil {
			ld, err := u.MarshalJSON()
			if err != nil {
				return fmt.Errorf(
					"resource exists: [ ApiVersion=%s, Kind=%s, Namespace=%s, Name=%s ]. Error marshaling as json: %w",
					u.GetAPIVersion(), u.GetKind(), u.GetNamespace(), u.GetName(), err)
			}
			return fmt.Errorf("resource exists: %s", ld)
		}
	}

	return nil
}

func (k *Kubernetes) CreateNamespace(ctx context.Context, namespace string) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	if err := k.Create(ctx, &ns, &client.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (k *Kubernetes) CreateNamespaceWithLabels(ctx context.Context, namespace string, labels map[string]string) (*corev1.Namespace, error) {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: labels,
		},
	}
	if err := k.Create(ctx, &ns, &client.CreateOptions{}); err != nil {
		return nil, err
	}
	return &ns, nil
}

// unstructured
func (k *Kubernetes) CreateResourceUnstructured(ctx context.Context, u unstructured.Unstructured) error {
	if err := k.Create(ctx, &u, &client.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (k *Kubernetes) DeleteResourceUnstructured(ctx context.Context, u unstructured.Unstructured) error {
	if err := k.Delete(ctx, &u, &client.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func (k *Kubernetes) WatchResourceUnstructured(ctx context.Context, u unstructured.Unstructured) (watch.Interface, error) {
	fs, err := fields.ParseSelector(fmt.Sprintf("metadata.name=%s", u.GetName()))
	if err != nil {
		return nil, err
	}

	o := &client.ListOptions{
		FieldSelector: fs,
	}
	return k.Watch(ctx, &u, o)
}

func (k *Kubernetes) WaitForDeletionOfResourceUnstructured(ctx context.Context, u unstructured.Unstructured) error {
	return k.WatchForEventOnResourceUnstructured(ctx, u, func(e watch.Event) (bool, error) {
		return e.Type == watch.Deleted, nil
	})
}

func (k *Kubernetes) WatchForEventOnResourceUnstructured(ctx context.Context, u unstructured.Unstructured, check func(e watch.Event) (bool, error)) error {
	// watch resource
	w, err := k.WatchResourceUnstructured(ctx, u)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	defer w.Stop()

	// check for event to happen
	for {
		e := <-w.ResultChan()

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
