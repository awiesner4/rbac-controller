package informers

import (
	"github.com/awiesner4/rbac-controller/internal/reconciler"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func roleBindingInformer(customFactory *CustomInformerFactory) cache.SharedIndexInformer {
	roleBindingInformer := customFactory.Factory.Rbac().V1().RoleBindings().Informer()

	// Add an event handler to watch for updates to RoleBindings
	roleBindingInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			rb, ok := obj.(*v1.RoleBinding)
			if !ok {
				logrus.Error("failed to type-assert object to RoleBinding")
				return
			}
			logrus.Infof("RoleBinding added in namespace %s: %s", rb.Namespace, rb.Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldRoleBinding := oldObj.(*v1.RoleBinding)
			newRoleBinding := newObj.(*v1.RoleBinding)

			// Check if the resource version has actually changed
			if oldRoleBinding.ResourceVersion != newRoleBinding.ResourceVersion {
				// logrus.Infof("RoleBinding update detected in namespace %s: %s", newRoleBinding.Namespace, newRoleBinding.Name)
				// // Log details when a genuine update occurs
				// logrus.Infof("Detected RoleBinding update. Old ResourceVersion: %v, New ResourceVersion: %v", oldRoleBinding.ResourceVersion, newRoleBinding.ResourceVersion)

				// If resource versions are the same, skip further processing
				rbr := reconciler.Reconciler{Clientset: customFactory.Clientset}

				// Create a custom ownerReferences slice using values from oldRoleBinding
				var ownerReferences []metav1.OwnerReference
				for _, ownerRef := range oldRoleBinding.OwnerReferences {
					ownerReferences = append(ownerReferences, metav1.OwnerReference{
						APIVersion: ownerRef.APIVersion,
						Kind:       ownerRef.Kind,
						Name:       ownerRef.Name,
						UID:        ownerRef.UID,
						// Add other fields if needed, e.g., BlockOwnerDeletion or Controller
						BlockOwnerDeletion: ownerRef.BlockOwnerDeletion,
						Controller:         ownerRef.Controller,
					})
				}

				if err := rbr.ReconcileOwners(ownerReferences, "RoleBinding", oldRoleBinding.RoleRef.Name); err != nil {
					logrus.Errorf("failed to reconcile RoleBindings: %v", err)
					return
				}

			}

		},
		DeleteFunc: func(obj interface{}) {
			roleBinding := obj.(*v1.RoleBinding)
			rbr := reconciler.Reconciler{Clientset: customFactory.Clientset}
			var ownerReferences []metav1.OwnerReference

			for _, ownerRef := range roleBinding.OwnerReferences {
				ownerReferences = append(ownerReferences, metav1.OwnerReference{
					APIVersion:         ownerRef.APIVersion,
					Kind:               ownerRef.Kind,
					Name:               ownerRef.Name,
					UID:                ownerRef.UID,
					BlockOwnerDeletion: ownerRef.BlockOwnerDeletion,
					Controller:         ownerRef.Controller,
				})
			}

			if err := rbr.ReconcileOwners(ownerReferences, "RoleBinding", roleBinding.RoleRef.Name); err != nil {
				logrus.Errorf("failed to reconcile RoleBindings: %v", err)
				return
			}
		},
	})
	return roleBindingInformer
}
