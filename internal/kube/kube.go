package kube

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LabelKey is the key of the key/value pair given to all resources managed by RBAC Manager
const LabelKey = "devopscentral.io/managed-by"

// LabelValue is the value of the key/value pair given to all resources managed by RBAC Manager
const LabelValue = "tenant-operator"

// Labels is the key/value pair given to all resources managed by RBAC Manager
// var Labels = map[string]string{LabelKey: LabelValue}

// ListOptions is the default set of options to find resources managed by RBAC Manager
var ListOptions = metav1.ListOptions{LabelSelector: LabelKey + "=" + LabelValue}

// GetLabelSelector returns a LabelSelector configured for RBAC Manager resources
func GetLabelSelector() *metav1.LabelSelector {
	return &metav1.LabelSelector{MatchLabels: map[string]string{LabelKey: LabelValue}}
}

// WithLabelSelector applies the label selector to ListOptions
func WithLabelSelector(options *metav1.ListOptions) {
	options.LabelSelector = LabelKey + "=" + LabelValue
}

func AddManagementLabels() map[string]string {
	return map[string]string{
		LabelKey: LabelValue,
	}
}
