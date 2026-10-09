package v1alpha1

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPromotionStepStatus_Compare(t *testing.T) {
	tests := []struct {
		name     string
		lhs      PromotionStepStatus
		rhs      PromotionStepStatus
		expected int
	}{
		{
			name:     "Succeeded < Skipped",
			lhs:      PromotionStepStatusSucceeded,
			rhs:      PromotionStepStatusSkipped,
			expected: -1,
		},
		{
			name:     "Skipped < Running",
			lhs:      PromotionStepStatusSkipped,
			rhs:      PromotionStepStatusRunning,
			expected: -1,
		},
		{
			name:     "Running < Aborted",
			lhs:      PromotionStepStatusRunning,
			rhs:      PromotionStepStatusAborted,
			expected: -1,
		},
		{
			name:     "Aborted < Failed",
			lhs:      PromotionStepStatusAborted,
			rhs:      PromotionStepStatusFailed,
			expected: -1,
		},
		{
			name:     "Failed < Errored",
			lhs:      PromotionStepStatusFailed,
			rhs:      PromotionStepStatusErrored,
			expected: -1,
		},
		{
			name:     "Succeeded == Succeeded",
			lhs:      PromotionStepStatusSucceeded,
			rhs:      PromotionStepStatusSucceeded,
			expected: 0,
		},
		{
			name:     "Skipped == Skipped",
			lhs:      PromotionStepStatusSkipped,
			rhs:      PromotionStepStatusSkipped,
			expected: 0,
		},
		{
			name:     "Running == Running",
			lhs:      PromotionStepStatusRunning,
			rhs:      PromotionStepStatusRunning,
			expected: 0,
		},
		{
			name:     "Aborted == Aborted",
			lhs:      PromotionStepStatusAborted,
			rhs:      PromotionStepStatusAborted,
			expected: 0,
		},
		{
			name:     "Failed == Failed",
			lhs:      PromotionStepStatusFailed,
			rhs:      PromotionStepStatusFailed,
			expected: 0,
		},
		{
			name:     "Errored == Errored",
			lhs:      PromotionStepStatusErrored,
			rhs:      PromotionStepStatusErrored,
			expected: 0,
		},
		{
			name:     "Skipped > Succeeded",
			lhs:      PromotionStepStatusSkipped,
			rhs:      PromotionStepStatusSucceeded,
			expected: 1,
		},
		{
			name:     "Running > Skipped",
			lhs:      PromotionStepStatusRunning,
			rhs:      PromotionStepStatusSkipped,
			expected: 1,
		},
		{
			name:     "Aborted > Running",
			lhs:      PromotionStepStatusAborted,
			rhs:      PromotionStepStatusRunning,
			expected: 1,
		},
		{
			name:     "Failed > Aborted",
			lhs:      PromotionStepStatusFailed,
			rhs:      PromotionStepStatusAborted,
			expected: 1,
		},
		{
			name:     "Errored > Failed",
			lhs:      PromotionStepStatusErrored,
			rhs:      PromotionStepStatusFailed,
			expected: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.lhs.Compare(tt.rhs)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestPromotionRetry_GetTimeout(t *testing.T) {
	tests := []struct {
		name     string
		retry    *PromotionStepRetry
		fallback time.Duration
		want     time.Duration
	}{
		{
			name:     "retry is nil",
			retry:    nil,
			fallback: time.Hour,
			want:     time.Hour,
		},
		{
			name:     "timeout is not set",
			retry:    &PromotionStepRetry{},
			fallback: time.Hour,
			want:     time.Hour,
		},
		{
			name: "timeout is set",
			retry: &PromotionStepRetry{
				Timeout: &metav1.Duration{
					Duration: 3 * time.Hour,
				},
			},
			want: 3 * time.Hour,
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, tt.retry.GetTimeout(tt.fallback))
	}
}

func TestPromotionRetry_GetErrorThreshold(t *testing.T) {
	tests := []struct {
		name     string
		retry    *PromotionStepRetry
		fallback uint32
		want     uint32
	}{
		{
			name:     "retry is nil",
			retry:    nil,
			fallback: 1,
			want:     1,
		},
		{
			name:     "threshold is not set",
			retry:    &PromotionStepRetry{},
			fallback: 1,
			want:     1,
		},
		{
			name: "threshold is set",
			retry: &PromotionStepRetry{
				ErrorThreshold: 3,
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, tt.retry.GetErrorThreshold(tt.fallback))
	}
}

func TestStepExecutionMetadataList_HasFailures(t *testing.T) {
	tests := []struct {
		name     string
		metadata StepExecutionMetadataList
		expected bool
	}{
		{
			name: "no errors/failures at all",
			metadata: StepExecutionMetadataList{
				{Status: PromotionStepStatusSucceeded},
				{Status: PromotionStepStatusSucceeded},
				{Status: PromotionStepStatusSucceeded},
			},
			expected: false,
		},
		{
			name: "has an error",
			metadata: StepExecutionMetadataList{
				{Status: PromotionStepStatusSucceeded},
				{Status: PromotionStepStatusErrored},
				{Status: PromotionStepStatusSucceeded},
			},
			expected: true,
		},
		{
			name: "has an error with continueOnError == true",
			metadata: StepExecutionMetadataList{
				{Status: PromotionStepStatusSucceeded},
				{
					ContinueOnError: true,
					Status:          PromotionStepStatusErrored,
				},
				{Status: PromotionStepStatusSucceeded},
			},
			expected: false,
		},
		{
			name: "has a failure",
			metadata: StepExecutionMetadataList{
				{Status: PromotionStepStatusSucceeded},
				{Status: PromotionStepStatusFailed},
				{Status: PromotionStepStatusSucceeded},
			},
			expected: true,
		},
		{
			name: "has a failure with continueOnError == true",
			metadata: StepExecutionMetadataList{
				{Status: PromotionStepStatusSucceeded},
				{
					ContinueOnError: true,
					Status:          PromotionStepStatusFailed,
				},
				{Status: PromotionStepStatusSucceeded},
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.metadata.HasFailures())
		})
	}
}

// TestPromotionStep_DescriptionValidation guards the admission rules on
// PromotionStep.Description. They are enforced by the generated CRDs, so this
// asserts on the markers that generate them, in the same style as
// TestPromotionRequestSpec_Immutability: dropping or loosening one is a test
// failure rather than a silent behavior change.
//
// The length cap is a CEL rule and deliberately not MaxLength. The API server's
// built-in maxLength error says "bytes", although the limit counts characters.
func TestPromotionStep_DescriptionValidation(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "promotion_types.go", nil, parser.ParseComments)
	require.NoError(t, err)

	var doc string
	ast.Inspect(f, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != "PromotionStep" {
			return true
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return false
		}
		for _, field := range structType.Fields.List {
			if len(field.Names) > 0 && field.Names[0].Name == "Description" && field.Doc != nil {
				doc = field.Doc.Text()
			}
		}
		return false
	})
	require.NotEmpty(t, doc, "PromotionStep.Description has no doc comment")

	for _, marker := range []string{
		`+kubebuilder:validation:Optional`,
		// An empty description is rejected; wizards omit the key when blank.
		`+kubebuilder:validation:MinLength=1`,
		`+kubebuilder:validation:XValidation:` +
			`message="description must be 256 characters or fewer",` +
			`rule="self.size() <= 256"`,
	} {
		require.Contains(t, doc, marker)
	}
	require.NotContains(t, doc, "validation:MaxLength")
}
