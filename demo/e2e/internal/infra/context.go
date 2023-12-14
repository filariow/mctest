package infra

import (
	"context"

	econtext "github.com/filariow/mctest/pkg/context"
	pinfra "github.com/filariow/mctest/pkg/infra"
	"github.com/filariow/mctest/pkg/kube"
)

const (
	// provisioners
	keyProvisioners string = "provisioners"
	// clusters
	keyCluster           string = "cluster"
	keyManagementCluster string = "management-cluster"
	// namespaces
	keyScenarioNamespace          string = "scenario-namespace"
	keyAuxiliaryScenarioNamespace string = "auxiliary-scenario-namespace"
)

// provisioners
func ProvisionersIntoContext(ctx context.Context, value map[string]pinfra.ClusterProvisioner) context.Context {
	return econtext.IntoContext(ctx, keyProvisioners, value)
}

func ProvisionersFromContext(ctx context.Context) (map[string]pinfra.ClusterProvisioner, error) {
	return econtext.FromContext[map[string]pinfra.ClusterProvisioner](ctx, keyProvisioners)
}

func ProvisionersFromContextOrDie(ctx context.Context) map[string]pinfra.ClusterProvisioner {
	return econtext.FromContextOrDie[map[string]pinfra.ClusterProvisioner](ctx, keyProvisioners)
}

// management cluster
func ManagementClusterIntoContext(ctx context.Context, value kube.Client) context.Context {
	return econtext.IntoContext(ctx, keyManagementCluster, value)
}

func ManagementClusterFromContext(ctx context.Context) (kube.Client, error) {
	return econtext.FromContext[kube.Client](ctx, keyManagementCluster)
}

func ManagementClusterFromContextOrDie(ctx context.Context) kube.Client {
	return econtext.FromContextOrDie[kube.Client](ctx, keyManagementCluster)
}

func AuxiliaryScenarioNamespaceIntoContext(ctx context.Context, value string) context.Context {
	return econtext.IntoContext(ctx, keyAuxiliaryScenarioNamespace, value)
}

func AuxiliaryScenarioNamespaceFromContext(ctx context.Context) (string, error) {
	return econtext.FromContext[string](ctx, keyAuxiliaryScenarioNamespace)
}

func AuxiliaryScenarioNamespaceFromContextOrDie(ctx context.Context) string {
	return econtext.FromContextOrDie[string](ctx, keyAuxiliaryScenarioNamespace)
}

// scenario cluster
func ScenarioClusterIntoContext(ctx context.Context, value kube.Client) context.Context {
	return econtext.IntoContext(ctx, keyCluster, value)
}

func ScenarioClusterFromContext(ctx context.Context) (kube.Client, error) {
	return econtext.FromContext[kube.Client](ctx, keyCluster)
}

func ScenarioClusterFromContextOrDie(ctx context.Context) kube.Client {
	return econtext.FromContextOrDie[kube.Client](ctx, keyCluster)
}

func ScenarioNamespaceIntoContext(ctx context.Context, value string) context.Context {
	return econtext.IntoContext(ctx, keyScenarioNamespace, value)
}

func ScenarioNamespaceFromContext(ctx context.Context) (string, error) {
	return econtext.FromContext[string](ctx, keyScenarioNamespace)
}

func ScenarioNamespaceFromContextOrDie(ctx context.Context) string {
	return econtext.FromContextOrDie[string](ctx, keyScenarioNamespace)
}
