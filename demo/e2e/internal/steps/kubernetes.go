package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/cucumber/godog"
	"github.com/filariow/mctest/demo/e2e/internal/infra"
	"github.com/filariow/mctest/pkg/poll"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
}

func ResourcesExist(ctx context.Context, spec string) error {
	lctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return poll.Do(lctx, time.Second, func(cctx context.Context) error {
		k := infra.ClusterFromContextOrDie(ctx)
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
	lctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return func(ctx context.Context) error {
		k := infra.ClusterFromContextOrDie(ctx)
		uu, err := k.ParseResources(ctx, spec)
		if err != nil {
			return err
		}

		for _, u := range uu {
			ctxd, cf := context.WithTimeout(ctx, 10*time.Second)
			_, err = poll.DoR(ctxd, time.Second, func(ictx context.Context) (*unstructured.Unstructured, error) {
				lctx, lcf := context.WithTimeout(ictx, 1*time.Minute)
				defer lcf()

				t := types.NamespacedName{Namespace: u.GetNamespace(), Name: u.GetName()}
				lu := u.DeepCopy()
				if err := k.Get(lctx, t, lu, &client.GetOptions{}); err != nil {
					if kerrors.IsNotFound(err) {
						return nil, nil
					}
				}
				return nil, err
			})
			cf()

			if err != nil {
				ld, err := u.MarshalJSON()
				if err != nil {
					return fmt.Errorf(
						"resource exists: [ ApiVersion=%s, Kind=%s, Namespace=%s, Name=%s ]. Error marshaling as json: %w",
						u.GetAPIVersion(), u.GetKind(), u.GetNamespace(), u.GetName(), err)
				}
				return fmt.Errorf("resource exists: %s", ld)
			}
		}

		return nil
	}(lctx)
}

func ResourcesAreUpdated(ctx context.Context, spec string) error {
	lctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	k := infra.ClusterFromContextOrDie(lctx)
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		if err := k.Update(ctx, u.DeepCopy(), &client.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func ResourcesAreCreated(ctx context.Context, spec string) error {
	lctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return poll.Do(lctx, 5*time.Second, func(ctx context.Context) error {
		k := infra.ClusterFromContextOrDie(ctx)
		uu, err := k.ParseResources(ctx, spec)
		if err != nil {
			return err
		}

		for _, u := range uu {
			if err := k.Create(ctx, u.DeepCopy(), &client.CreateOptions{}); err != nil {
				return err
			}
		}
		return nil
	})
}
