package targets

import (
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	libhttp "github.com/akuity/kargo/pkg/http"
)

func TestValidateTarget(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		target *kargoapi.Target
		fields []string
	}{
		{
			name:   "valid",
			target: &kargoapi.Target{ObjectMeta: metav1.ObjectMeta{Name: "us-east-1"}},
		},
		{
			name: "valid with type meta and labels",
			target: &kargoapi.Target{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kargoapi.GroupVersion.String(),
					Kind:       "Target",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   "us-east-1",
					Labels: map[string]string{"region": "us", "kargo.akuity.io/tier": "prod"},
				},
			},
		},
		{
			name:   "name is required",
			target: &kargoapi.Target{},
			fields: []string{"metadata.name"},
		},
		{
			name:   "name must be a DNS subdomain",
			target: &kargoapi.Target{ObjectMeta: metav1.ObjectMeta{Name: "Not_Valid"}},
			fields: []string{"metadata.name"},
		},
		{
			name: "wrong type meta",
			target: &kargoapi.Target{
				TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{Name: "us-east-1"},
			},
			fields: []string{"apiVersion", "kind"},
		},
		{
			name: "invalid label",
			target: &kargoapi.Target{ObjectMeta: metav1.ObjectMeta{
				Name:   "us-east-1",
				Labels: map[string]string{"bad key": "x"},
			}},
			fields: []string{"metadata.labels"},
		},
		{
			name: "metadata Kubernetes would have honored is refused",
			target: &kargoapi.Target{ObjectMeta: metav1.ObjectMeta{
				Name:            "us-east-1",
				Annotations:     map[string]string{"a": "b"},
				Finalizers:      []string{"x"},
				OwnerReferences: []metav1.OwnerReference{{Name: "owner"}},
			}},
			fields: []string{"metadata.annotations", "metadata.finalizers", "metadata.ownerReferences"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			errs := validateTarget(testCase.target)
			var fields []string
			for _, err := range errs {
				fields = append(fields, err.Field)
			}
			slices.Sort(fields)
			require.Equal(t, testCase.fields, slices.Compact(fields))
		})
	}
}

func TestStoreError(t *testing.T) {
	t.Parallel()
	status := func(err error) int {
		var httpErr *libhttp.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr.Code()
		}
		return 0
	}
	testCases := []struct {
		name   string
		err    error
		status int
	}{
		{name: "not found", err: database.ErrNotFound, status: http.StatusNotFound},
		{name: "already exists", err: database.ErrAlreadyExists, status: http.StatusConflict},
		{name: "conflict", err: database.ErrConflict, status: http.StatusConflict},
		{name: "invalid", err: database.ErrInvalid, status: http.StatusUnprocessableEntity},
		{name: "project not mirrored", err: database.ErrProjectNotMirrored, status: http.StatusServiceUnavailable},
		{name: "anything else", err: errors.New("boom"), status: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := storeError(testCase.err, "us-east-1")
			require.Equal(t, testCase.status, status(err))
			if testCase.status == 0 {
				require.Same(t, testCase.err, err)
			}
		})
	}

	// A Kubernetes status error keeps its status; anything else passes through.
	require.Equal(t, http.StatusForbidden, status(httpError(apierrors.NewForbidden(
		kargoapi.GroupVersion.WithResource("targets").GroupResource(), "x", errors.New("no"),
	))))
	plain := errors.New("plain")
	require.Same(t, plain, httpError(plain))
}
