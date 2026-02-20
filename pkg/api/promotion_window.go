package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/robfig/cron/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CheckPromotionWindows checks all defined promotion windows to determine if
// promotion is allowed at the current time. It follows these rules:
//
//  1. If no promotion windows are defined, promotions are allowed by default.
//  2. If a allow window is defined, the current time must fall within the allow windows for
//     promotion to be allowed.
//  3. If a deny window is defined, the current time must not fall within any deny windows for
//     promotion to be allowed.
//  4. If both a allow and deny window is active, the deny window takes precedence and
//     promotions are denied.
// 5. If both a allow and deny window are defined, but none are active, promotions are denied.

func CheckPromotionWindows(ctx context.Context,
	currentTime time.Time,
	promotionWindows []kargoapi.PromotionWindowReference,
	k8sclient client.Client,
	project string,
) (bool, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.Debug("checking promotion windows")

	if len(promotionWindows) == 0 {
		logger.Debug("no promotion windows defined, allowing promotion by default")
		return true, nil
	}

	anyActiveAllowWindows := false
	anyAllowWindows := false
	for _, windowRef := range promotionWindows {
		windowSpec, err := getPromotionWindowSpec(ctx, windowRef, k8sclient, project)
		if err != nil {
			return false, fmt.Errorf("error getting PromotionWindow %q for PromotionPolicy in Project %q: %w",
				windowRef.Name, project, err)
		}
		active, err := checkPromotionWindow(ctx, currentTime, windowSpec)
		if err != nil {
			return false, fmt.Errorf("error checking PromotionWindow %q for PromotionPolicy in Project %q: %w",
				windowRef.Name, project, err)
		}
		switch windowSpec.Kind {
		case "allow":
			anyAllowWindows = true
			if active {
				anyActiveAllowWindows = true
			}
		case "deny":
			if active {
				return false, nil
			}
		default:
			return false, fmt.Errorf("unknown PromotionWindow kind %q in %q", windowSpec.Kind, windowRef.Name)
		}
	}

	if anyActiveAllowWindows {
		logger.Debug("active allow promotion windows")
		return true, nil
	}

	if anyAllowWindows {
		logger.Debug("no active allow promotion windows")
		return false, nil
	}

	return true, nil
}

// checkPromotionWindow checks if the current time falls within any of the defined
// promotion window. It returns true if promotion is active, false otherwise.
func checkPromotionWindow(ctx context.Context,
	currentTime time.Time,
	promotionWindowSpec *kargoapi.PromotionWindowSpec,
) (bool, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.Debug("checking promotion window spec")

	if promotionWindowSpec == nil {
		return false, errors.New("promotion window spec is nil")
	}

	cronParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	sched, err := cronParser.Parse(promotionWindowSpec.Schedule)
	if err != nil {
		return false, err
	}
	duration, err := time.ParseDuration(promotionWindowSpec.Duration)
	if err != nil {
		return false, err
	}
	if duration <= 0 {
		return false, fmt.Errorf("duration must be positive")
	}

	loc, err := time.LoadLocation(promotionWindowSpec.TimeZone)
	if err != nil {
		return false, fmt.Errorf("unable to load time zone: %w", err)
	}

	now := currentTime.In(loc)
	nextTime := sched.Next(now.Add(-duration))
	timeDiff := now.Sub(nextTime)

	if timeDiff < 0 || timeDiff >= duration {
		logger.Debug("promotion window is not active")
		return false, nil
	}

	logger.Debug("promotion window is active")
	return true, nil
}

// getPromotionWindowSpec retrieves the PromotionWindow spec from the given reference.
func getPromotionWindowSpec(ctx context.Context,
	ref kargoapi.PromotionWindowReference,
	k8sClient client.Client,
	project string,
) (*kargoapi.PromotionWindowSpec, error) {
	var spec kargoapi.PromotionWindowSpec

	if ref == (kargoapi.PromotionWindowReference{}) {
		return nil, errors.New("missing promotion window reference")
	}

	if k8sClient == nil {
		return nil, errors.New("k8s client is nil")
	}

	if project == "" {
		return nil, errors.New("project is empty")
	}

	switch ref.Kind {
	case "PromotionWindow", "":
		window := &kargoapi.PromotionWindow{}
		if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: project, Name: ref.Name}, window); err != nil {
			return nil, err
		}
		spec = window.Spec
	default:
		return nil, fmt.Errorf("unknown promotion window reference kind %q", ref.Kind)
	}

	return &spec, nil
}
