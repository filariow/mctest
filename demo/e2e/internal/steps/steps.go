package steps

import "github.com/cucumber/godog"

func InjectSteps(ctx *godog.ScenarioContext) {
	RegisterStepFuncsKubernetes(ctx)
}
