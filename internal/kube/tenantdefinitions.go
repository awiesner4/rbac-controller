/*
Copyright 2018 FairwindsOps Inc.

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

package kube

import (
	"context"

	multitenancyv1alpha1 "github.com/awiesner4/rbac-controller/api/v1alpha"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// GetRbacDefinition returns an RbacDefinition for a specified name or an error
func GetTenantDefinition(name string) (multitenancyv1alpha1.Tenant, error) {
	tenant := multitenancyv1alpha1.Tenant{}

	client, err := getTenantClient()
	if err != nil {
		return tenant, err
	}

	err = client.Get().Resource("tenants").Name(name).Do(context.TODO()).Into(&tenant)

	return tenant, err
}

// GetRbacDefinitions returns an RbacDefinitionList or an error
func GetRbacDefinitions() (multitenancyv1alpha1.TenantList, error) {
	list := multitenancyv1alpha1.TenantList{}

	client, err := getTenantClient()
	if err != nil {
		return list, err
	}

	err = client.Get().Resource("tenants").Do(context.TODO()).Into(&list)

	return list, err
}

func getTenantClient() (*rest.RESTClient, error) {
	_ = multitenancyv1alpha1.AddToScheme(scheme.Scheme)
	clientConfig := config.GetConfigOrDie()
	clientConfig.ContentConfig.GroupVersion = &multitenancyv1alpha1.GroupVersion
	clientConfig.APIPath = "/apis"
	clientConfig.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	clientConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	return rest.UnversionedRESTClientFor(clientConfig)
}
