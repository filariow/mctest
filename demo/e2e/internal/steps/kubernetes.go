package steps

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/cucumber/godog"
	"github.com/filariow/mctest/demo/e2e/internal/infra"
	"github.com/filariow/mctest/pkg/poll"
	"github.com/filariow/mctest/pkg/testrun"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterStepFuncsKubernetes(ctx *godog.ScenarioContext) {
	ctx.Step(`^Resource is created:$`, ResourcesAreCreated)
	ctx.Step(`^Resources are created:$`, ResourcesAreCreated)

	ctx.Step(`^Resource can not be created:$`, ResourcesCanNotBeCreated)
	ctx.Step(`^Resources can not be created:$`, ResourcesCanNotBeCreated)

	ctx.Step(`^Resource is updated:$`, ResourcesAreUpdated)
	ctx.Step(`^Resources are updated:$`, ResourcesAreUpdated)

	ctx.Step(`^Resource exists:$`, ResourcesExist)
	ctx.Step(`^Resource exists in scenario namespace:$`, ResourcesExist)
	ctx.Step(`^Resources exist:$`, ResourcesExist)

	ctx.Step(`^Resource doesn't exist:$`, ResourcesNotExist)
	ctx.Step(`^Resources don't exist:$`, ResourcesNotExist)

	ctx.Step(`^Operator "([\w]+[\w-]*)" is installed$`, DeployOperator)
	ctx.Step(`^Operator "([\w]+[\w-]*)" is installed in namespace "([\w]+[\w-]*)"$`, DeployOperatorInNamespace)
}

func ResourcesExist(ctx context.Context, spec string) error {
	return poll.DoWithTimeout(ctx, time.Second, 10*time.Second, func(ctx context.Context) error {
		k := infra.ScenarioClusterFromContextOrDie(ctx)
		uu, err := k.ParseResources(ctx, spec)
		if err != nil {
			return err
		}

		for _, u := range uu {
			lu := u.DeepCopy()

			t := types.NamespacedName{Namespace: lu.GetNamespace(), Name: lu.GetName()}
			if err := k.Get(ctx, t, lu, &client.GetOptions{}); err != nil {
				return err
			}
		}
		return nil
	})
}

func ResourcesNotExist(ctx context.Context, spec string) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	k := infra.ScenarioClusterFromContextOrDie(ctx)
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	// TODO: use concurrency here
	for _, u := range uu {
		lu, err := poll.DoRWithTimeout(ctx, time.Second, 20*time.Second, func(ctx context.Context) (*unstructured.Unstructured, error) {
			t := types.NamespacedName{Namespace: u.GetNamespace(), Name: u.GetName()}
			lu := u.DeepCopy()
			if err := k.Get(ctx, t, lu, &client.GetOptions{}); err != nil {
				if kerrors.IsNotFound(err) {
					return nil, nil
				}
			}
			return nil, err
		})

		if err != nil {
			ld, err := lu.MarshalJSON()
			if err != nil {
				return fmt.Errorf(
					"expected resource not to exists. Found: [ ApiVersion=%s, Kind=%s, Namespace=%s, Name=%s ]. Error marshaling as json: %w",
					lu.GetAPIVersion(), lu.GetKind(), lu.GetNamespace(), lu.GetName(), err)
			}
			return fmt.Errorf("expected resource not to exists. Found: %s", ld)
		}
	}

	return nil
}

func ResourcesAreUpdated(ctx context.Context, spec string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	k := infra.ScenarioClusterFromContextOrDie(ctx)
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	// TODO: implement concurrency
	for _, u := range uu {
		// retry ~5 times
		if err := poll.DoWithTimeout(ctx, 2*time.Second, 10*time.Second, func(ctx context.Context) error {
			if err := k.Update(ctx, u.DeepCopy(), &client.UpdateOptions{}); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func ResourcesAreCreated(ctx context.Context, spec string) error {
	return resourcesAreCreated(ctx, spec, nil)
}

func resourcesAreCreated(ctx context.Context, spec string, namespace *string) error {
	k := infra.ScenarioClusterFromContextOrDie(ctx)
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		lu := u.DeepCopy()
		if namespace != nil {
			lu.SetNamespace(*namespace)
		}

		if err := poll.DoWithTimeout(ctx, 5*time.Second, 1*time.Minute, func(ctx context.Context) error {
			return k.Create(ctx, lu, &client.CreateOptions{})
		}); err != nil {
			return err
		}
	}
	return nil
}

func ResourcesCanNotBeCreated(ctx context.Context, spec string) error {
	k := infra.ScenarioClusterFromContextOrDie(ctx)
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		lu := u.DeepCopy()
		if err := k.Create(ctx, lu, &client.CreateOptions{}); err == nil {
			ld, err := lu.MarshalJSON()
			if err != nil {
				return fmt.Errorf(
					"expected resource not to be created. Created: [ ApiVersion=%s, Kind=%s, Namespace=%s, Name=%s ]. Error marshaling as json: %w",
					lu.GetAPIVersion(), lu.GetKind(), lu.GetNamespace(), lu.GetName(), err)
			}
			return fmt.Errorf("expected resource not to be created. Created: %s", ld)
		}
	}
	return nil
}

func DeployOperator(ctx context.Context, operator string) error {
	tf, err := testrun.TestFolderFromContext(ctx)
	if err != nil {
		return errors.Join(testrun.ErrTestFolderNotFound, err)
	}

	// read deployment manifests
	applyFile := func(fname string) error {
		opd := path.Join(tf, "config", "default", fname)
		op, err := os.ReadFile(opd)
		if err != nil {
			return err
		}

		// Apply deployment resources
		return ResourcesAreCreated(ctx, string(op))
	}

	if err := applyFile(fmt.Sprintf("%s-rbac.yaml", operator)); err != nil {
		return err
	}
	return applyFile(fmt.Sprintf("%s.yaml", operator))
}

func DeployOperatorInNamespace(ctx context.Context, operator, namespace string) error {
	tf, err := testrun.TestFolderFromContext(ctx)
	if err != nil {
		return errors.Join(testrun.ErrTestFolderNotFound, err)
	}

	// read deployment manifests
	applyFile := func(fname string) error {
		opd := path.Join(tf, "config", "default", fname)
		op, err := os.ReadFile(opd)
		if err != nil {
			return err
		}

		// Apply deployment resources
		if err := resourcesAreCreated(ctx, string(op), &namespace); err != nil {
			return fmt.Errorf("%w: creating resource in namespace %s: %s ", err, namespace, string(op))
		}
		return nil
	}

	if err := applyFile(fmt.Sprintf("%s-rbac.yaml", operator)); err != nil {
		return err
	}
	return applyFile(fmt.Sprintf("%s.yaml", operator))
}
