package utils

import (
	"context"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/akuity/kargo/hack/test/e2e/envfuncs"
)

const NamespaceKey envfuncs.ContextKey = "namespace"

func SetupFixturesInNamespace(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		return SetupFixtures(context.WithValue(ctx, NamespaceKey, namespace), t, cfg)
	}
}

func SetupFixtures(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	testdata := os.DirFS("testdata")
	pattern := "*"
	namespace, ok := ctx.Value(NamespaceKey).(string)
	t.Logf("namespace %v\n", namespace)
	if !ok {
		t.Logf("Using config namespace \n")
		namespace = cfg.Namespace()
	}
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := decoder.DecodeEachFile(ctx, testdata, pattern,
		decoder.CreateHandler(r),           // try to CREATE objects after decoding
		decoder.MutateNamespace(namespace), // inject a namespace into decoded objects, before calling CreateHandler
	); err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TeardownFixturesInNamespace(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		return TeardownFixtures(context.WithValue(ctx, NamespaceKey, namespace), t, cfg)
	}
}

func TeardownFixtures(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	testdata := os.DirFS("testdata")
	pattern := "*"
	namespace, ok := ctx.Value(NamespaceKey).(string)
	t.Logf("namespace %v\n", namespace)
	if !ok {
		t.Logf("Using config namespace \n")
		namespace = cfg.Namespace()
	}
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := decoder.DecodeEachFile(ctx, testdata, pattern,
		decoder.DeleteHandler(r),           // try to DELETE objects after decoding
		decoder.MutateNamespace(namespace), // inject a namespace into decoded objects, before calling CreateHandler
	); err != nil {
		t.Fatal(err)
	}
	return ctx
}

func CreateNamespace(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		client := cfg.Client()
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		t.Logf("CREATE namespace %v\n", ns)
		if err := client.Resources().Create(ctx, ns); err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}

func DeleteNamespace(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		client := cfg.Client()
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		t.Logf("DELETE namespace %v\n", ns)
		if err := client.Resources().Delete(ctx, ns); err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}
