//nolint:forcetypeassert
package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/watch"
	"github.com/akuity/kargo/pkg/x/client/generated"
)

// PromoteAndWaitForPhase starts a promotion of freightName to stage and waits
// until the promotion reaches phase, which may be a running or a terminal
// phase. It asserts the observed phase equals phase.
func PromoteAndWaitForPhase(
	ctx context.Context,
	t *testing.T,
	project, stage, freightName string,
	phase kargoapi.PromotionPhase,
	timeout time.Duration,
) (*kargoapi.Promotion, error) {
	name := StartPromotion(ctx, t, project, stage, freightName, timeout)
	return WaitForPromotionPhase(ctx, t, project, name, phase, timeout)
}

// PromoteWithPRMerge promotes freightName to a stage whose promotion opens a
// pull request and blocks on git-wait-for-pr. It waits for the promotion to be
// Running, reads the pull request number recorded by the git-open-pr step
// (identified by prStepAlias), merges that pull request via the GitHub API
// using token, then waits for the promotion to succeed.
func PromoteWithPRMerge(
	ctx context.Context,
	t *testing.T,
	project, stage, freightName string,
	repoURL, token, prStepAlias string,
	timeout time.Duration,
) (*kargoapi.Promotion, error) {
	running, err := PromoteAndWaitForPhase(
		ctx, t,
		project, stage, freightName,
		kargoapi.PromotionPhaseRunning,
		timeout,
	)
	if err != nil {
		return nil, err
	}

	prNumber := WaitForPullRequestID(ctx, t, project, running.Name, prStepAlias, timeout)

	if err := MergePullRequest(ctx, repoURL, token, prNumber, timeout); err != nil {
		t.Fatalf("error merging pull request %d: %v", prNumber, err)
	}

	return WaitForPromotionPhase(
		ctx, t,
		project, running.Name,
		kargoapi.PromotionPhaseSucceeded,
		timeout,
	)
}

// StartPromotion issues a promote request for freightName to stage and returns
// the name of the created Promotion.
//
// A Stage transiently rejects a promotion with 400 Bad Request while the
// freight is still being qualified in an upstream stage, so the request is
// retried on 400 until it is accepted or timeout elapses. Any other error is
// fatal immediately.
func StartPromotion(
	ctx context.Context,
	t *testing.T,
	project, stage, freightName string,
	timeout time.Duration,
) string {
	kargoClient := ctx.Value(KargoCLIKey).(generated.APIClient)

	_, httpRes, err := kargoClient.CoreAPI.GetStage(ctx, project, stage).Execute()
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
	if err != nil {
		t.Fatalf("error getting stage: %v", err)
	}

	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		promoteRes, httpRes, promoteErr := kargoClient.CoreAPI.
			PromoteToStage(timedCtx, project, stage).
			Body(generated.PromoteToStageRequest{
				Freight: &freightName,
			}).
			Execute()
		statusCode := 0
		if httpRes != nil {
			statusCode = httpRes.StatusCode
			_ = httpRes.Body.Close()
		}

		if promoteErr == nil {
			if promoteRes.Metadata.Name == nil {
				t.Log("Promotion", promoteRes)
				t.Fatalf("Error promoting: promotion name is missing")
			}
			return *promoteRes.Metadata.Name
		}

		// Only the transient "stage not ready for this freight yet" case is
		// retried; every other failure is reported immediately.
		if statusCode != http.StatusBadRequest {
			t.Fatalf("Error promoting: %v (response: %v, http: %v)", promoteErr, promoteRes, httpRes)
		}

		t.Logf("Stage %q not ready to accept freight yet (400), retrying promotion", stage)
		select {
		case <-timedCtx.Done():
			t.Fatalf("Error promoting after retrying for %v: %v", timeout, promoteErr)
		case <-ticker.C:
		}
	}
}

// TryPromoteToStage issues a single promote request for freightName to stage
// without retrying, returning the HTTP status code and error. It is used to
// assert whether a promotion is currently permitted (e.g. before a required
// soak time has elapsed), where an unavailable freight yields 400 Bad Request.
func TryPromoteToStage(
	ctx context.Context,
	project, stage, freightName string,
) (int, error) {
	kargoClient := ctx.Value(KargoCLIKey).(generated.APIClient)
	_, httpRes, promoteErr := kargoClient.CoreAPI.
		PromoteToStage(ctx, project, stage).
		Body(generated.PromoteToStageRequest{
			Freight: &freightName,
		}).
		Execute()
	statusCode := 0
	if httpRes != nil {
		statusCode = httpRes.StatusCode
		_ = httpRes.Body.Close()
	}
	return statusCode, promoteErr
}

// WaitForPromotionPhase watches the named promotion until it reaches phase or
// any terminal phase, whichever comes first, then asserts the observed phase
// equals phase. Passing a terminal phase makes it wait for completion.
func WaitForPromotionPhase(
	ctx context.Context,
	t *testing.T,
	project, name string,
	phase kargoapi.PromotionPhase,
	timeout time.Duration,
) (*kargoapi.Promotion, error) {
	promotion, err := watchPromotionForPhase(ctx, project, name, phase, timeout)
	if err != nil {
		t.Fatalf("Error waiting for promotion %q: %v", name, err)
	}
	if promotion.Status.Phase != phase {
		t.Fatalf(
			"Promotion '%v' did not reach phase '%v', actual phase: '%v'. Message: '%v'",
			promotion.Name, phase, promotion.Status.Phase, promotion.Status.Message)
	}
	return promotion, nil
}

// watchPromotionForPhase returns the promotion once its phase equals phase or
// is terminal.
func watchPromotionForPhase(
	ctx context.Context,
	project, name string,
	phase kargoapi.PromotionPhase,
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
				current := event.Object.Status.Phase
				if current == phase || current.IsTerminal() {
					return event.Object, nil
				}
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

// WaitForPullRequestID watches the named promotion until the git-open-pr step
// with the given alias records a pull request id (its pr.id output), returning
// it. It fails the test if the promotion reaches a terminal phase first or the
// timeout elapses.
func WaitForPullRequestID(
	ctx context.Context,
	t *testing.T,
	project, name, stepAlias string,
	timeout time.Duration,
) int {
	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	watchClient := ctx.Value(KargoCLIWatchKey).(watch.Client)
	watchChan, errorChan := watchClient.WatchPromotion(timedCtx, project, name)
	for {
		select {
		case event := <-watchChan:
			if event.Object != nil {
				if id, ok := pullRequestIDFromState(event.Object, stepAlias); ok {
					return id
				}
				if event.Object.Status.Phase.IsTerminal() {
					t.Fatalf(
						"promotion %q reached terminal phase %q before recording a pull request id. Full status: %v",
						name, event.Object.Status.Phase, event.Object.Status)
				}
			}
		case err := <-errorChan:
			if strings.Contains(err.Error(), "unexpected status 404") {
				watchChan, errorChan = watchClient.WatchPromotion(timedCtx, project, name)
			} else {
				t.Fatalf("error watching promotion %q: %v", name, err)
			}
		case <-timedCtx.Done():
			t.Fatalf("timed out waiting for pull request id from promotion %q", name)
		}
	}
}

// PromotionStepOutput returns the string value stored by the step with the
// given alias under key in the promotion's shared state (i.e. the value a step
// referenced as outputs[stepAlias].key).
func PromotionStepOutput(promotion *kargoapi.Promotion, stepAlias, key string) (string, bool) {
	stepOutput, ok := promotion.Status.GetState()[stepAlias].(map[string]any)
	if !ok {
		return "", false
	}
	value, ok := stepOutput[key].(string)
	return value, ok
}

// pullRequestIDFromState extracts the pull request id recorded by the
// git-open-pr step, stored in the promotion state under stepAlias -> pr -> id.
func pullRequestIDFromState(promotion *kargoapi.Promotion, stepAlias string) (int, bool) {
	stepOutput, ok := promotion.Status.GetState()[stepAlias].(map[string]any)
	if !ok {
		return 0, false
	}
	pr, ok := stepOutput["pr"].(map[string]any)
	if !ok {
		return 0, false
	}
	switch id := pr["id"].(type) {
	case float64:
		return int(id), true
	case int64:
		return int(id), true
	case int:
		return id, true
	default:
		return 0, false
	}
}

func RefreshStage(
	ctx context.Context,
	_ *testing.T,
	project, stage string,
) error {
	kargoClient := ctx.Value(KargoCLIKey).(generated.APIClient)
	httpRes, err := kargoClient.CoreAPI.RefreshStage(ctx, project, stage).Execute()
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
	return err
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

// WaitForStageVerified watches the named Stage until its most recent Freight
// selection has been verified successfully, returning the Stage. It fails the
// test if verification reaches a terminal, non-successful phase or the timeout
// elapses.
func WaitForStageVerified(
	ctx context.Context,
	t *testing.T,
	project, stage string,
	timeout time.Duration,
) *kargoapi.Stage {
	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	watchClient := ctx.Value(KargoCLIWatchKey).(watch.Client)
	watchChan, errorChan := watchClient.WatchStage(timedCtx, project, stage)
	for {
		select {
		case event := <-watchChan:
			if event.Object == nil {
				continue
			}
			info := currentStageVerification(event.Object)
			if info == nil {
				continue
			}
			switch info.Phase {
			case kargoapi.VerificationPhaseSuccessful:
				return event.Object
			case kargoapi.VerificationPhaseFailed,
				kargoapi.VerificationPhaseError,
				kargoapi.VerificationPhaseAborted,
				kargoapi.VerificationPhaseInconclusive:
				t.Fatalf("stage %q verification finished with phase %q: %s", stage, info.Phase, info.Message)
			}
		case err := <-errorChan:
			if strings.Contains(err.Error(), "unexpected status 404") {
				watchChan, errorChan = watchClient.WatchStage(timedCtx, project, stage)
			} else {
				t.Fatalf("error watching stage %q: %v", stage, err)
			}
		case <-timedCtx.Done():
			t.Fatalf("timed out waiting for stage %q to be verified", stage)
		}
	}
}

// currentStageVerification returns the verification info for the Stage's most
// recent Freight selection, or nil if none has been recorded yet.
func currentStageVerification(stage *kargoapi.Stage) *kargoapi.VerificationInfo {
	current := stage.Status.FreightHistory.Current()
	if current == nil {
		return nil
	}
	return current.VerificationHistory.Current()
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
