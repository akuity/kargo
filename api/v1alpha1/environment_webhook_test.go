package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefault(t *testing.T) {
	const testNamespace = "fake-namespace"
	e := Environment{
		ObjectMeta: v1.ObjectMeta{
			Name:      "fake-stage-env",
			Namespace: testNamespace,
		},
	}
	e.Spec.Subscriptions.UpstreamEnvs = []EnvironmentSubscription{
		{
			Name: "fake-test-env",
		},
	}
	e.Spec.PromotionMechanisms.ArgoCDAppUpdates = []ArgoCDAppUpdate{
		{
			AppName: "fake-prod-app",
		},
	}
	e.Spec.HealthChecks.ArgoCDAppChecks = []ArgoCDAppCheck{
		{
			AppName: "fake-prod-app",
		},
	}
	e.Default()
	require.Len(t, e.Spec.Subscriptions.UpstreamEnvs, 1)
	require.Equal(
		t,
		testNamespace,
		e.Spec.Subscriptions.UpstreamEnvs[0].Namespace,
	)
	require.Len(t, e.Spec.PromotionMechanisms.ArgoCDAppUpdates, 1)
	require.Equal(
		t,
		testNamespace,
		e.Spec.PromotionMechanisms.ArgoCDAppUpdates[0].AppNamespace,
	)
	require.Len(t, e.Spec.HealthChecks.ArgoCDAppChecks, 1)
	require.Equal(
		t,
		testNamespace,
		e.Spec.HealthChecks.ArgoCDAppChecks[0].AppNamespace)

}
