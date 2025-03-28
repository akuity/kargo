package v1alpha1

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetFreightByNameOrAlias returns a pointer to the Freight resource specified
// by the project, and name OR alias arguments. If no such resource is found,
// nil is returned instead.
func GetFreightByNameOrAlias(
	ctx context.Context,
	c client.Client,
	project string,
	name string,
	alias string,
) (*Freight, error) {
	if name != "" {
		return GetFreight(
			ctx,
			c,
			types.NamespacedName{
				Namespace: project,
				Name:      name,
			},
		)
	}
	return GetFreightByAlias(ctx, c, project, alias)
}

// GetFreight returns a pointer to the Freight resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetFreight(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Freight, error) {
	freight := Freight{}
	if err := c.Get(ctx, namespacedName, &freight); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Freight %q in namespace %q: %w",
			namespacedName.Name,
			namespacedName.Namespace,
			err,
		)
	}
	return &freight, nil
}

// GetFreightByAlias returns a pointer to the Freight resource specified by the
// project and alias arguments. If no such resource is found, nil is returned
// instead.
func GetFreightByAlias(
	ctx context.Context,
	c client.Client,
	project string,
	alias string,
) (*Freight, error) {
	freightList := FreightList{}
	if err := c.List(
		ctx,
		&freightList,
		client.InNamespace(project),
		client.MatchingLabels{
			AliasLabelKey: alias,
		},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing Freight with alias %q in namespace %q: %w",
			alias,
			project,
			err,
		)
	}
	if len(freightList.Items) == 0 {
		return nil, nil
	}
	return &freightList.Items[0], nil
}

// ListFreightByCurrentStage returns a list of Freight resources that think
// they're currently in use by the Stage specified.
func ListFreightByCurrentStage(
	ctx context.Context,
	c client.Client,
	stage *Stage,
) ([]Freight, error) {
	freightList := FreightList{}
	if err := c.List(
		ctx,
		&freightList,
		client.InNamespace(stage.Namespace),
		client.MatchingFields{"currentlyIn": stage.Name},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing Freight in namespace %q with current stage %q: %w",
			stage.Namespace,
			stage.Name,
			err,
		)
	}
	return freightList.Items, nil
}

// IsCurrentlyIn returns whether the Freight is currently in the specified
// Stage.
func (f *Freight) IsCurrentlyIn(stage string) bool {
	// NB: This method exists for convenience. It doesn't require the caller to
	// know anything about the Freight status' internal data structure.
	_, in := f.Status.CurrentlyIn[stage]
	return in
}

// IsVerifiedIn returns whether the Freight has been verified in the specified
// Stage.
func (f *Freight) IsVerifiedIn(stage string) bool {
	// NB: This method exists for convenience. It doesn't require the caller to
	// know anything about the Freight status' internal data structure.
	_, verified := f.Status.VerifiedIn[stage]
	return verified
}

// IsApprovedFor returns whether the Freight has been approved for the specified
// Stage.
func (f *Freight) IsApprovedFor(stage string) bool {
	// NB: This method exists for convenience. It doesn't require the caller to
	// know anything about the Freight status' internal data structure.
	_, approved := f.Status.ApprovedFor[stage]
	return approved
}

// GetLongestSoak returns the longest soak time for the Freight in the specified
// Stage if it's been verified in that Stage. If it has not, zero will be
// returned instead. If the Freight is currently in use by the specified Stage,
// the current soak time is calculated and compared to the longest completed
// soak time on record.
func (f *Freight) GetLongestSoak(stage string) time.Duration {
	if _, verified := f.Status.VerifiedIn[stage]; !verified {
		return 0
	}
	var longestCompleted time.Duration
	if record, isVerified := f.Status.VerifiedIn[stage]; isVerified && record.LongestCompletedSoak != nil {
		longestCompleted = record.LongestCompletedSoak.Duration
	}
	var current time.Duration
	if record, isCurrent := f.Status.CurrentlyIn[stage]; isCurrent {
		current = time.Since(record.Since.Time)
	}
	return time.Duration(max(longestCompleted.Nanoseconds(), current.Nanoseconds()))
}

// HasSoakedIn returns whether the Freight has soaked in the specified Stage for
// at least the specified duration. If the specified duration is nil, this
// method will return true.
func (f *Freight) HasSoakedIn(stage string, dur *metav1.Duration) bool {
	if f == nil {
		return false
	}
	if dur == nil {
		return true
	}
	return f.GetLongestSoak(stage) >= dur.Duration
}

// AddCurrentStage updates the Freight status to reflect that the Freight is
// currently in the specified Stage.
func (f *FreightStatus) AddCurrentStage(stage string, since time.Time) {
	if _, alreadyIn := f.CurrentlyIn[stage]; !alreadyIn {
		if f.CurrentlyIn == nil {
			f.CurrentlyIn = make(map[string]CurrentStage)
		}
		f.CurrentlyIn[stage] = CurrentStage{
			Since: &metav1.Time{Time: since},
		}
	}
}

// RemoveCurrentStage updates the Freight status to reflect that the Freight is
// no longer in the specified Stage. If the Freight was verified in the
// specified Stage, the longest completed soak time will be updated if
// necessary.
func (f *FreightStatus) RemoveCurrentStage(stage string) {
	if record, in := f.CurrentlyIn[stage]; in {
		if _, verified := f.VerifiedIn[stage]; verified {
			soak := time.Since(record.Since.Time)
			if soak > f.VerifiedIn[stage].LongestCompletedSoak.Duration {
				f.VerifiedIn[stage] = VerifiedStage{
					LongestCompletedSoak: &metav1.Duration{Duration: soak},
				}
			}
		}
		delete(f.CurrentlyIn, stage)
	}
}

// AddVerifiedStage updates the Freight status to reflect that the Freight has
// been verified in the specified Stage.
func (f *FreightStatus) AddVerifiedStage(stage string, verifiedAt time.Time) {
	if _, verified := f.VerifiedIn[stage]; !verified {
		record := VerifiedStage{VerifiedAt: &metav1.Time{Time: verifiedAt}}
		if f.VerifiedIn == nil {
			f.VerifiedIn = map[string]VerifiedStage{stage: record}
		}
		f.VerifiedIn[stage] = record
	}
}

// AddApprovedStage updates the Freight status to reflect that the Freight has
// been approved for the specified Stage.
func (f *FreightStatus) AddApprovedStage(stage string, approvedAt time.Time) {
	if _, approved := f.ApprovedFor[stage]; !approved {
		record := ApprovedStage{ApprovedAt: &metav1.Time{Time: approvedAt}}
		if f.ApprovedFor == nil {
			f.ApprovedFor = map[string]ApprovedStage{stage: record}
		}
		f.ApprovedFor[stage] = record
	}
}
