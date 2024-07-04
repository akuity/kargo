package conditions

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Getter is an interface that allows getting conditions from a (sub)resource.
type Getter interface {
	// GetConditions returns the conditions on the resource.
	GetConditions() []metav1.Condition
}

// Setter is an interface that allows setting conditions on a (sub)resource.
type Setter interface {
	Getter

	// SetConditions sets the conditions on the resource.
	SetConditions(conditions []metav1.Condition)
}

// Get returns the condition with the given type from the resource. If the
// condition does not exist, nil is returned.
func Get(on Getter, conditionType string) *metav1.Condition {
	if on == nil {
		return nil
	}

	for _, condition := range on.GetConditions() {
		if condition.Type == conditionType {
			return &condition
		}
	}

	return nil
}

// Set updates the conditions on the given resource. If a condition with the
// same type already exists, it is replaced. If the condition is new, it is
// appended to the list of conditions.
//
// If the setter also implements v1.Object, the observed generation is set
// on the condition.
func Set(on Setter, conditions ...*metav1.Condition) {
	if len(conditions) == 0 {
		return
	}

	existingConditions := on.GetConditions()
	newTransitionTime := metav1.NewTime(time.Now().UTC().Truncate(time.Second))

	// If the setter is also an object, get its generation
	var objGeneration int64
	if obj, ok := on.(metav1.Object); ok {
		objGeneration = obj.GetGeneration()
	}

	for _, condition := range conditions {
		if condition == nil {
			continue
		}

		// Set ObservedGeneration if applicable
		if objGeneration != 0 {
			condition.ObservedGeneration = objGeneration
		}

		updated := false
		for i, existing := range existingConditions {
			if existing.Type == condition.Type {
				if Equal(existing, *condition) && existing.ObservedGeneration >= condition.ObservedGeneration {
					updated = true
					break
				}

				if condition.LastTransitionTime.IsZero() {
					if !existing.LastTransitionTime.IsZero() {
						condition.LastTransitionTime = existing.LastTransitionTime
					} else {
						condition.LastTransitionTime = newTransitionTime
					}
				}

				// Replace existing condition
				existingConditions[i] = *condition
				updated = true
				break
			}
		}

		if !updated {
			// Condition not found, append new condition
			if condition.LastTransitionTime.IsZero() {
				condition.LastTransitionTime = newTransitionTime
			}
			existingConditions = append(existingConditions, *condition)
		}
	}

	on.SetConditions(existingConditions)
}

// Delete removes the condition with the given type from the resource.
func Delete(on Setter, conditionType string) {
	if on == nil || conditionType == "" {
		return
	}

	conditions := on.GetConditions()
	for i, existing := range conditions {
		if existing.Type == conditionType {
			on.SetConditions(append(conditions[:i], conditions[i+1:]...))
			return
		}
	}
}

// Equal returns true if the two conditions have the same type, status, reason,
// and message. It does not compare the last transition time or observed
// generation.
func Equal(a, b metav1.Condition) bool {
	return a.Type == b.Type && a.Status == b.Status && a.Reason == b.Reason && a.Message == b.Message
}
