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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	demomctestiov1alpha1 "github.com/filariow/mctest/demo/show/api/v1alpha1"
)

// ShowReconciler reconciles a Show object
type ShowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=demo.mctest.io,resources=shows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=demo.mctest.io,resources=shows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=demo.mctest.io,resources=shows/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ShowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	s := demomctestiov1alpha1.Show{}
	if err := r.Get(ctx, req.NamespacedName, &s); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// initialize state if required, and requeue
	if s.Status.State == "" {
		s.Status.State = demomctestiov1alpha1.ShowStateOpen
		if err := r.Status().Update(ctx, &s); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// fetch all exibitions for show
	lr, err := labels.NewRequirement("show", selection.Equals, []string{s.GetName()})
	if err != nil {
		panic(err)
	}
	ee := demomctestiov1alpha1.ExibitionList{}
	ls := labels.NewSelector().Add(*lr)
	if err := r.List(ctx, &ee, &client.ListOptions{LabelSelector: ls}); err != nil {
		return ctrl.Result{}, err
	}

	// check if show is already complete
	if len(ee.Items) >= s.Spec.Capacity {
		if s.Status.State != demomctestiov1alpha1.ShowStateComplete {
			s.Status.State = demomctestiov1alpha1.ShowStateComplete
			if err := r.Status().Update(ctx, &s); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// create new exibitions for the show
	ap := map[string]*demomctestiov1alpha1.Exibition{}
	for _, p := range ee.Items {
		ap[p.Spec.Performer] = &p
	}

	pp := demomctestiov1alpha1.PerformerList{}
	if err := r.List(ctx, &pp, &client.ListOptions{Namespace: s.GetNamespace()}); err != nil {
		return ctrl.Result{}, err
	}

	for _, p := range pp.Items {
		// check if show's capacity is reached
		if len(ap) == s.Spec.Capacity {
			if s.Status.State != demomctestiov1alpha1.ShowStateComplete {
				s.Status.State = demomctestiov1alpha1.ShowStateComplete
				if err := r.Status().Update(ctx, &s); err != nil {
					return ctrl.Result{}, err
				}
			}
			break
		}

		// skip performer if already performing at the show
		if _, ok := ap[p.GetName()]; ok {
			continue
		}

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
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, e, &client.CreateOptions{}); err != nil {
			return ctrl.Result{}, err
		}

		ap[p.GetName()] = e
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	handleExibitionRequests := func(o client.Object) []reconcile.Request {
		s, ok := o.GetLabels()["show"]
		if ok {
			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      s,
						Namespace: o.GetNamespace(),
					},
				},
			}
		}
		return nil
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&demomctestiov1alpha1.Show{}).
		Watches(
			&source.Kind{Type: &demomctestiov1alpha1.Exibition{}},
			handler.EnqueueRequestsFromMapFunc(handleExibitionRequests)).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
