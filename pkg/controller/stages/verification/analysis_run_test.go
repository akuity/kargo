package verification

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// fakeAnalysisRunner is a minimal AnalysisRunner implementation for exercising
// the Verifier's runner-selection and dispatch logic without a Kubernetes
// client.
type fakeAnalysisRunner struct {
	validateErr error
	createRun   *AnalysisRun
	createMsg   string
	createErr   error
}

func (f *fakeAnalysisRunner) Validate(string) error { return f.validateErr }

func (f *fakeAnalysisRunner) GetAnalysisRunStatus(
	context.Context,
	kargoapi.AnalysisRunReference,
) (*AnalysisRunStatus, error) {
	return nil, nil
}

func (f *fakeAnalysisRunner) AbortAnalysisRun(
	context.Context,
	kargoapi.AnalysisRunReference,
) error {
	return nil
}

func (f *fakeAnalysisRunner) FindExistingAnalysisRun(
	context.Context,
	types.NamespacedName,
	string,
) (*AnalysisRun, error) {
	return nil, nil
}

func (f *fakeAnalysisRunner) CreateAnalysisRun(
	context.Context,
	kargoapi.Stage,
	kargoapi.FreightCollection,
	string,
) (*AnalysisRun, string, error) {
	return f.createRun, f.createMsg, f.createErr
}

func (f *fakeAnalysisRunner) ClearAnalysisRunsForStage(context.Context, *kargoapi.Stage) error {
	return nil
}

func TestVerifier_createAnalysisRun(t *testing.T) {
	tests := []struct {
		name         string
		selectRunner func(kargoapi.Stage, map[kargoapi.AnalysisRunKind]AnalysisRunner) (AnalysisRunner, bool, error)
		assertions   func(*testing.T, *AnalysisRun, string, error)
	}{
		{
			name: "runner selection returns ErrInvalid: message without bubbling error",
			selectRunner: func(
				kargoapi.Stage,
				map[kargoapi.AnalysisRunKind]AnalysisRunner,
			) (AnalysisRunner, bool, error) {
				return nil, false, fmt.Errorf("integration disabled: %w", ErrInvalid)
			},
			assertions: func(t *testing.T, run *AnalysisRun, msg string, err error) {
				// ErrInvalid must not be returned as an error, to avoid
				// reconciler restarts, but should surface as a message.
				require.NoError(t, err)
				assert.Nil(t, run)
				assert.Contains(t, msg, "integration disabled")
			},
		},
		{
			name: "runner selection returns generic error: message and error",
			selectRunner: func(
				kargoapi.Stage,
				map[kargoapi.AnalysisRunKind]AnalysisRunner,
			) (AnalysisRunner, bool, error) {
				return nil, false, fmt.Errorf("boom")
			},
			assertions: func(t *testing.T, run *AnalysisRun, msg string, err error) {
				require.ErrorContains(t, err, "boom")
				assert.Nil(t, run)
				assert.Equal(t, "boom", msg)
			},
		},
		{
			name: "no runner found: error returned",
			selectRunner: func(
				kargoapi.Stage,
				map[kargoapi.AnalysisRunKind]AnalysisRunner,
			) (AnalysisRunner, bool, error) {
				return nil, false, nil
			},
			assertions: func(t *testing.T, run *AnalysisRun, msg string, err error) {
				require.ErrorContains(t, err, "analysis runner not found for stage")
				assert.Nil(t, run)
				assert.Empty(t, msg)
			},
		},
		{
			name: "runner found: delegates to runner",
			selectRunner: func(
				_ kargoapi.Stage,
				runners map[kargoapi.AnalysisRunKind]AnalysisRunner,
			) (AnalysisRunner, bool, error) {
				return runners[kargoapi.AnalysisRunKindRun], true, nil
			},
			assertions: func(t *testing.T, run *AnalysisRun, msg string, err error) {
				require.NoError(t, err)
				require.NotNil(t, run)
				assert.Equal(t, "created-run", run.Name)
				assert.Empty(t, msg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRunner := &fakeAnalysisRunner{
				createRun: &AnalysisRun{Name: "created-run"},
			}
			ver := Verifier{
				analysisRunners: map[kargoapi.AnalysisRunKind]AnalysisRunner{
					kargoapi.AnalysisRunKindRun: fakeRunner,
				},
				selectRunnerForCreateFn: tt.selectRunner,
			}

			run, msg, err := ver.createAnalysisRun(
				t.Context(),
				kargoapi.Stage{ObjectMeta: metav1.ObjectMeta{Name: "test-stage"}},
				kargoapi.FreightCollection{},
				"",
			)
			tt.assertions(t, run, msg, err)
		})
	}
}
