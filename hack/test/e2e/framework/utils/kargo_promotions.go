//nolint:forcetypeassert
package utils

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/watch"
	"github.com/akuity/kargo/pkg/x/client/generated"
)

func PromoteAndWaitForPhase(
	ctx context.Context,
	t *testing.T,
	project, stage, freightName string,
	phase kargoapi.PromotionPhase,
	timeout time.Duration,
) (*kargoapi.Promotion, error) {
	promotion, err := PromoteAndWaitForCompletion(ctx, t, project, stage, freightName, timeout)
	if err != nil {
		return nil, err
	}
	if promotion.Status.Phase != phase {
		t.Fatalf(
			"Promotion '%v' did not finish with phase '%v', actual phase: '%v'",
			promotion.Name, phase, promotion.Status.Phase)
	}
	return promotion, err
}

func RefreshStage(
	ctx context.Context,
	_ *testing.T,
	project, stage string,
) error {
	kargoClient := ctx.Value(KargoCLIKey).(generated.APIClient)
	_, err := kargoClient.CoreAPI.RefreshStage(ctx, project, stage).Execute()
	return err
}

func PromoteAndWaitForCompletion(
	ctx context.Context,
	t *testing.T,
	project, stage, freightName string,
	timeout time.Duration,
) (*kargoapi.Promotion, error) {
	kargoClient := ctx.Value(KargoCLIKey).(generated.APIClient)

	_, httpRes, err := kargoClient.CoreAPI.GetStage(ctx, project, stage).Execute()
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
	if err != nil {
		t.Fatalf("error getting stage: %v", err)
	}

	promoteRes, httpRes, promoteErr := kargoClient.CoreAPI.
		PromoteToStage(ctx, project, stage).
		Body(generated.PromoteToStageRequest{
			Freight: &freightName,
		}).
		Execute()
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
	if promoteErr != nil {
		t.Fatalf("Error promoting %v, %v", promoteErr, promoteRes)
	}

	promoName := promoteRes.Metadata.Name
	if promoName == nil {
		t.Log("Promotion", promoteRes)
		t.Fatalf("Error promoting: promotion name is missing")
	}
	promotion, err := WaitForPromotion(ctx, t, project, *promoName, timeout)

	if err != nil {
		t.Fatalf("Error getting promotion %v", err)
	}
	return promotion, nil

}

func WaitForPromotion(
	ctx context.Context,
	_ *testing.T,
	project, name string,
	timeout time.Duration,
) (*kargoapi.Promotion, error) {
	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	watchClient := ctx.Value(KargoCLIWatchKey).(watch.Client)
	watchChan, errorChan := watchClient.WatchPromotion(timedCtx, project, name)
	for {
		select {
		case event := <-watchChan:
			if event.Object != nil {
				phase := event.Object.Status.Phase
				if phase == "" || phase == kargoapi.PromotionPhaseRunning || phase == kargoapi.PromotionPhasePending {
					continue
				}
				return event.Object, nil
			}
		case err := <-errorChan:
			if strings.Contains(err.Error(), "unexpected status 404") {
				// Retry wait on 404 until timeout
				watchChan, errorChan = watchClient.WatchPromotion(timedCtx, project, name)
			} else {
				return nil, err
			}
		case <-timedCtx.Done():
			return nil, errors.New("context canceled")
		}
	}
}

func WaitForLatestFreight(ctx context.Context, project, origin string, timeout time.Duration) (string, error) {
	watchClient := ctx.Value(KargoCLIWatchKey).(watch.Client)
	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	watchChan, errorChan := watchClient.WatchWarehouse(timedCtx, project, origin)
	for {
		select {
		case event := <-watchChan:
			if event.Object != nil && event.Object.Status.LastFreightID != "" {
				return event.Object.Status.LastFreightID, nil
			}
		case err := <-errorChan:
			return "", err
		case <-timedCtx.Done():
			return "", errors.New("context canceled")
		}
	}
}

func WaitForFreight(
	ctx context.Context,
	project, freightID string,
	timeout time.Duration, filter func(*kargoapi.Freight) bool,
) (*kargoapi.Freight, error) {
	watchClient := ctx.Value(KargoCLIWatchKey).(watch.Client)
	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	watchChan, errorChan := watchClient.WatchFreight(timedCtx, project, freightID)
	for {
		select {
		case event := <-watchChan:
			if filter(event.Object) {
				return event.Object, nil
			}
		case err := <-errorChan:
			return nil, err
		case <-timedCtx.Done():
			return nil, errors.New("context canceled")
		}
	}
}

func WaitForFreightToBeVerified(
	ctx context.Context,
	t *testing.T,
	project, freightID, stage string,
	timeout time.Duration,
) *kargoapi.Freight {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	freight, err := WaitForFreight(timeoutCtx, project, freightID, 10*time.Minute, func(freight *kargoapi.Freight) bool {
		if freight != nil {
			_, ok := freight.Status.VerifiedIn[stage]
			return ok
		}
		return false
	})
	if err != nil {
		t.Fatalf("Error waiting for freight to be verified %v", err)
	}
	// To an extra get to make sure cache is refreshed
	_, err = GetFreight(timeoutCtx, project, freightID)
	if err != nil {
		t.Fatalf("Error getting freight %s from api %v", freightID, err)
	}
	return freight
}

func GetFreight(ctx context.Context, project, freightID string) (*generated.Freight, error) {
	kargoClient := ctx.Value(KargoCLIKey).(generated.APIClient)

	freightOK, httpRes, err := kargoClient.CoreAPI.GetFreight(ctx, project, freightID).Execute()
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	fmt.Printf("FREIGHT: %v", freightOK)
	return freightOK, nil
}

// func getAnyFreight(kargoClient generated.APIClient, project, origin string) (*kargoapi.Freight, error) {

// 	params := core.NewQueryFreightsRestParams().WithProject(project).WithOrigins([]string{origin})

// 	freightRes, err := kargoClient.CoreAPI.QueryFreightsRest(params, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("Error querying freight %v", err)
// 	}

// 	// FIXME: change that once we make freight response typed
// 	var freightJSON []byte
// 	if freightJSON, err = json.Marshal(freightRes); err != nil {
// 		return nil, fmt.Errorf("marshal freight: %w", err)
// 	}
// 	// The response is {"groups": {"": {"items": [...]}}}
// 	type freightList struct {
// 		Items []*kargoapi.Freight `json:"items"`
// 	}
// 	var result struct {
// 		Groups map[string]*freightList `json:"groups"`
// 	}
// 	if err = json.Unmarshal(freightJSON, &result); err != nil {
// 		return nil, fmt.Errorf("unmarshal freight: %v", err)
// 	}
// 	freights := result.Groups[""].Items
// 	if len(freights) < 1 {
// 		return nil, fmt.Errorf("no freights found")
// 	}
// 	return freights[0], nil
// }
