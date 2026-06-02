package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NoorDigitalAgency/vakt/internal/buildinfo"
)

func TestRunVersionDoesNotRequireConfig(t *testing.T) {
	t.Parallel()

	previous := buildinfo.Version
	buildinfo.Version = "test-version"
	defer func() { buildinfo.Version = previous }()

	var output bytes.Buffer
	if err := run([]string{"version"}, &output); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if got := strings.TrimSpace(output.String()); got != "test-version" {
		t.Fatalf("version output = %q, want %q", got, "test-version")
	}
}

func TestRunValidateConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `monitor:
  poll_interval: 15s
  snapshot_schedule: "0 */6 * * *"
  storage_path: "/"
  host_alias: ""
  http_timeout: 10s
notifications:
  slack:
    webhook_url: "https://hooks.slack.com/services/T000/B000/XXXX"
    channel: "#ops-alerts"
    username: "vakt"
    icon_emoji: ":satellite:"
thresholds:
  cpu:
    percent: 85
    sustain_for: 2m
    clear_for: 5m
    followup_step_percent: 5
    cooldown: 15m
  memory:
    percent: 90
    sustain_for: 2m
    clear_for: 5m
    followup_step_percent: 5
    cooldown: 15m
  storage:
    percent: 90
    sustain_for: 5m
    clear_for: 10m
    followup_step_percent: 3
    cooldown: 30m
`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var output bytes.Buffer
	if err := run([]string{"validate-config", "--config", configPath}, &output); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !strings.Contains(output.String(), "is valid") {
		t.Fatalf("validate-config output = %q, want substring %q", output.String(), "is valid")
	}
}
