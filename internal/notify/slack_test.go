package notify

import (
	"strings"
	"testing"
	"time"

	"github.com/NoorDigitalAgency/vakt/internal/monitor"
)

func TestBuildPayloadIncludesSnapshotDetails(t *testing.T) {
	t.Parallel()

	notifier := &SlackNotifier{channel: "#ops", username: "vakt", iconEmoji: ":satellite:"}
	payload := notifier.buildPayload(Notification{
		Kind:      KindAlert,
		Resource:  "cpu",
		Current:   91.5,
		Threshold: 85,
		Snapshot: monitor.Snapshot{
			Hostname:            "server-1",
			PublicIP:            "203.0.113.10",
			CPUUsagePercent:     91.5,
			MemoryUsagePercent:  73.2,
			MemoryUsedBytes:     8 * 1024 * 1024 * 1024,
			MemoryTotalBytes:    16 * 1024 * 1024 * 1024,
			StorageUsagePercent: 88.1,
			StorageUsedBytes:    220 * 1024 * 1024 * 1024,
			StorageTotalBytes:   250 * 1024 * 1024 * 1024,
			Load1:               1.25,
			Load5:               1.1,
			Load15:              0.9,
			Uptime:              48 * time.Hour,
			StoragePath:         "/",
		},
	})

	if payload.Channel != "#ops" {
		t.Fatalf("payload.Channel = %q, want %q", payload.Channel, "#ops")
	}
	if len(payload.Blocks) != 2 {
		t.Fatalf("len(payload.Blocks) = %d, want 2", len(payload.Blocks))
	}
	if got := payload.Text; !strings.Contains(got, "CPU threshold breached") {
		t.Fatalf("payload.Text = %q, want CPU alert summary", got)
	}
	if got := payload.Blocks[1].Fields[2].Text; !strings.Contains(got, "220.00 GiB") {
		t.Fatalf("storage field = %q, want formatted size", got)
	}
	if got := payload.Blocks[1].Fields[4].Text; !strings.Contains(got, "server-1") {
		t.Fatalf("server field = %q, want server name", got)
	}
	if got := payload.Blocks[1].Fields[5].Text; !strings.Contains(got, "203.0.113.10") {
		t.Fatalf("public ip field = %q, want public ip", got)
	}
}

func TestHumanBytes(t *testing.T) {
	t.Parallel()

	testCases := map[uint64]string{
		512:                              "512 B",
		1024:                             "1.00 KiB",
		1024 * 1024:                      "1.00 MiB",
		5 * 1024 * 1024 * 1024:           "5.00 GiB",
		1024 * 1024 * 1024 * 1024 * 1024: "1.00 PiB",
	}

	for input, want := range testCases {
		if got := humanBytes(input); got != want {
			t.Fatalf("humanBytes(%d) = %q, want %q", input, got, want)
		}
	}
}
