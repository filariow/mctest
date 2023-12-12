package steps

import (
	"context"
	"time"

	"github.com/cucumber/godog"
	"github.com/filariow/mctest/demo/e2e/internal/infra"
)

func RegisterStepFuncsKubernetes(ctx *godog.ScenarioContext) {
	ctx.Step(`^Resource is created:$`, ResourcesAreCreated)
	ctx.Step(`^Resources are created:$`, ResourcesAreCreated)

	ctx.Step(`^Resource is updated:$`, ResourcesAreUpdated)
	ctx.Step(`^Resources are updated:$`, ResourcesAreUpdated)

	ctx.Step(`^Resource exists:$`, ResourcesExist)
	ctx.Step(`^Resources exist:$`, ResourcesExist)

	ctx.Step(`^Resource doesn't exist:$`, ResourcesNotExist)
	ctx.Step(`^Resources don't exist:$`, ResourcesNotExist)

	ctx.Step(`^Create namespace "([\w]+[\w-]*)"$`, CreateNamespace)
}

func ResourcesExist(ctx context.Context, spec string) error {
	lctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	hns := infra.ScenarioNamespaceFromContextOrDie(ctx)
	return infra.ClusterFromContextOrDie(ctx).
		ResourcesExistInNamespace(lctx, hns, spec)
}

func ResourcesNotExist(ctx context.Context, spec string) error {
	lctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	return infra.ClusterFromContextOrDie(lctx).
		ResourcesNotExist(lctx, spec)
}

func ResourcesAreUpdated(ctx context.Context, spec string) error {
	lctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	return infra.ClusterFromContextOrDie(lctx).
		ResourcesAreUpdated(lctx, spec)
}

func ResourcesAreCreated(ctx context.Context, spec string) error {
	lctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	hns := infra.ScenarioNamespaceFromContextOrDie(lctx)
	return infra.ClusterFromContextOrDie(lctx).
		ResourcesAreCreatedInNamespace(lctx, hns, spec)
}

func CreateNamespace(ctx context.Context, namespace string) error {
	lctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	return infra.ClusterFromContextOrDie(lctx).
		CreateNamespace(lctx, namespace)
}
