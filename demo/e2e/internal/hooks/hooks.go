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

	// create auxiliary namespace in Management Cluster
	ctx.Before(prepareAuxiliaryNamespaceInManagementCluster)

	// inject provisioners
	ctx.Before(injectProvisioners)

	// prepare the test environment
	ctx.Before(prepareTestEnvironment)

	// set timeout for single test
	ctx.Before(setTimeout)
}

func injectHookCleanup(ctx *godog.ScenarioContext) {
	// cancel run context
	ctx.After(cancelRunContext)

	// unprovision clusters
	ctx.After(unprovisionClusters)

	// delete the ContextNamespace if no errors occurred
	ctx.After(destroyHostResources)

	// cleanup temp folder
	ctx.After(destroyScenarioTestFolder)
}
