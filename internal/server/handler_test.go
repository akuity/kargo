package server

import (
	"embed"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

//go:embed testdata/*
var testData embed.FS

func mustNewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(fmt.Errorf("add core v1 to scheme: %w", err))
	}
	if err := kargoapi.AddToScheme(scheme); err != nil {
		panic(fmt.Errorf("add kargo v1alpha1 scheme: %w", err))
	}
	return scheme
}

func mustNewObject[T any](path string) *T {
	rawObj, err := testData.ReadFile(path)
	if err != nil {
		panic(fmt.Errorf("read file from path %q: %w", path, err))
	}

	var obj T
	if err := yaml.Unmarshal(rawObj, &obj); err != nil {
		panic(fmt.Errorf("unmarshal yaml: %w", err))
	}
	return &obj
}
