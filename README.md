# vakt

Linux server monitoring CLI and systemd service for Ubuntu hosts.

## Features

- Polls CPU, RAM, and storage usage directly from `/proc` and `statfs`
- Posts resource snapshots to a Slack webhook on scheduled intervals and threshold events
- Sends one alert after sustained high usage, follow-up alerts after further increases, and recovery messages after sustained normalization
- Ships with a default YAML config, build script, and systemd installation scripts
- Supports versioned release packaging and curl-based installation from GitHub Releases

## Configuration

Start from `configs/config.example.yaml` and copy it to `/etc/vakt/config.yaml`.

Key settings:

- `monitor.poll_interval`: how often resources are sampled
- `monitor.snapshot_schedule`: cron-style schedule for periodic snapshots
- `monitor.storage_path`: filesystem path used for storage monitoring
- `monitor.server_name`: display name included in snapshots; defaults to the machine hostname during release installation
- `thresholds.*.percent`: usage percentage that starts an alert cycle
- `thresholds.*.sustain_for`: how long the metric must stay above the threshold before the first alert
- `thresholds.*.followup_step_percent`: minimum increase since the last alert before a follow-up snapshot is sent
- `thresholds.*.cooldown`: minimum quiet period between follow-up snapshots
- `thresholds.*.clear_for`: how long the metric must stay below the threshold before recovery is reported

## CLI

```bash
go run ./cmd/vakt --config ./configs/config.example.yaml
```

Commands:

- `run`: start the monitoring loop (default)
- `snapshot`: send one manual snapshot to Slack
- `validate-config`: validate the YAML file and exit
- `version`: print the embedded build version

## Build

```bash
./scripts/build.sh
```

Create a release archive locally:

```bash
VERSION=dev ./scripts/package-release.sh
```

## Install as a service

1. Update `/etc/vakt/config.yaml` with your Slack webhook and thresholds.
2. Run `sudo ./scripts/install-service.sh`
3. Verify with `systemctl status vakt.service`

The installer builds a static Linux binary, installs it to `/usr/local/bin/vakt`, installs the default config if one is missing, and enables the packaged systemd unit.

## Install from GitHub Releases

Install the latest published build:

```bash
curl -fsSL https://raw.githubusercontent.com/NoorDigitalAgency/vakt/main/scripts/install-release.sh | sudo bash -s --
```

Install a specific release tag and accept defaults for any values you do not override:

```bash
curl -fsSL https://raw.githubusercontent.com/NoorDigitalAgency/vakt/main/scripts/install-release.sh | sudo bash -s -- \
  --version v2026.06.02.1 \
  --webhook-url https://hooks.slack.com/services/REPLACE/ME \
  --server-name web-01 \
  --defaults
```

The release installer downloads `vakt_linux_amd64.tar.gz`, verifies its SHA-256 checksum, prompts for configuration values with defaults, uses the current machine hostname as the default `monitor.server_name`, writes `/etc/vakt/config.yaml`, installs the systemd unit, and enables and starts `vakt.service`.

Snapshots include the configured server name and the detected public IP address when the public IP lookup succeeds.

## GitHub Actions release automation

`.github/workflows/release.yml` builds a Linux amd64 archive on every push to `main` and on manual dispatch, then publishes:

- a dated version tag in the form `vYYYY.MM.DD.N`
- a moving `latest` release with the same assets

Each release includes the archive and a `.sha256` checksum file for the curl installer.
