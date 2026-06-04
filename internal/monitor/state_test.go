package monitor

import (
	"testing"
	"time"

	"github.com/NoorDigitalAgency/vakt/internal/config"
)

func TestThresholdTrackerAlertFollowupRecovery(t *testing.T) {
	t.Parallel()

	tracker := NewThresholdTracker("cpu", config.ResourceThreshold{
		Percent:             80,
		SustainFor:          time.Minute,
		ClearFor:            2 * time.Minute,
		FollowupStepPercent: 5,
		Cooldown:            10 * time.Minute,
		ReminderAfter:       4 * time.Hour,
	})

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	if event := tracker.Evaluate(base, 81); event != nil {
		t.Fatalf("unexpected event on initial threshold breach: %#v", event)
	}
	event := tracker.Evaluate(base.Add(time.Minute), 82)
	if event == nil || event.Kind != EventAlert {
		t.Fatalf("expected alert event, got %#v", event)
	}
	if event.Current != 82 {
		t.Fatalf("alert current = %v, want 82", event.Current)
	}

	if event := tracker.Evaluate(base.Add(5*time.Minute), 84); event != nil {
		t.Fatalf("unexpected followup before cooldown: %#v", event)
	}

	event = tracker.Evaluate(base.Add(11*time.Minute), 88)
	if event == nil || event.Kind != EventFollowup {
		t.Fatalf("expected followup event, got %#v", event)
	}
	if event.PreviousAlert != 82 {
		t.Fatalf("followup previous alert = %v, want 82", event.PreviousAlert)
	}

	if event := tracker.Evaluate(base.Add(12*time.Minute), 70); event != nil {
		t.Fatalf("unexpected recovery before clear period: %#v", event)
	}
	event = tracker.Evaluate(base.Add(14*time.Minute), 70)
	if event == nil || event.Kind != EventRecovery {
		t.Fatalf("expected recovery event, got %#v", event)
	}
}

func TestThresholdTrackerReminder(t *testing.T) {
	t.Parallel()

	tracker := NewThresholdTracker("memory", config.ResourceThreshold{
		Percent:             80,
		SustainFor:          time.Minute,
		ClearFor:            5 * time.Minute,
		FollowupStepPercent: 10,
		Cooldown:            1 * time.Hour,
		ReminderAfter:       2 * time.Hour,
	})

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// trigger alert
	tracker.Evaluate(base, 85)
	event := tracker.Evaluate(base.Add(time.Minute), 85)
	if event == nil || event.Kind != EventAlert {
		t.Fatalf("expected alert event, got %#v", event)
	}

	// no reminder before ReminderAfter has elapsed
	if event := tracker.Evaluate(base.Add(time.Minute+time.Hour), 85); event != nil {
		t.Fatalf("unexpected event before reminder threshold: %#v", event)
	}

	// reminder fires after ReminderAfter
	event = tracker.Evaluate(base.Add(time.Minute+2*time.Hour), 86)
	if event == nil || event.Kind != EventReminder {
		t.Fatalf("expected reminder event, got %#v", event)
	}
	if event.Current != 86 {
		t.Fatalf("reminder current = %v, want 86", event.Current)
	}

	// no second reminder before another ReminderAfter elapses
	if event := tracker.Evaluate(base.Add(time.Minute+3*time.Hour), 86); event != nil {
		t.Fatalf("unexpected event before second reminder: %#v", event)
	}

	// second reminder fires
	event = tracker.Evaluate(base.Add(time.Minute+4*time.Hour), 86)
	if event == nil || event.Kind != EventReminder {
		t.Fatalf("expected second reminder event, got %#v", event)
	}
}

func TestThresholdTrackerFollowupResetsReminderTimer(t *testing.T) {
	t.Parallel()

	tracker := NewThresholdTracker("storage", config.ResourceThreshold{
		Percent:             70,
		SustainFor:          time.Minute,
		ClearFor:            5 * time.Minute,
		FollowupStepPercent: 5,
		Cooldown:            30 * time.Minute,
		ReminderAfter:       1 * time.Hour,
	})

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// trigger alert
	tracker.Evaluate(base, 75)
	event := tracker.Evaluate(base.Add(time.Minute), 75)
	if event == nil || event.Kind != EventAlert {
		t.Fatalf("expected alert event, got %#v", event)
	}

	// followup fires (usage climbed enough, cooldown elapsed)
	event = tracker.Evaluate(base.Add(35*time.Minute), 82)
	if event == nil || event.Kind != EventFollowup {
		t.Fatalf("expected followup event, got %#v", event)
	}

	// reminder should NOT fire yet; timer was reset by followup
	if event := tracker.Evaluate(base.Add(35*time.Minute+30*time.Minute), 82); event != nil {
		t.Fatalf("unexpected reminder after followup reset: %#v", event)
	}

	// reminder fires after ReminderAfter from the followup
	event = tracker.Evaluate(base.Add(35*time.Minute+time.Hour), 82)
	if event == nil || event.Kind != EventReminder {
		t.Fatalf("expected reminder event after followup reset, got %#v", event)
	}
}

func TestThresholdTrackerReminderClearedOnRecovery(t *testing.T) {
	t.Parallel()

	tracker := NewThresholdTracker("cpu", config.ResourceThreshold{
		Percent:             80,
		SustainFor:          time.Minute,
		ClearFor:            2 * time.Minute,
		FollowupStepPercent: 5,
		Cooldown:            1 * time.Hour,
		ReminderAfter:       2 * time.Hour,
	})

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// trigger alert
	tracker.Evaluate(base, 85)
	tracker.Evaluate(base.Add(time.Minute), 85)

	// trigger reminder
	event := tracker.Evaluate(base.Add(time.Minute+2*time.Hour), 85)
	if event == nil || event.Kind != EventReminder {
		t.Fatalf("expected reminder, got %#v", event)
	}

	// recovery
	tracker.Evaluate(base.Add(time.Minute+2*time.Hour+time.Minute), 70)
	event = tracker.Evaluate(base.Add(time.Minute+2*time.Hour+3*time.Minute), 70)
	if event == nil || event.Kind != EventRecovery {
		t.Fatalf("expected recovery, got %#v", event)
	}

	// breach again — alert fires fresh, reminder timer is reset
	tracker.Evaluate(base.Add(time.Minute+2*time.Hour+4*time.Minute), 85)
	event = tracker.Evaluate(base.Add(time.Minute+2*time.Hour+5*time.Minute), 85)
	if event == nil || event.Kind != EventAlert {
		t.Fatalf("expected fresh alert after recovery, got %#v", event)
	}

	// no reminder yet within the new ReminderAfter window
	if event := tracker.Evaluate(base.Add(time.Minute+2*time.Hour+6*time.Minute), 85); event != nil {
		t.Fatalf("unexpected reminder too soon after re-alert: %#v", event)
	}
}
