package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTarget_GetStatus(t *testing.T) {
	t.Parallel()
	target := &Target{}
	require.Same(t, &target.Status, target.GetStatus())
}
