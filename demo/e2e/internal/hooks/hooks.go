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

	// inject ClusterAPI's management cluster request
	ctx.Before(injectManagementCluster)

	// set and create the ContextNamespace on ClusterAPI's Cluster
	ctx.Before(prepareScenarioNamespaces)

	// inject provisioners
	ctx.Before(injectProvisioners)

	// handle dedicated cluster request
	ctx.Before(injectDedicatedClusterIfRequired)

	// set timeout for single test
	ctx.Before(setTimeout)
}

func injectHookCleanup(ctx *godog.ScenarioContext) {
	// unprovision clusters
	ctx.After(unprovisionClusters)

	// delete the ContextNamespace if no errors occurred
	ctx.After(destroyHostResources)

	// cleanup temp folder
	ctx.After(hookDestroyScenarioTestFolder)

	// cancel run context
	ctx.After(cancelRunContext)
}
