package reconciler

import (
	"context"

	multitenancyv1alpha "github.com/awiesner4/rbac-controller/api/v1alpha"
	"github.com/awiesner4/rbac-controller/internal/kube"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// import (
// 	"sync"

// 	tenantcontrollerv1alpha "github.com/awiesner4/rbac-controller/api/v1alpha"
// 	"github.com/sirupsen/logrus"
// 	"k8s.io/client-go/kubernetes"
// )

// // import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type Reconciler struct {
	Clientset kubernetes.Interface
	ownerRefs []metav1.OwnerReference
}

func (r *Reconciler) Reconcile(tenant *multitenancyv1alpha.Tenant) error {
	mux.Lock()
	defer mux.Unlock()

	logrus.Infof("Reconciling Tenant %v", tenant.Name)

	r.ownerRefs = tenantOwnerRefs(tenant)

	p := ParsedTenant{
		Clientset: r.Clientset,
		ownerRefs: r.ownerRefs,
	}

	var err error

	err = p.ParseTenant(*tenant, "RoleBinding", "all")

	if err != nil {
		return err
	}
	// reconcile namepsaces
	for _, namespaceSpec := range tenant.Spec.Namespaces {
		_, err = r.ReconcileNamespace(namespaceSpec.Name)
		if err != nil {
			logrus.Info("Failed to reconcile namespace")
		}
	}

	for _, roleBinding := range p.roleBindings {
		err = r.reconcileRoleBinding(&roleBinding)
		if err != nil {
			return err
		}
	}

	return nil
}

// Entry point from informers
func (r *Reconciler) ReconcileOwners(ownerRefs []metav1.OwnerReference, kind string, clusterRole string) error {
	mux.Lock()
	defer mux.Unlock()

	// namespaces, err := r.Clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	logrus.Debug("Error listing namespaces")
	// 	return err
	// }

	for _, ownerRef := range ownerRefs {
		if ownerRef.Kind == "Tenant" {
			tenant, err := kube.GetTenantDefinition(ownerRef.Name)
			if err != nil {
				return err
			}

			r.ownerRefs = tenantOwnerRefs(&tenant)

			p := ParsedTenant{
				Clientset: r.Clientset,
				ownerRefs: r.ownerRefs,
			}

			if kind == "RoleBinding" {
				p.ParseTenant(tenant, kind, clusterRole)
				// return r.reconcileRoleBinding(&p.roleBindings, "reconcileOwners")
				for _, roleBinding := range p.roleBindings {
					err = r.reconcileRoleBinding(&roleBinding)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (r *Reconciler) ReconcileNamespace(namespaceName string) (reconcile.Result, error) {
	ns, err := r.Clientset.CoreV1().Namespaces().Get(context.TODO(), namespaceName, metav1.GetOptions{})

	if errors.IsNotFound(err) {
		logrus.Infof("Creating Namespace: %s", namespaceName)
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
				Labels: map[string]string{
					"devopscentral.io/managed-by": "tenant-operator",
				},
			},
		}
		if _, err := r.Clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{}); err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	} else {
		// If namespace exists and does not have the required label, add it
		if _, ok := ns.Labels["devopscentral.io/managed-by"]; !ok {
			logrus.Infof("Updating Namespace - Adding managed-by label: %s", namespaceName)
			ns.Labels["devopscentral.io/managed-by"] = "tenant-operator"
			if _, err = r.Clientset.CoreV1().Namespaces().Update(context.TODO(), ns, metav1.UpdateOptions{}); err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcileRoleBinding(requested *rbacv1.RoleBinding) error {
	// for _, requestedRB := range *requested {
	// logrus.Infof("Reconciling Role Binding: %v", requested.Name)
	// Attempt to get the RoleBinding by name and namespace
	existingRB, err := r.Clientset.RbacV1().RoleBindings(requested.Namespace).Get(context.TODO(), requested.Name, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			// RoleBinding doesn't exist, create it
			logrus.Infof("Creating Role Binding: %v", requested.Name)
			_, err := r.Clientset.RbacV1().RoleBindings(requested.Namespace).Create(context.TODO(), requested, metav1.CreateOptions{})
			if err != nil {
				logrus.Errorf("Error creating Role Binding: %v", err)
				return err
			}
		} else {
			// Some other error occurred
			logrus.Errorf("Error fetching Role Binding: %v", err)
			return err
		}
		// continue
	}

	// Check if the existing RoleBinding has the correct OwnerReference
	if !ownerRefsMatch(&existingRB.OwnerReferences, &requested.OwnerReferences) {
		logrus.Warnf("Existing Role Binding %v does not have the correct OwnerReference; skipping update", existingRB.Name)
		// continue
	}

	// Compare existing and requested RoleBinding state
	if !rbMatches(existingRB, requested) {
		logrus.Infof("Changes detected. Updating Role Binding: %v", existingRB.Name)
		requested.ObjectMeta.ResourceVersion = existingRB.ObjectMeta.ResourceVersion // Preserve resource version for update
		_, err := r.Clientset.RbacV1().RoleBindings(existingRB.Namespace).Update(context.TODO(), requested, metav1.UpdateOptions{})
		if err != nil {
			logrus.Errorf("Error updating Role Binding: %v", err)
			return err
		}
	} else {
		logrus.Debugf("Role Binding already matches desired state: %v", existingRB.Name)
	}
	// }

	// Cleanup orphaned RoleBindings
	// existing, err := r.Clientset.RbacV1().RoleBindings("").List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	return err
	// }

	// for _, existingRB := range existing.Items {
	// 	// Check if the RoleBinding is owned by this controller
	// 	if reflect.DeepEqual(existingRB.OwnerReferences, r.ownerRefs) {
	// 		// Determine if this RoleBinding is part of the requested state
	// 		isOrphaned := true
	// 		for _, requestedRB := range *requested {
	// 			if existingRB.Name == requestedRB.Name && existingRB.Namespace == requestedRB.Namespace {
	// 				isOrphaned = false
	// 				break
	// 			}
	// 		}

	// 		if isOrphaned {
	// 			logrus.Infof("Deleting orphaned Role Binding: %v", existingRB.Name)
	// 			err := r.Clientset.RbacV1().RoleBindings(existingRB.Namespace).Delete(context.TODO(), existingRB.Name, metav1.DeleteOptions{})
	// 			if err != nil {
	// 				logrus.Errorf("Error deleting orphaned Role Binding: %v", err)
	// 			}
	// 		}
	// 	}
	// }

	return nil
}

func tenantOwnerRefs(tenant *multitenancyv1alpha.Tenant) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(tenant, schema.GroupVersionKind{
			Group:   multitenancyv1alpha.GroupVersion.Group,
			Version: multitenancyv1alpha.GroupVersion.Version,
			Kind:    "Tenant",
		}),
	}
}
