package infra

import (
	"context"
	"errors"
	"os"
	"path"
	"time"

	"github.com/filariow/mctest/pkg/kube"
	"github.com/filariow/mctest/pkg/poll"
	"github.com/filariow/mctest/pkg/testrun"
)

type Cluster struct {
	*kube.Kubernetes
}

func NewCluster(k *kube.Kubernetes) *Cluster {
	return &Cluster{k}
}

// operator
func (c *Cluster) DeployOperatorInNamespace(ctx context.Context, opPath string, ns string) error {
	tf, err := testrun.TestFolderFromContext(ctx)
	if err != nil {
		return errors.Join(testrun.ErrTestFolderNotFound, err)
	}

	// read deployment manifests
	opd := path.Join(*tf, "config", "default", opPath)
	op, err := os.ReadFile(opd)
	if err != nil {
		return err
	}

	// Apply deployment resources
	if err := poll.Do(ctx, 10*time.Second, func(ctx context.Context) error {
		uu, err := c.Kubernetes.ParseResources(ctx, string(op))
		if err != nil {
			return err
		}

		for _, u := range uu {
			if u.GetKind() == "Namespace" {
				continue
			}
			u.SetNamespace(ns)
			if err := c.Kubernetes.CreateNamespacedResourceUnstructured(ctx, u); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
