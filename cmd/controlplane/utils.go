package main

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func argoCDExists(
	ctx context.Context,
	restCfg *rest.Config,
	namespace string,
) bool {
	if client, err := dynamic.NewForConfig(restCfg); err == nil {
		if _, err = client.Resource(
			schema.GroupVersionResource{
				Group:    "argoproj.io",
				Version:  "v1alpha1",
				Resource: "applications",
			},
		).Namespace(namespace).List(ctx, metav1.ListOptions{Limit: 1}); err == nil {
			return true
		}
	}
	return false
}

func argoRolloutsExists(ctx context.Context, restCfg *rest.Config) bool {
	if client, err := dynamic.NewForConfig(restCfg); err == nil {
		if _, err = client.Resource(
			schema.GroupVersionResource{
				Group:    "argoproj.io",
				Version:  "v1alpha1",
				Resource: "analysistemplates",
			},
		).List(ctx, metav1.ListOptions{Limit: 1}); err == nil {
			return true
		}
	}
	return false
}
