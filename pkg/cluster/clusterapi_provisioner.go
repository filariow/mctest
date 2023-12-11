package cluster

import (
	"context"
	"fmt"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/filariow/mctest/pkg/infra"
	"github.com/filariow/mctest/pkg/kube"
	"github.com/filariow/mctest/pkg/poll"
)

const clusterKind string = "Cluster"

var ErrClusterNotFound error = fmt.Errorf("error cluster not found")

type ClusterAPIProvisioner struct {
	Kubernetes kube.Kubernetes
	Manifests  []unstructured.Unstructured
	Suffix     *string
}

func NewClusterAPIProvisioner(
	kubernetes kube.Kubernetes,
	manifests []unstructured.Unstructured,
	suffix *string,
) infra.ClusterProvisioner {
	return &ClusterAPIProvisioner{
		Kubernetes: kubernetes,
		Manifests:  manifests,
		Suffix:     suffix,
	}
}

// returns the kubeconfig for a given provisioned cluster
func (p *ClusterAPIProvisioner) GetAllAdminKubeconfigs(ctx context.Context) (map[string]rest.Config, error) {
	// create resources
	cc, _ := p.manifests()

	// fetch clusters rest.config
	cfgs := map[string]rest.Config{}
	for _, u := range cc {
		sn := fmt.Sprintf("%s-kubeconfig", u.GetName())
		lctx, lcancel := context.WithTimeout(ctx, 1*time.Minute)
		defer lcancel()
		cfg, err := poll.DoR(lctx, 10*time.Second, func(ctx context.Context) (*rest.Config, error) {
			s, err := p.Kubernetes.Cli.CoreV1().Secrets(u.GetNamespace()).Get(ctx, sn, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}

			// etract kubeconfig from secret and build rest.Config
			return clientcmd.RESTConfigFromKubeConfig(s.Data["value"])
		})
		if err != nil {
			return nil, err
		}
		cfgs[u.GetName()] = *cfg
	}

	return cfgs, nil
}

// returns the kubeconfigs for all provisioned clusters
func (p *ClusterAPIProvisioner) GetAdminKubeconfig(ctx context.Context, cluster string) (*rest.Config, error) {
	// create resources
	cc, _ := p.manifests()

	for _, u := range cc {
		// needed to ensure a cluster provisioned with this provisioner is requested
		if u.GetName() != cluster {
			continue
		}

		sn := fmt.Sprintf("%s-kubeconfig", u.GetName())
		lctx, lcancel := context.WithTimeout(ctx, 1*time.Minute)
		defer lcancel()
		cfg, err := poll.DoR(lctx, 10*time.Second, func(ctx context.Context) (*rest.Config, error) {
			s, err := p.Kubernetes.Cli.CoreV1().Secrets(u.GetNamespace()).Get(ctx, sn, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}

			// etract kubeconfig from secret and build rest.Config
			return clientcmd.RESTConfigFromKubeConfig(s.Data["value"])
		})
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrClusterNotFound, cluster)

}

// Provision provisions the cluster api manifests for a new cluster
// It will create Clusters as lasts.
func (p *ClusterAPIProvisioner) Provision(ctx context.Context) error {
	return p.provisionWithSuffix(ctx, nil)
}

// Provisions a cluster appending the given suffix to resources name
func (p *ClusterAPIProvisioner) ProvisionWithSuffix(ctx context.Context, suffix string) error {
	return p.provisionWithSuffix(ctx, &suffix)
}

func (p *ClusterAPIProvisioner) provisionWithSuffix(ctx context.Context, suffix *string) error {
	// create resources
	cc, oo := p.manifests()

	// create other resources
	for _, u := range oo {
		if err := p.Kubernetes.CreateNamespacedResourceUnstructured(ctx, u); err != nil {
			return fmt.Errorf("error creating namespaced ClusterAPI resource:%w\n%v", err, u)
		}
	}

	// create clusters
	for _, u := range cc {
		n := fmt.Sprintf("%s-%s", u.GetName(), *p.Suffix)
		if suffix != nil {
			n += *suffix
		}
		u.SetName(n)

		if err := p.Kubernetes.CreateNamespacedResourceUnstructured(ctx, u); err != nil {
			return fmt.Errorf("error creating namespaced ClusterAPI Cluster resource:%w\n%v", err, u)
		}
	}

	// wait for cluster status to be ready
	for _, u := range cc {
		if err := p.Kubernetes.WatchForEventOnResourceUnstructured(ctx, u, func(e watch.Event) (bool, error) {
			// convert to unstructured
			m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(e.Object)
			if err != nil {
				return false, err
			}

			a, ok := m["status"]
			if !ok {
				// resource has not been reconciled
				return false, nil
			}

			am, ok := a.(map[string]interface{})
			if !ok {
				return false, fmt.Errorf("'status' is not a map[string]interface{}")
			}

			p, ok := am["phase"]
			if !ok {
				// resource has not been reconciled
				return false, nil
			}

			ps, ok := p.(string)
			if !ok {
				return false, fmt.Errorf("field 'status.phase' is not a string")
			}

			return ps == "Provisioned", nil
		}); err != nil {
			return fmt.Errorf("error watching for cluster '%s/%s' status: %w", u.GetNamespace(), u.GetName(), err)
		}
	}
	return nil
}

// Returns the number of cluster that will be created in a single execution of Provision.
// It is needed for managing tunable-manifests provisioners, like the ClusterAPI one.
func (p *ClusterAPIProvisioner) NumClustersProvisionedInProvisionRound() int {
	cc, _ := p.manifests()
	return len(cc)
}

// Unprovisions the clusters previously provisioned.
// It will delete Clusters as firsts, the other CRs
func (p *ClusterAPIProvisioner) Unprovision(ctx context.Context) error {
	// find clusters
	cc, oo := p.manifests()

	// delete clusters before other to avoid deletion errors
	tw := []unstructured.Unstructured{}
	for _, c := range cc {
		err := p.Kubernetes.DeleteResourceUnstructured(ctx, c)
		switch {
		case err == nil:
			tw = append(tw, c) // fill the list of ones to wait for deletion
			continue
		case kerrors.IsNotFound(err):
			continue
		default:
			return err
		}
	}

	// TODO: enhance this, if deleted right before watching it will wait indefinitely
	// wait for clusters deletion
	for _, c := range tw {
		c.SetName(fmt.Sprintf("%s-%s", c.GetName(), *p.Suffix))
		if err := p.Kubernetes.WaitForDeletionOfResourceUnstructured(ctx, c); !kerrors.IsNotFound(err) {
			return err
		}
	}

	// delete other resources
	for _, u := range oo {
		if err := p.Kubernetes.DeleteResourceUnstructured(ctx, u); !kerrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// auxiliaries
func (p *ClusterAPIProvisioner) manifests() ([]unstructured.Unstructured, []unstructured.Unstructured) {
	cc, oo := []unstructured.Unstructured{}, []unstructured.Unstructured{}
	for _, u := range p.Manifests {
		if u.GetKind() == clusterKind {
			cc = append(cc, u)
		} else {
			oo = append(oo, u)
		}
	}
	return cc, oo
}
