package kubeclient

import (
	"errors"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestIgnoreInvalid(t *testing.T) {
	testCases := []struct {
		name    string
		err     error
		wantErr bool
	}{
		{
			name:    "nil",
			err:     nil,
			wantErr: false,
		},
		{
			name:    "arbitrary error",
			err:     errors.New("not invalid"),
			wantErr: true,
		},
		{
			name: "invalid",
			err: apierrors.NewInvalid(
				schema.GroupKind{
					Group: kargoapi.GroupVersion.Group,
					Kind:  "Freight",
				},
				"freight",
				nil,
			),
			wantErr: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := IgnoreInvalid(testCase.err); (err != nil) != testCase.wantErr {
				t.Errorf("IgnoreInvalid() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}
