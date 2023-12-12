package hooks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/cucumber/godog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/filariow/mctest/demo/e2e/internal/infra"
	"github.com/filariow/mctest/pkg/testrun"
)

func cancelRunContext(ctx context.Context, _ *godog.Scenario, err error) (context.Context, error) {
	cancel, err := testrun.TimeoutContextCancelFromContext(ctx)
	if err != nil {
		return ctx, err
	}
	(*cancel)()

	return ctx, err
}

func unprovisionClusters(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	// if an error occurred before, do not cleanup
	if err != nil {
		return ctx, err
	}

	// fetch all provisioners and unprovision
	pp, err := infra.ProvisionersFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	errs := []error{}
	for _, p := range *pp {
		if err := p.Unprovision(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if err := errors.Join(errs...); err != nil {
		log.Printf("error unprovisioning clusters: %v", err)
		return ctx, err
	}
	return ctx, nil
}

func destroyHostResources(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	// if an error occurred before, do not cleanup
	if err != nil {
		return ctx, err
	}

	// fetch host cluster from context
	kh, err := infra.ClusterFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	// delete test namespace from host
	r, err := labels.NewRequirement("scenario", selection.Equals, []string{sc.Id})
	if err != nil {
		return ctx, err
	}
	s := labels.NewSelector().Add(*r)
	deleteOpts := &client.DeleteAllOfOptions{ListOptions: client.ListOptions{LabelSelector: s}}
	if errDel := kh.Kubernetes.CRCli.DeleteAllOf(ctx, &corev1.Namespace{}, deleteOpts); errDel != nil {
		cerr := errors.Join(err, errDel)
		log.Printf("error destroying host resources: %s", cerr)
		return ctx, cerr
	}

	return ctx, nil
}

func hookDestroyScenarioTestFolder(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	// if test failed or any other error happened,
	// do not delete the test folder to allow inspection
	if err != nil {
		return ctx, err
	}

	tf, err := testrun.TestFolderFromContext(ctx)
	if err != nil {
		// if that's happen, it is to be considered a bug as test folder
		// should be injected before starting the scenario in its own hook
		// (cf. buildHookPrepareScenarioNamespace)
		return ctx, errors.Join(err, testrun.ErrTestFolderNotFound)
	}

	// delete test folder
	if err := os.RemoveAll(*tf); err != nil {
		return ctx, fmt.Errorf("error cleaning up temp folder for test %s: %w", sc.Id, err)
	}

	return ctx, err
}
