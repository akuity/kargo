package api

import (
	"embed"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/akuity/kargo/internal/kubeclient"
)

//go:embed testdata/*
var testData embed.FS

func mustNewScheme() *runtime.Scheme {
	scheme, err := kubeclient.NewAPIScheme()
	if err != nil {
		panic(errors.Wrap(err, "new api scheme"))
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
