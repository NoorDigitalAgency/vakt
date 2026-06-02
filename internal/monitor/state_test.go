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
