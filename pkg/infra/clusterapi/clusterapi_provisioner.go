package clusterapi

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/filariow/mctest/pkg/infra"
	"github.com/filariow/mctest/pkg/kube"
	"github.com/filariow/mctest/pkg/poll"
)

const clusterKind string = "Cluster"

var ErrClusterNotFound error = fmt.Errorf("error cluster not found")

type ClusterAPIProvisioner struct {
	Kubernetes kube.Client
	Manifests  []unstructured.Unstructured

	suffix     string
	clusters   []unstructured.Unstructured
	clusterDef []unstructured.Unstructured
}

func NewClusterAPIProvisioner(
	kubernetes kube.Client,
	manifests []unstructured.Unstructured,
	suffix string,
) infra.ClusterProvisioner {
	cc, mm := splitManifests(manifests, &suffix)

	return &ClusterAPIProvisioner{
		Kubernetes: kubernetes,
		Manifests:  manifests,

		clusters:   cc,
		clusterDef: mm,
		suffix:     suffix,
	}
}

func (p *ClusterAPIProvisioner) WaitForProvisionedClusters(ctx context.Context) error {
	return poll.Do(ctx, 5*time.Second, func(ctx context.Context) error {
		if err := func() error {
			cfgs, err := p.GetAllAdminKubeconfigs(ctx)
			if err != nil {
				return err
			}

			for _, cfg := range cfgs {
				k, err := kube.New(&cfg, client.Options{Scheme: scheme.Scheme})
				if err != nil {
					return err
				}

				lc, err := k.Livez(ctx)
				if err != nil {
					return err
				}
				log.Printf("livez called successfully, response is: %v", string(lc))

				hc, err := k.Healthz(ctx)
				if err != nil {
					return err
				}
				log.Printf("healthz called successfully, response is: %v", string(hc))
			}

			return nil
		}(); err != nil {
			log.Printf("error checking if cluster is provisioned: %v", err)
			return err
		}
		return nil
	})
}

// returns the kubeconfig for a given provisioned cluster
func (p *ClusterAPIProvisioner) GetAllAdminKubeconfigs(ctx context.Context) (map[string]rest.Config, error) {
	// fetch clusters rest.config
	cfgs := map[string]rest.Config{}
	for _, u := range p.clusters {
		sn := fmt.Sprintf("%s-kubeconfig", u.GetName())
		lctx, lcancel := context.WithTimeout(ctx, 1*time.Minute)
		defer lcancel()

		cfg, err := poll.DoR(lctx, 10*time.Second, func(ctx context.Context) (*rest.Config, error) {
			s := corev1.Secret{}
			t := types.NamespacedName{Namespace: u.GetNamespace(), Name: sn}
			if err := p.Kubernetes.Get(ctx, t, &s, &client.GetOptions{}); err != nil {
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

// Provision provisions the cluster api manifests for a new cluster
// It will create Clusters as lasts.
func (p *ClusterAPIProvisioner) Provision(ctx context.Context) error {
	// create other resources
	for _, u := range p.clusterDef {
		if err := p.Kubernetes.Create(ctx, u.DeepCopy(), &client.CreateOptions{}); err != nil {
			return fmt.Errorf("error creating namespaced ClusterAPI resource:%w\n%v", err, u)
		}
	}

	// create clusters
	for _, u := range p.clusters {
		if err := p.Kubernetes.Create(ctx, u.DeepCopy(), &client.CreateOptions{}); err != nil {
			return fmt.Errorf("error creating namespaced ClusterAPI Cluster resource:%w\n%v", err, u)
		}
	}

	// wait for cluster status to be ready
	for _, u := range p.clusters {
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
	return len(p.clusters)
}

// Unprovisions the clusters previously provisioned.
// It will delete Clusters as firsts, the other CRs
func (p *ClusterAPIProvisioner) Unprovision(ctx context.Context) error {
	// delete clusters before other to avoid deletion errors
	tw := []unstructured.Unstructured{}
	for _, c := range p.clusters {
		err := p.Kubernetes.DeleteAndWait(ctx, c, &client.DeleteOptions{})
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

	log.Println("deleting ClusterAPI CRs")
	// delete other resources
	for _, u := range p.clusterDef {
		if err := p.Kubernetes.Delete(ctx, u.DeepCopy(), &client.DeleteOptions{}); !kerrors.IsNotFound(err) {
			return err
		}
	}
	log.Println("deleted ClusterAPI CRs")
	return nil
}

// auxiliaries
func splitManifests(manifests []unstructured.Unstructured, clusterSuffix *string) ([]unstructured.Unstructured, []unstructured.Unstructured) {
	cc, oo := []unstructured.Unstructured{}, []unstructured.Unstructured{}
	for _, u := range manifests {
		l := u.DeepCopy()
		if u.GetKind() == clusterKind {
			if clusterSuffix != nil {
				l.SetName(fmt.Sprintf("%s-%s", l.GetName(), *clusterSuffix))
			}
			cc = append(cc, *l)
		} else {
			oo = append(oo, *l)
		}
	}
	return cc, oo
}
