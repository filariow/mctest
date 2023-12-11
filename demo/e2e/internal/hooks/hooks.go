package hooks

import (
	"github.com/cucumber/godog"
)

func InjectHooks(ctx *godog.ScenarioContext) {
	injectHookSetup(ctx)
	injectHookCleanup(ctx)
}

func injectHookSetup(ctx *godog.ScenarioContext) {
	// create temp folder for scenario
	ctx.Before(hookPrepareScenarioTestFolder)

	// inject client for management cluster
	ctx.Before(buildInjectManagementClusterKube)

	// set and create the ContextNamespace on Management Cluster
	ctx.Before(prepareScenarioNamespaceInManagementCluster)

	// handle dedicated host request
	ctx.Before(injectHostCluster)

	// set and create the ContextNamespace on Host Cluster
	ctx.Before(prepareScenarioNamespaceInHost)

	// inject provisioners
	ctx.Before(injectProvisioners)

	// set timeout for single test
	ctx.Before(setTimeout)
}

func injectHookCleanup(ctx *godog.ScenarioContext) {
	// unprovision member clusters
	ctx.After(unprovisionMemberClusters)

	// delete the ContextNamespace if no errors occurred
	ctx.After(destroyHostResources)

	// cleanup temp folder
	ctx.After(hookDestroyScenarioTestFolder)

	// cancel run context
	ctx.After(cancelRunContext)
}
