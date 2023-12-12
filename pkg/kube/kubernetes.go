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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Kubernetes struct {
	Cfg    *rest.Config
	Scheme *runtime.Scheme

	Cli             *kubernetes.Clientset
	Dyn             *dynamic.DynamicClient
	CRCli           client.Client
	DiscoveryClient discovery.CachedDiscoveryInterface
	Mapper          *restmapper.DeferredDiscoveryRESTMapper

	IsDedicated bool
}

func New(cfg *rest.Config, scheme *runtime.Scheme, dedicated bool) (*Kubernetes, error) {
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	discoveryClient := cacheddiscovery.NewMemCacheClient(cli.Discovery())
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)

	opts := client.Options{Scheme: scheme, Mapper: mapper}
	crcli, err := client.New(cfg, opts)
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		Cfg:             cfg,
		Cli:             cli,
		Dyn:             dyn,
		CRCli:           crcli,
		DiscoveryClient: discoveryClient,
		Mapper:          mapper,
		IsDedicated:     dedicated,
	}, nil
}

func (k *Kubernetes) DynNamespacedResource(
	gvr schema.GroupVersionResource,
	obj interface{},
) (*unstructured.Unstructured, dynamic.ResourceInterface, error) {
	um, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, nil, err
	}
	u := &unstructured.Unstructured{Object: um}
	return u, k.Dyn.Resource(gvr).Namespace(u.GetNamespace()), nil
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

func (k *Kubernetes) BuildClientForResource(
	ctx context.Context,
	unstructuredObj unstructured.Unstructured,
) (dynamic.ResourceInterface, error) {
	return k.BuildClientForGroupVersionKind(
		ctx, unstructuredObj.GroupVersionKind())
}

func (k *Kubernetes) BuildNamespacedClientForResource(
	ctx context.Context,
	unstructuredObj unstructured.Unstructured,
	namespace string,
) (dynamic.ResourceInterface, error) {
	return k.BuildNamespacedClientForGroupVersionKind(
		ctx, unstructuredObj.GroupVersionKind(), namespace)
}

func (k *Kubernetes) BuildNamespacedClientForGroupVersionKind(
	ctx context.Context,
	gvk schema.GroupVersionKind,
	namespace string,
) (dynamic.ResourceInterface, error) {
	if namespace == "" {
		return nil, fmt.Errorf("empty namespaced provided, can not create namespaced client")
	}
	cli, err := k.BuildClientForGroupVersionKind(ctx, gvk)
	if err != nil {
		return nil, err
	}

	return cli.Namespace(namespace), nil
}

func (k *Kubernetes) BuildClientForGroupVersionKind(
	ctx context.Context,
	gvk schema.GroupVersionKind,
) (dynamic.NamespaceableResourceInterface, error) {
	gr, err := restmapper.GetAPIGroupResources(k.Cli.Discovery())
	if err != nil {
		return nil, fmt.Errorf("error building namespaced client: %w", err)
	}

	mapper := restmapper.NewDiscoveryRESTMapper(gr)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("error building namespaced client: %w", err)
	}

	return k.Dyn.Resource(mapping.Resource), nil
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
		if err := k.CreateNamespacedResourceUnstructured(ctx, u); err != nil {
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
			if err := k.CreateNamespacedResourceUnstructured(ictx, u); err != nil {
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
		dri, err := k.BuildClientForResource(ctx, u)
		if err != nil {
			return err
		}

		po, err := dri.Get(ctx, u.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		po.Object["spec"] = u.Object["spec"]
		if _, err := dri.Update(ctx, po, metav1.UpdateOptions{}); err != nil {
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
		if err := k.CRCli.Get(ctx, t, lu, &client.GetOptions{}); err != nil {
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
		dri, err := k.BuildClientForResource(ctx, u)
		if err != nil {
			return err
		}

		ctxd, cf := context.WithTimeout(ctx, 2*time.Minute)
		_, err = poll.DoR(ctxd, time.Second, func(ictx context.Context) (*unstructured.Unstructured, error) {
			lctx, lcf := context.WithTimeout(ictx, 1*time.Minute)
			defer lcf()

			if _, err := dri.Get(lctx, u.GetName(), metav1.GetOptions{}); err != nil {
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
	if _, err := k.Cli.CoreV1().Namespaces().Create(ctx, &ns, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

// unstructured
func (k *Kubernetes) CreateResourceUnstructured(ctx context.Context, u unstructured.Unstructured) error {
	dri, err := k.BuildClientForResource(ctx, u)
	if err != nil {
		return err
	}

	if _, err := dri.Create(ctx, &u, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (k *Kubernetes) CreateNamespacedResourceUnstructured(ctx context.Context, u unstructured.Unstructured) error {
	dri, err := k.BuildNamespacedClientForResource(ctx, u, u.GetNamespace())
	if err != nil {
		return fmt.Errorf("error building client for resource: %w\n%v", err, u)
	}

	if _, err := dri.Create(ctx, &u, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("error creating resource: %w\n%v", err, u)
	}
	return nil
}

func (k *Kubernetes) DeleteResourceUnstructured(ctx context.Context, u unstructured.Unstructured) error {
	dri, err := k.BuildClientForResource(ctx, u)
	if err != nil {
		return err
	}

	if err := dri.Delete(ctx, u.GetName(), metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func (k *Kubernetes) WatchResourceUnstructured(ctx context.Context, u unstructured.Unstructured) (watch.Interface, error) {
	dri, err := k.BuildClientForResource(ctx, u)
	if err != nil {
		return nil, err
	}

	o := metav1.ListOptions{
		TypeMeta:      metav1.TypeMeta{},
		FieldSelector: fmt.Sprintf("metadata.name=%s", u.GetName()),
	}
	return dri.Watch(ctx, o)
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
