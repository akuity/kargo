package v1alpha1

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/akuity/kargo/internal/logging"
)

type AppProjectSpec struct {
	SyncWindows SyncWindows `json:"syncWindows,omitempty"`
}

type SyncWindows []*SyncWindow

type SyncWindow struct {
	Kind         string   `json:"kind,omitempty"`
	Schedule     string   `json:"schedule,omitempty"`
	Duration     string   `json:"duration,omitempty"`
	Applications []string `json:"applications,omitempty"`
	Namespaces   []string `json:"namespaces,omitempty"`
	Clusters     []string `json:"clusters,omitempty"`
	ManualSync   bool     `json:"manualSync,omitempty"`
	TimeZone     string   `json:"timeZone,omitempty"`
}

func (s *SyncWindows) HasWindows() bool {
	return s != nil && len(*s) > 0
}

func (s *SyncWindows) Active() (*SyncWindows, error) {
	return s.active(time.Now())
}

func (s *SyncWindows) active(currentTime time.Time) (*SyncWindows, error) {
	// If SyncWindows.Active() is called outside of a UTC locale, it should be
	// first converted to UTC before we scan through the SyncWindows.
	currentTime = currentTime.In(time.UTC)

	if s.HasWindows() {
		var active SyncWindows
		specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		for _, w := range *s {
			schedule, sErr := specParser.Parse(w.Schedule)
			if sErr != nil {
				return nil, fmt.Errorf("cannot parse schedule '%s': %w", w.Schedule, sErr)
			}
			duration, dErr := time.ParseDuration(w.Duration)
			if dErr != nil {
				return nil, fmt.Errorf("cannot parse duration '%s': %w", w.Duration, dErr)
			}

			// Offset the nextWindow time to consider the timeZone of the sync window
			timeZoneOffsetDuration := w.scheduleOffsetByTimeZone()
			nextWindow := schedule.Next(currentTime.Add(timeZoneOffsetDuration - duration))
			if nextWindow.Before(currentTime.Add(timeZoneOffsetDuration)) {
				active = append(active, w)
			}
		}
		if len(active) > 0 {
			return &active, nil
		}
	}
	return nil, nil
}

func (s *SyncWindows) InactiveAllows() (*SyncWindows, error) {
	return s.inactiveAllows(time.Now())
}

func (s *SyncWindows) inactiveAllows(currentTime time.Time) (*SyncWindows, error) {
	// If SyncWindows.InactiveAllows() is called outside of a UTC locale, it should be
	// first converted to UTC before we scan through the SyncWindows.
	currentTime = currentTime.In(time.UTC)

	if s.HasWindows() {
		var inactive SyncWindows
		specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		for _, w := range *s {
			if w.Kind == "allow" {
				schedule, sErr := specParser.Parse(w.Schedule)
				if sErr != nil {
					return nil, fmt.Errorf("cannot parse schedule '%s': %w", w.Schedule, sErr)
				}
				duration, dErr := time.ParseDuration(w.Duration)
				if dErr != nil {
					return nil, fmt.Errorf("cannot parse duration '%s': %w", w.Duration, dErr)
				}
				// Offset the nextWindow time to consider the timeZone of the sync window
				timeZoneOffsetDuration := w.scheduleOffsetByTimeZone()
				nextWindow := schedule.Next(currentTime.Add(timeZoneOffsetDuration - duration))

				if !nextWindow.Before(currentTime.Add(timeZoneOffsetDuration)) {
					inactive = append(inactive, w)
				}
			}
		}
		if len(inactive) > 0 {
			return &inactive, nil
		}
	}
	return nil, nil
}

func (s *SyncWindow) scheduleOffsetByTimeZone() time.Duration {
	loc, err := time.LoadLocation(s.TimeZone)
	if err != nil {
		logging.LoggerFromContext(context.TODO()).Error(
			fmt.Errorf("invalid time zone %s specified", s.TimeZone),
			"using UTC as default time zone",
		)
		loc = time.Now().UTC().Location()
	}
	_, tzOffset := time.Now().In(loc).Zone()
	return time.Duration(tzOffset) * time.Second
}

func (s *SyncWindows) Matches(app *Application) *SyncWindows {
	if s.HasWindows() {
		var matchingWindows SyncWindows
		for _, w := range *s {
			if len(w.Applications) > 0 {
				for _, a := range w.Applications {
					if globMatch(a, app.Name, false) {
						matchingWindows = append(matchingWindows, w)
						break
					}
				}
			}
			if len(w.Clusters) > 0 {
				for _, c := range w.Clusters {
					dst := app.Spec.Destination
					dstNameMatched := dst.Name != "" && globMatch(c, dst.Name, false)
					dstServerMatched := dst.Server != "" && globMatch(c, dst.Server, false)
					if dstNameMatched || dstServerMatched {
						matchingWindows = append(matchingWindows, w)
						break
					}
				}
			}
			if len(w.Namespaces) > 0 {
				for _, n := range w.Namespaces {
					if globMatch(n, app.Spec.Destination.Namespace, false) {
						matchingWindows = append(matchingWindows, w)
						break
					}
				}
			}
		}
		if len(matchingWindows) > 0 {
			return &matchingWindows
		}
	}
	return nil
}

func (s *SyncWindows) CanSync(isManual bool) (bool, error) {
	if !s.HasWindows() {
		return true, nil
	}

	active, err := s.Active()
	if err != nil {
		return false, fmt.Errorf("invalid sync windows: %w", err)
	}
	hasActiveDeny, manualEnabled := active.hasDeny()

	if hasActiveDeny {
		if isManual && manualEnabled {
			return true, nil
		}
		return false, nil
	}

	if active.hasAllow() {
		return true, nil
	}

	inactiveAllows, err := s.InactiveAllows()
	if err != nil {
		return false, fmt.Errorf("invalid sync windows: %w", err)
	}
	if inactiveAllows.HasWindows() {
		if isManual && inactiveAllows.manualEnabled() {
			return true, nil
		}
		return false, nil
	}

	return true, nil
}

func (s *SyncWindows) hasDeny() (bool, bool) {
	if !s.HasWindows() {
		return false, false
	}
	var denyFound, manualEnabled bool
	for _, a := range *s {
		if a.Kind == "deny" {
			if !denyFound {
				manualEnabled = a.ManualSync
			} else if manualEnabled {
				manualEnabled = a.ManualSync
			}
			denyFound = true
		}
	}
	return denyFound, manualEnabled
}

func (s *SyncWindows) hasAllow() bool {
	if !s.HasWindows() {
		return false
	}
	for _, a := range *s {
		if a.Kind == "allow" {
			return true
		}
	}
	return false
}

func (s *SyncWindows) manualEnabled() bool {
	if !s.HasWindows() {
		return false
	}
	for _, s := range *s {
		if !s.ManualSync {
			return false
		}
	}
	return true
}
