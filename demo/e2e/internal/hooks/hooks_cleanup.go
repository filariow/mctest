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

	// fetch management cluster from context
	kh, err := infra.ManagementClusterFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	// build label selector for current scenario
	r, err := labels.NewRequirement("scenario", selection.Equals, []string{sc.Id})
	if err != nil {
		return ctx, err
	}
	s := labels.NewSelector().Add(*r)

	// list all namespaces related to current scenario
	nn := corev1.NamespaceList{}
	if errList := kh.CRCli.List(ctx, &nn, &client.ListOptions{LabelSelector: s}); errList != nil {
		cerr := errors.Join(err, errList)
		log.Printf("error listing namespaces before destroying: %s", cerr)
		return ctx, cerr

	}

	// delete all namespaces related to current scenario
	for _, n := range nn.Items {
		if errDel := kh.CRCli.Delete(ctx, &n, &client.DeleteOptions{}); errDel != nil {
			cerr := errors.Join(err, errDel)
			log.Printf("error destroying namespace %s in management cluster: %s", n.Name, cerr)
			return ctx, cerr
		}
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
