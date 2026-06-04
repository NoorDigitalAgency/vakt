package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadValidConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`monitor:
  poll_interval: 10s
  snapshot_schedule: "*/15 * * * *"
  storage_path: "/"
  server_name: "server-1"
  http_timeout: 5s
notifications:
  slack:
    webhook_url: "https://hooks.slack.com/services/T000/B000/XXXX"
thresholds:
  cpu:
    percent: 80
    sustain_for: 30s
    clear_for: 2m
    followup_step_percent: 4
    cooldown: 10m
    reminder_after: 4h
  memory:
    percent: 85
    sustain_for: 45s
    clear_for: 2m
    followup_step_percent: 5
    cooldown: 10m
    reminder_after: 4h
  storage:
    percent: 90
    sustain_for: 1m
    clear_for: 5m
    followup_step_percent: 2
    cooldown: 30m
    reminder_after: 4h
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := cfg.Monitor.PollInterval; got != 10*time.Second {
		t.Fatalf("poll interval = %v, want %v", got, 10*time.Second)
	}
	if got := cfg.Thresholds.Storage.FollowupStepPercent; got != 2 {
		t.Fatalf("storage followup step = %v, want 2", got)
	}
	if got := cfg.Thresholds.CPU.ReminderAfter; got != 4*time.Hour {
		t.Fatalf("cpu reminder_after = %v, want %v", got, 4*time.Hour)
	}
	if got := cfg.Monitor.ServerName; got != "server-1" {
		t.Fatalf("server name = %q, want %q", got, "server-1")
	}
}

func TestValidateRejectsInvalidWebhook(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Notifications.Slack.WebhookURL = "http://hooks.slack.com/invalid"
	cfg.Thresholds.CPU.Percent = 80
	cfg.Thresholds.Memory.Percent = 80
	cfg.Thresholds.Storage.Percent = 80

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}
