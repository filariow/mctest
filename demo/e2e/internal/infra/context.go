package infra

import (
	"context"

	econtext "github.com/filariow/mctest/pkg/context"
	pinfra "github.com/filariow/mctest/pkg/infra"
)

const (
	// provisioners
	keyProvisioners string = "provisioners"
	// clusters
	keyCluster string = "cluster"
	// namespaces
	keyScenarioNamespace          string = "scenario-namespace"
	keyAuxiliaryScenarioNamespace string = "auxiliary-scenario-namespace"
)

// provisioners
func ProvisionersIntoContext(ctx context.Context, value map[string]pinfra.ClusterProvisioner) context.Context {
	return econtext.IntoContext(ctx, keyProvisioners, value)
}

func ProvisionersFromContext(ctx context.Context) (*map[string]pinfra.ClusterProvisioner, error) {
	return econtext.FromContext[map[string]pinfra.ClusterProvisioner](ctx, keyProvisioners)
}

func ProvisionersFromContextOrDie(ctx context.Context) map[string]pinfra.ClusterProvisioner {
	return econtext.FromContextOrDie[map[string]pinfra.ClusterProvisioner](ctx, keyProvisioners)
}

// kubes
func ClusterIntoContext(ctx context.Context, value pinfra.Cluster) context.Context {
	return econtext.IntoContext(ctx, keyCluster, value)
}

func ClusterFromContext(ctx context.Context) (*pinfra.Cluster, error) {
	return econtext.FromContext[pinfra.Cluster](ctx, keyCluster)
}

func ClusterFromContextOrDie(ctx context.Context) pinfra.Cluster {
	return econtext.FromContextOrDie[pinfra.Cluster](ctx, keyCluster)
}

// namespaces
func ScenarioNamespaceIntoContext(ctx context.Context, value string) context.Context {
	return econtext.IntoContext(ctx, keyScenarioNamespace, value)
}

func ScenarioNamespaceFromContext(ctx context.Context) (*string, error) {
	return econtext.FromContext[string](ctx, keyScenarioNamespace)
}

func ScenarioNamespaceFromContextOrDie(ctx context.Context) string {
	return econtext.FromContextOrDie[string](ctx, keyScenarioNamespace)
}

func AuxiliaryScenarioNamespaceIntoContext(ctx context.Context, value string) context.Context {
	return econtext.IntoContext(ctx, keyAuxiliaryScenarioNamespace, value)
}

func AuxiliaryScenarioNamespaceFromContext(ctx context.Context) (*string, error) {
	return econtext.FromContext[string](ctx, keyAuxiliaryScenarioNamespace)
}

func AuxiliaryScenarioNamespaceFromContextOrDie(ctx context.Context) string {
	return econtext.FromContextOrDie[string](ctx, keyAuxiliaryScenarioNamespace)
}
