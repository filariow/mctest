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
func ProvisionersIntoContext(ctx context.Context, value map[string]pinfra.ClusterProvisioner) context.Context {
	return econtext.IntoContext(ctx, keyHostProvisioners, value)
}

func ProvisionersFromContext(ctx context.Context) (*map[string]pinfra.ClusterProvisioner, error) {
	return econtext.FromContext[map[string]pinfra.ClusterProvisioner](ctx, keyHostProvisioners)
}

func ProvisionersFromContextOrDie(ctx context.Context) map[string]pinfra.ClusterProvisioner {
	return econtext.FromContextOrDie[map[string]pinfra.ClusterProvisioner](ctx, keyHostProvisioners)
}

// kubes
func ClusterIntoContext(ctx context.Context, value pinfra.Cluster) context.Context {
	return econtext.IntoContext(ctx, keyHostCluster, value)
}

func ClusterFromContext(ctx context.Context) (*pinfra.Cluster, error) {
	return econtext.FromContext[pinfra.Cluster](ctx, keyHostCluster)
}

func ClusterFromContextOrDie(ctx context.Context) pinfra.Cluster {
	return econtext.FromContextOrDie[pinfra.Cluster](ctx, keyHostCluster)
}

// namespaces
func ScenarioNamespaceIntoContext(ctx context.Context, value string) context.Context {
	return econtext.IntoContext(ctx, keyHostScenarioNamespace, value)
}

func ScenarioNamespaceFromContext(ctx context.Context) (*string, error) {
	return econtext.FromContext[string](ctx, keyHostScenarioNamespace)
}

func ScenarioNamespaceFromContextOrDie(ctx context.Context) string {
	return econtext.FromContextOrDie[string](ctx, keyHostScenarioNamespace)
}
