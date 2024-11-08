/*
Copyright 2024.

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

package controller

import (
	"context"

	multitenancyv1alpha1 "github.com/awiesner4/rbac-controller/api/v1alpha"
	"github.com/awiesner4/rbac-controller/internal/reconciler"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type TenantReconciler struct {
	client.Client
	Clientset kubernetes.Interface
	ownerRefs []metav1.OwnerReference
	Scheme    *runtime.Scheme
	// RoleBindingInformer cache.SharedInformer
}

func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// log := r.Log.WithValues("tenant", req.NamespacedName)

	logrus.Info("Reconciling Tenant")
	tenant := &multitenancyv1alpha1.Tenant{}
	tr := reconciler.Reconciler{Clientset: r.Clientset}

	if err := r.Client.Get(ctx, req.NamespacedName, tenant); err != nil {
		if errors.IsNotFound(err) {
			logrus.Info("Tenant resource not found. Ignoring since object must be deleted.")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	//fetch tenant definition
	err := r.Get(ctx, req.NamespacedName, tenant)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = tr.Reconcile(tenant)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil

}

func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multitenancyv1alpha1.Tenant{}).
		Complete(r)
}
