package monitor

import (
	"time"

	"github.com/NoorDigitalAgency/vakt/internal/config"
)

type EventKind string

const (
	EventAlert    EventKind = "alert"
	EventFollowup EventKind = "followup"
	EventRecovery EventKind = "recovery"
	EventReminder EventKind = "reminder"
)

type Event struct {
	Kind          EventKind
	Resource      string
	Current       float64
	PreviousAlert float64
	Threshold     config.ResourceThreshold
}

type ThresholdTracker struct {
	resource  string
	threshold config.ResourceThreshold

	active         bool
	aboveSince     time.Time
	belowSince     time.Time
	lastAlertValue float64
	lastAlertAt    time.Time
	lastReminderAt time.Time
}

func NewThresholdTracker(resource string, threshold config.ResourceThreshold) *ThresholdTracker {
	return &ThresholdTracker{resource: resource, threshold: threshold}
}

func (t *ThresholdTracker) Evaluate(now time.Time, value float64) *Event {
	if !t.active {
		if value >= t.threshold.Percent {
			if t.aboveSince.IsZero() {
				t.aboveSince = now
			}
			if now.Sub(t.aboveSince) >= t.threshold.SustainFor {
				t.active = true
				t.belowSince = time.Time{}
				t.lastAlertValue = value
				t.lastAlertAt = now
				return &Event{
					Kind:      EventAlert,
					Resource:  t.resource,
					Current:   value,
					Threshold: t.threshold,
				}
			}
		} else {
			t.aboveSince = time.Time{}
		}
		return nil
	}

	if value < t.threshold.Percent {
		if t.belowSince.IsZero() {
			t.belowSince = now
		}
		if now.Sub(t.belowSince) >= t.threshold.ClearFor {
			t.active = false
			t.aboveSince = time.Time{}
			t.belowSince = time.Time{}
			previousAlert := t.lastAlertValue
			t.lastAlertValue = 0
			t.lastAlertAt = time.Time{}
			t.lastReminderAt = time.Time{}
			return &Event{
				Kind:          EventRecovery,
				Resource:      t.resource,
				Current:       value,
				PreviousAlert: previousAlert,
				Threshold:     t.threshold,
			}
		}
		return nil
	}

	t.belowSince = time.Time{}
	if value >= t.lastAlertValue+t.threshold.FollowupStepPercent && now.Sub(t.lastAlertAt) >= t.threshold.Cooldown {
		previousAlert := t.lastAlertValue
		t.lastAlertValue = value
		t.lastAlertAt = now
		t.lastReminderAt = now
		return &Event{
			Kind:          EventFollowup,
			Resource:      t.resource,
			Current:       value,
			PreviousAlert: previousAlert,
			Threshold:     t.threshold,
		}
	}

	reminderBase := t.lastReminderAt
	if reminderBase.IsZero() {
		reminderBase = t.lastAlertAt
	}
	if t.threshold.ReminderAfter > 0 && now.Sub(reminderBase) >= t.threshold.ReminderAfter {
		t.lastReminderAt = now
		return &Event{
			Kind:      EventReminder,
			Resource:  t.resource,
			Current:   value,
			Threshold: t.threshold,
		}
	}

	return nil
}
