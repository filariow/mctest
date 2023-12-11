/*
Copyright 2023 The MCTest Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	demomctestiov1alpha1 "github.com/filariow/mctest/demo/show/api/v1alpha1"
)

// PerformerReconciler reconciles a Performer object
type PerformerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=demo.mctest.io,resources=performers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=demo.mctest.io,resources=performers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=demo.mctest.io,resources=performers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PerformerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	p := demomctestiov1alpha1.Performer{}
	if err := r.Get(ctx, req.NamespacedName, &p); err != nil {
		if errors.IsNotFound(err) {
			lr, err := labels.NewRequirement("performer", selection.Equals, []string{p.GetName()})
			if err != nil {
				panic(err)
			}
			ls := labels.NewSelector().Add(*lr)
			opts := &client.DeleteAllOfOptions{ListOptions: client.ListOptions{LabelSelector: ls}}
			if err := r.DeleteAllOf(ctx, &demomctestiov1alpha1.Exibition{}, opts); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// enroll performer to all open shows
	ss := demomctestiov1alpha1.ShowList{}
	if err := r.List(ctx, &ss, &client.ListOptions{}); err != nil {
		return ctrl.Result{}, err
	}
	for _, s := range ss.Items {
		if err := r.ensurePerformerIsEnrolledToShow(ctx, &p, s); err != nil {
			return ctrl.Result{}, err
		}
	}

	// update observed generation
	p.Status.ObservedGeneration = p.ObjectMeta.Generation
	if err := r.Status().Update(ctx, &p); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PerformerReconciler) ensurePerformerIsEnrolledToShow(
	ctx context.Context,
	p *demomctestiov1alpha1.Performer,
	s demomctestiov1alpha1.Show,
) error {
	if s.Status.State == demomctestiov1alpha1.ShowStateComplete {
		return nil
	}

	lrp, err := labels.NewRequirement("performer", selection.Equals, []string{p.GetName()})
	if err != nil {
		panic(err)
	}
	lrs, err := labels.NewRequirement("show", selection.Equals, []string{s.GetName()})
	if err != nil {
		panic(err)
	}
	ls := labels.NewSelector().Add(*lrp).Add(*lrs)

	ee := demomctestiov1alpha1.ExibitionList{}
	if err := r.List(ctx, &ee, &client.ListOptions{LabelSelector: ls}); err != nil {
		return err
	}

	if i := len(ee.Items); i > 1 {
		return fmt.Errorf(
			"expected just one exibition for performer %s and space %s, found %d",
			p.GetName(), s.GetName(), i)
	}

	if len(ee.Items) == 0 {
		// if the performer is not already performing at show, create an exibition
		e := &demomctestiov1alpha1.Exibition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", s.GetName(), p.GetName()),
				Namespace: s.GetNamespace(),
				Labels: map[string]string{
					"show":      s.GetName(),
					"performer": p.GetName(),
				},
			},
			Spec: demomctestiov1alpha1.ExibitionSpec{
				Performer: p.GetName(),
				Show:      s.GetName(),
			},
		}
		if err := controllerutil.SetControllerReference(&s, e, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, e, &client.CreateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PerformerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demomctestiov1alpha1.Performer{}).
		Complete(r)
}
