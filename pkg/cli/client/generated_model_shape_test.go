package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/client/generated/models"
)

func TestGeneratedModelsUnmarshalAutoPromotionHoldScalars(t *testing.T) {
	t.Parallel()

	var spec models.PromotionSpec
	require.NoError(t, json.Unmarshal(
		[]byte(`{"freight":"fake-freight","stage":"fake-stage","source":"auto","steps":[{"uses":"fake-step"}]}`),
		&spec,
	))
	require.Equal(t, "auto", spec.Source)

	var hold models.AutoPromotionHold
	require.NoError(t, json.Unmarshal(
		[]byte(
			`{"freight":{"name":"fake-freight","origin":`+
				`{"kind":"Warehouse","name":"fake-warehouse"}},"state":"Active"}`,
		),
		&hold,
	))
	require.NotNil(t, hold.State)
	require.Equal(t, "Active", *hold.State)

	var status models.StageStatus
	require.NoError(t, json.Unmarshal(
		[]byte(
			`{"autoPromotionHolds":{"Warehouse/fake-warehouse":`+
				`{"freight":{"name":"fake-freight","origin":`+
				`{"kind":"Warehouse","name":"fake-warehouse"}},"state":"Pending"}}}`,
		),
		&status,
	))
	require.Contains(t, status.AutoPromotionHolds, "Warehouse/fake-warehouse")
	require.NotNil(t, status.AutoPromotionHolds["Warehouse/fake-warehouse"].State)
	require.Equal(t, "Pending", *status.AutoPromotionHolds["Warehouse/fake-warehouse"].State)

	var resumeReq models.ResumeStageAutoPromotionRequest
	require.NoError(t, json.Unmarshal(
		[]byte(`{"origin":{"kind":"Warehouse","name":"fake-warehouse"}}`),
		&resumeReq,
	))
	require.NotNil(t, resumeReq.Origin.Kind)
	require.NotNil(t, resumeReq.Origin.Name)
	require.Equal(t, "Warehouse", *resumeReq.Origin.Kind)
	require.Equal(t, "fake-warehouse", *resumeReq.Origin.Name)
}
