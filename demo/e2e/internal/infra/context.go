package infra

import (
	"context"

	econtext "github.com/filariow/mctest/pkg/context"
	pinfra "github.com/filariow/mctest/pkg/infra"
)

const (
	// provisioners
	keyHostProvisioners   string = "host-provisioners"
	keyMemberProvisioners string = "member-provisioners"
	// clusters
	keyHostCluster string = "host-cluster"
	// namespaces
	keyHostScenarioNamespace string = "scenario-namespace-host"
)

// provisioners
func HostProvisionersIntoContext(ctx context.Context, value map[string]pinfra.ClusterProvisioner) context.Context {
	return econtext.IntoContext(ctx, keyHostProvisioners, value)
}

func HostProvisionersFromContext(ctx context.Context) (*map[string]pinfra.ClusterProvisioner, error) {
	return econtext.FromContext[map[string]pinfra.ClusterProvisioner](ctx, keyHostProvisioners)
}

func HostProvisionersFromContextOrDie(ctx context.Context) map[string]pinfra.ClusterProvisioner {
	return econtext.FromContextOrDie[map[string]pinfra.ClusterProvisioner](ctx, keyHostProvisioners)
}

// kubes
func HostClusterIntoContext(ctx context.Context, value Cluster) context.Context {
	return econtext.IntoContext(ctx, keyHostCluster, value)
}

func HostClusterFromContext(ctx context.Context) (*Cluster, error) {
	return econtext.FromContext[Cluster](ctx, keyHostCluster)
}

func HostClusterFromContextOrDie(ctx context.Context) Cluster {
	return econtext.FromContextOrDie[Cluster](ctx, keyHostCluster)
}

// namespaces
func HostScenarioNamespaceIntoContext(ctx context.Context, value string) context.Context {
	return econtext.IntoContext(ctx, keyHostScenarioNamespace, value)
}

func HostScenarioNamespaceFromContext(ctx context.Context) (*string, error) {
	return econtext.FromContext[string](ctx, keyHostScenarioNamespace)
}

func HostScenarioNamespaceFromContextOrDie(ctx context.Context) string {
	return econtext.FromContextOrDie[string](ctx, keyHostScenarioNamespace)
}
