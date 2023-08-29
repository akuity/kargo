package api

import (
	"embed"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
)

//go:embed testdata/*
var testData embed.FS

func mustNewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(errors.Wrap(err, "add core v1 scheme"))
	}
	if err := kubev1alpha1.AddToScheme(scheme); err != nil {
		panic(errors.Wrap(err, "add kargo v1alpha1 scheme"))
	}
	return scheme
}

func mustNewObject[T any](path string) *T {
	rawObj, err := testData.ReadFile(path)
	if err != nil {
		panic(errors.Wrapf(err, "read file from path %q", path))
	}

	var obj T
	if err := yaml.Unmarshal(rawObj, &obj); err != nil {
		panic(errors.Wrap(err, "unmarshal yaml"))
	}
	return &obj
}
