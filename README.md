# vakt

Linux server monitoring CLI and systemd service for Ubuntu hosts.

## Features

- Polls CPU, RAM, and storage usage directly from `/proc` and `statfs`
- Posts resource snapshots to a Slack webhook on scheduled intervals and threshold events
- Sends one alert after sustained high usage, follow-up alerts after further increases, and recovery messages after sustained normalization
- Ships with a default YAML config, build script, and systemd installation scripts

## Configuration

Start from `configs/config.example.yaml` and copy it to `/etc/vakt/config.yaml`.

Key settings:

- `monitor.poll_interval`: how often resources are sampled
- `monitor.snapshot_schedule`: cron-style schedule for periodic snapshots
- `monitor.storage_path`: filesystem path used for storage monitoring
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

## Build

```bash
./scripts/build.sh
```

## Install as a service

1. Update `/etc/vakt/config.yaml` with your Slack webhook and thresholds.
2. Run `sudo ./scripts/install-service.sh`
3. Verify with `systemctl status vakt.service`

The installer builds a static Linux binary, installs it to `/usr/local/bin/vakt`, installs the default config if one is missing, and enables the packaged systemd unit.
