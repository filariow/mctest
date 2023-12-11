package infra

import (
	"context"

	"k8s.io/client-go/rest"
)

type ClusterProvisioner interface {
	// returns the kubeconfig for a given provisioned cluster
	GetAdminKubeconfig(ctx context.Context, cluster string) (*rest.Config, error)
	// returns the kubeconfigs for all provisioned clusters
	GetAllAdminKubeconfigs(ctx context.Context) (map[string]rest.Config, error)
	// Provision a cluster
	Provision(ctx context.Context) error
	// Provisions a cluster appending the given suffix to resources name
	ProvisionWithSuffix(ctx context.Context, suffix string) error
	// Returns the number of cluster that will be created in a single execution of Provision.
	// It is needed for managing tunable-manifests provisioners, like the ClusterAPI one.
	NumClustersProvisionedInProvisionRound() int
	// Unprovision unprovisions the clusters previously provisioned.
	Unprovision(ctx context.Context) error
}
