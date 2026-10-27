package event

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// BasePrefix is the base subject prefix used for all Kargo events, including those published by
	// system components. It is the prefix that should be used in NATS subscriptions to receive all
	// Kargo events.
	BasePrefix = "akuity.kargo"

	// EventsSubjectPrefix is the subject prefix used for the user-facing events published by Kargo
	// components. Other system components will publish events on other subjects
	EventsSubjectPrefix = BasePrefix + ".events"
)

// ErrInvalidEventType is returned when an event type cannot be turned into subject tokens.
var ErrInvalidEventType = errors.New("event type must be non-empty and contain only ASCII letters and digits")

// NewKargoSubject returns the given topic prefixed with the global Kargo subject prefix.
//
// For example, if the topic is "my-topic", the returned subject will be "akuity.kargo.my-topic".
func NewKargoSubject(topic string) string {
	return fmt.Sprintf("%s.%s", BasePrefix, topic)
}

// NewEventSubject returns the subject a user-facing event of the given type is published on. It is
// shorthand for SubjectFromType(EventsSubjectPrefix, eventType). For example, a "PromotionCreated"
// event is published on "akuity.kargo.events.promotion.created".
//
// This is a convenience function for user-facing events that is a wrapper around `SubjectFromType`
// with the `EventsSubjectPrefix`.
func NewEventSubject(eventType string) (string, error) {
	return SubjectFromType(EventsSubjectPrefix, eventType)
}

// SubjectFromType returns the subject an event of the given type is published on under the given
// prefix: the prefix joined with the type's SubjectSuffix.
func SubjectFromType(subjectPrefix, eventType string) (string, error) {
	suffix, err := SubjectSuffix(eventType)
	if err != nil {
		return "", err
	}
	return subjectPrefix + "." + suffix, nil
}

// SubjectSuffix converts a PascalCase event type into the dot-separated subject tokens it is
// published under, so that each word of the type is its own level of the subject hierarchy and
// consumers can subscribe to e.g. "freight.verification.>" for all verification outcomes.
//
// Each word is lowercased. A run of capitals is kept together as a single word, ending where the
// next word begins, and digits stay attached to the word they follow:
//
//	PromotionCreated             -> promotion.created
//	FreightVerificationSucceeded -> freight.verification.succeeded
//	APITokenCreated              -> api.token.created
//	V2Created                    -> v2.created
//
// Only ASCII letters and digits are accepted, which guarantees every token is a valid NATS subject
// token. Any other input returns ErrInvalidEventType.
func SubjectSuffix(eventType string) (string, error) {
	if eventType == "" {
		return "", ErrInvalidEventType
	}
	for i := 0; i < len(eventType); i++ {
		if !isUpper(eventType[i]) && !isLower(eventType[i]) && !isDigit(eventType[i]) {
			return "", fmt.Errorf("%w: %q", ErrInvalidEventType, eventType)
		}
	}
	var b strings.Builder
	b.Grow(len(eventType) + 4)
	for i := 0; i < len(eventType); i++ {
		c := eventType[i]
		if i > 0 && isUpper(c) && startsWord(eventType, i) {
			b.WriteByte('.')
		}
		if isUpper(c) {
			c += 'a' - 'A'
		}
		b.WriteByte(c)
	}
	return b.String(), nil
}

// startsWord reports whether the capital letter at index i (i > 0) begins a new word. That is the
// case after a lowercase letter or digit ("promotionCreated", "v2Created"), or when it is the last
// capital of a run and a lowercase letter follows ("APIToken" splits before "Token").
func startsWord(s string, i int) bool {
	prev := s[i-1]
	if isLower(prev) || isDigit(prev) {
		return true
	}
	return i+1 < len(s) && isLower(s[i+1])
}

func isUpper(c byte) bool { return 'A' <= c && c <= 'Z' }

func isLower(c byte) bool { return 'a' <= c && c <= 'z' }

func isDigit(c byte) bool { return '0' <= c && c <= '9' }
