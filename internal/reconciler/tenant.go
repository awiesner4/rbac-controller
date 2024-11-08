package reconciler

import (
	"sync"

	multitenancyv1alpha "github.com/awiesner4/rbac-controller/api/v1alpha"
	"github.com/awiesner4/rbac-controller/internal/kube"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ParsedTenant struct {
	Clientset    kubernetes.Interface
	ownerRefs    []metav1.OwnerReference
	roleBindings []rbacv1.RoleBinding
}

func (p *ParsedTenant) ParseTenant(tenantDef multitenancyv1alpha.Tenant, kind string, clusterRole string) error {
	// Parse the Tenant definition into p.RoleBindings

	if kind == "RoleBinding" {
		for _, namespaceSpec := range tenantDef.Spec.Namespaces {
			for _, roleSpec := range namespaceSpec.Roles {
				if clusterRole == "all" {
					roleBinding := p.BuildRoleBinding(namespaceSpec.Name, roleSpec.Name, tenantDef.Name)
					p.roleBindings = append(p.roleBindings, roleBinding)
				} else {
					roleBinding := p.BuildRoleBinding(namespaceSpec.Name, clusterRole, tenantDef.Name)
					p.roleBindings = append(p.roleBindings, roleBinding)
					break
				}
			}
		}
	}
	return nil
}

var mux = sync.Mutex{}

// ReconcileOwners reconciles any RBACDefinitions found in owner references

// function to just return a rbacv1.RoleBinding standard that only takes in the namespace, clusterole, and serviceaccount name
func (r *ParsedTenant) BuildRoleBinding(namespace string, clusterRole string, serviceAccount string) rbacv1.RoleBinding {
	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceAccount + "-" + clusterRole,
			Namespace:       namespace,
			OwnerReferences: r.ownerRefs,
			Labels:          kube.AddManagementLabels(),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: namespace,
			},
		},
	}

	return roleBinding
}
