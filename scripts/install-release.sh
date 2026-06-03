#!/usr/bin/env bash
set -euo pipefail

REPO="NoorDigitalAgency/vakt"
VERSION="latest"
BINARY_DEST="/usr/local/bin/vakt"
CONFIG_DEST="/etc/vakt/config.yaml"
SERVICE_DEST="/etc/systemd/system/vakt.service"
INSTALL_DIR="/etc/vakt"
USE_DEFAULTS=0
TMP_DIR=""
PACKAGE_DIR=""

WEBHOOK_URL=""
CHANNEL=""
USERNAME=""
ICON_EMOJI=""
POLL_INTERVAL=""
SNAPSHOT_SCHEDULE=""
STORAGE_PATH=""
SERVER_NAME=""
HTTP_TIMEOUT=""
CPU_PERCENT=""
CPU_SUSTAIN_FOR=""
CPU_CLEAR_FOR=""
CPU_FOLLOWUP_STEP_PERCENT=""
CPU_COOLDOWN=""
MEMORY_PERCENT=""
MEMORY_SUSTAIN_FOR=""
MEMORY_CLEAR_FOR=""
MEMORY_FOLLOWUP_STEP_PERCENT=""
MEMORY_COOLDOWN=""
STORAGE_PERCENT=""
STORAGE_SUSTAIN_FOR=""
STORAGE_CLEAR_FOR=""
STORAGE_FOLLOWUP_STEP_PERCENT=""
STORAGE_COOLDOWN=""

usage() {
  cat <<'USAGE'
Usage: curl -fsSL https://raw.githubusercontent.com/NoorDigitalAgency/vakt/main/scripts/install-release.sh | sudo bash -s -- [options]

Options:
  --version <tag|latest>
  --webhook-url <url>
  --channel <channel>
  --username <name>
  --icon-emoji <emoji>
  --poll-interval <duration>
  --snapshot-schedule <cron>
  --storage-path <path>
  --server-name <name>
  --http-timeout <duration>
  --cpu-percent <number>
  --cpu-sustain-for <duration>
  --cpu-clear-for <duration>
  --cpu-followup-step-percent <number>
  --cpu-cooldown <duration>
  --memory-percent <number>
  --memory-sustain-for <duration>
  --memory-clear-for <duration>
  --memory-followup-step-percent <number>
  --memory-cooldown <duration>
  --storage-percent <number>
  --storage-sustain-for <duration>
  --storage-clear-for <duration>
  --storage-followup-step-percent <number>
  --storage-cooldown <duration>
  --defaults        accept defaults for any value not provided
  --help            show this help
USAGE
}

cleanup() {
  if [[ -n "$TMP_DIR" && -d "$TMP_DIR" ]]; then
    rm -rf "$TMP_DIR"
  fi
}

require_root() {
  if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
    echo "run as root" >&2
    exit 1
  fi
}

require_tools() {
  local tool
  for tool in curl tar sha256sum install systemctl mktemp; do
    if ! command -v "$tool" >/dev/null 2>&1; then
      echo "missing required tool: $tool" >&2
      exit 1
    fi
  done
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --version) VERSION="$2"; shift 2 ;;
      --webhook-url) WEBHOOK_URL="$2"; shift 2 ;;
      --channel) CHANNEL="$2"; shift 2 ;;
      --username) USERNAME="$2"; shift 2 ;;
      --icon-emoji) ICON_EMOJI="$2"; shift 2 ;;
      --poll-interval) POLL_INTERVAL="$2"; shift 2 ;;
      --snapshot-schedule) SNAPSHOT_SCHEDULE="$2"; shift 2 ;;
      --storage-path) STORAGE_PATH="$2"; shift 2 ;;
      --server-name|--host-alias) SERVER_NAME="$2"; shift 2 ;;
      --http-timeout) HTTP_TIMEOUT="$2"; shift 2 ;;
      --cpu-percent) CPU_PERCENT="$2"; shift 2 ;;
      --cpu-sustain-for) CPU_SUSTAIN_FOR="$2"; shift 2 ;;
      --cpu-clear-for) CPU_CLEAR_FOR="$2"; shift 2 ;;
      --cpu-followup-step-percent) CPU_FOLLOWUP_STEP_PERCENT="$2"; shift 2 ;;
      --cpu-cooldown) CPU_COOLDOWN="$2"; shift 2 ;;
      --memory-percent) MEMORY_PERCENT="$2"; shift 2 ;;
      --memory-sustain-for) MEMORY_SUSTAIN_FOR="$2"; shift 2 ;;
      --memory-clear-for) MEMORY_CLEAR_FOR="$2"; shift 2 ;;
      --memory-followup-step-percent) MEMORY_FOLLOWUP_STEP_PERCENT="$2"; shift 2 ;;
      --memory-cooldown) MEMORY_COOLDOWN="$2"; shift 2 ;;
      --storage-percent) STORAGE_PERCENT="$2"; shift 2 ;;
      --storage-sustain-for) STORAGE_SUSTAIN_FOR="$2"; shift 2 ;;
      --storage-clear-for) STORAGE_CLEAR_FOR="$2"; shift 2 ;;
      --storage-followup-step-percent) STORAGE_FOLLOWUP_STEP_PERCENT="$2"; shift 2 ;;
      --storage-cooldown) STORAGE_COOLDOWN="$2"; shift 2 ;;
      --defaults) USE_DEFAULTS=1; shift ;;
      --help|-h) usage; exit 0 ;;
      *) echo "unknown option: $1" >&2; usage; exit 1 ;;
    esac
  done
}

prompt_value() {
  local label="$1"
  local default_value="$2"
  local current_value="$3"

  if [[ -n "$current_value" ]]; then
    printf '%s' "$current_value"
    return
  fi

  if [[ "$USE_DEFAULTS" -eq 1 || ! -t 0 ]]; then
    printf '%s' "$default_value"
    return
  fi

  local entered=""
  read -r -p "$label [$default_value]: " entered < /dev/tty || true
  if [[ -z "$entered" ]]; then
    entered="$default_value"
  fi
  printf '%s' "$entered"
}

yaml_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

download_release() {
  local archive_url checksum_url

  TMP_DIR="$(mktemp -d)"
  trap cleanup EXIT

  archive_url="https://github.com/${REPO}/releases/download/${VERSION}/vakt_linux_amd64.tar.gz"
  checksum_url="${archive_url}.sha256"

  curl -fsSL "$archive_url" -o "$TMP_DIR/vakt_linux_amd64.tar.gz"
  curl -fsSL "$checksum_url" -o "$TMP_DIR/vakt_linux_amd64.tar.gz.sha256"
  (
    cd "$TMP_DIR"
    sha256sum -c vakt_linux_amd64.tar.gz.sha256
  )

  tar -xzf "$TMP_DIR/vakt_linux_amd64.tar.gz" -C "$TMP_DIR"
  PACKAGE_DIR="$TMP_DIR/vakt_linux_amd64"
}

configure_values() {
  local default_server_name
  default_server_name="${HOSTNAME:-$(hostname 2>/dev/null || uname -n 2>/dev/null || printf 'unknown-host')}"

  WEBHOOK_URL="$(prompt_value 'Slack webhook URL' 'https://hooks.slack.com/services/REPLACE/ME' "$WEBHOOK_URL")"
  CHANNEL="$(prompt_value 'Slack channel' '#ops-alerts' "$CHANNEL")"
  USERNAME="$(prompt_value 'Slack username' 'vakt' "$USERNAME")"
  ICON_EMOJI="$(prompt_value 'Slack icon emoji' ':satellite:' "$ICON_EMOJI")"
  POLL_INTERVAL="$(prompt_value 'Poll interval' '15s' "$POLL_INTERVAL")"
  SNAPSHOT_SCHEDULE="$(prompt_value 'Snapshot cron schedule' '0 */6 * * *' "$SNAPSHOT_SCHEDULE")"
  STORAGE_PATH="$(prompt_value 'Storage path' '/' "$STORAGE_PATH")"
  SERVER_NAME="$(prompt_value 'Server name' "$default_server_name" "$SERVER_NAME")"
  HTTP_TIMEOUT="$(prompt_value 'HTTP timeout' '10s' "$HTTP_TIMEOUT")"
  CPU_PERCENT="$(prompt_value 'CPU threshold percent' '85' "$CPU_PERCENT")"
  CPU_SUSTAIN_FOR="$(prompt_value 'CPU sustain period' '2m' "$CPU_SUSTAIN_FOR")"
  CPU_CLEAR_FOR="$(prompt_value 'CPU clear period' '5m' "$CPU_CLEAR_FOR")"
  CPU_FOLLOWUP_STEP_PERCENT="$(prompt_value 'CPU follow-up step percent' '5' "$CPU_FOLLOWUP_STEP_PERCENT")"
  CPU_COOLDOWN="$(prompt_value 'CPU follow-up cooldown' '15m' "$CPU_COOLDOWN")"
  MEMORY_PERCENT="$(prompt_value 'Memory threshold percent' '90' "$MEMORY_PERCENT")"
  MEMORY_SUSTAIN_FOR="$(prompt_value 'Memory sustain period' '2m' "$MEMORY_SUSTAIN_FOR")"
  MEMORY_CLEAR_FOR="$(prompt_value 'Memory clear period' '5m' "$MEMORY_CLEAR_FOR")"
  MEMORY_FOLLOWUP_STEP_PERCENT="$(prompt_value 'Memory follow-up step percent' '5' "$MEMORY_FOLLOWUP_STEP_PERCENT")"
  MEMORY_COOLDOWN="$(prompt_value 'Memory follow-up cooldown' '15m' "$MEMORY_COOLDOWN")"
  STORAGE_PERCENT="$(prompt_value 'Storage threshold percent' '90' "$STORAGE_PERCENT")"
  STORAGE_SUSTAIN_FOR="$(prompt_value 'Storage sustain period' '5m' "$STORAGE_SUSTAIN_FOR")"
  STORAGE_CLEAR_FOR="$(prompt_value 'Storage clear period' '10m' "$STORAGE_CLEAR_FOR")"
  STORAGE_FOLLOWUP_STEP_PERCENT="$(prompt_value 'Storage follow-up step percent' '3' "$STORAGE_FOLLOWUP_STEP_PERCENT")"
  STORAGE_COOLDOWN="$(prompt_value 'Storage follow-up cooldown' '30m' "$STORAGE_COOLDOWN")"
}

write_config() {
  mkdir -p "$INSTALL_DIR"
  
  # Backup existing config if it exists
  if [ -f "$CONFIG_DEST" ]; then
    local backup_path="${CONFIG_DEST}.backup.$(date +%Y%m%d-%H%M%S)"
    echo "Backing up existing config to $backup_path"
    cp "$CONFIG_DEST" "$backup_path"
  fi
  
  cat > "$CONFIG_DEST" <<CONFIG
monitor:
  poll_interval: ${POLL_INTERVAL}
  snapshot_schedule: "$(yaml_escape "$SNAPSHOT_SCHEDULE")"
  storage_path: "$(yaml_escape "$STORAGE_PATH")"
  server_name: "$(yaml_escape "$SERVER_NAME")"
  http_timeout: ${HTTP_TIMEOUT}

notifications:
  slack:
    webhook_url: "$(yaml_escape "$WEBHOOK_URL")"
    channel: "$(yaml_escape "$CHANNEL")"
    username: "$(yaml_escape "$USERNAME")"
    icon_emoji: "$(yaml_escape "$ICON_EMOJI")"

thresholds:
  cpu:
    percent: ${CPU_PERCENT}
    sustain_for: ${CPU_SUSTAIN_FOR}
    clear_for: ${CPU_CLEAR_FOR}
    followup_step_percent: ${CPU_FOLLOWUP_STEP_PERCENT}
    cooldown: ${CPU_COOLDOWN}
  memory:
    percent: ${MEMORY_PERCENT}
    sustain_for: ${MEMORY_SUSTAIN_FOR}
    clear_for: ${MEMORY_CLEAR_FOR}
    followup_step_percent: ${MEMORY_FOLLOWUP_STEP_PERCENT}
    cooldown: ${MEMORY_COOLDOWN}
  storage:
    percent: ${STORAGE_PERCENT}
    sustain_for: ${STORAGE_SUSTAIN_FOR}
    clear_for: ${STORAGE_CLEAR_FOR}
    followup_step_percent: ${STORAGE_FOLLOWUP_STEP_PERCENT}
    cooldown: ${STORAGE_COOLDOWN}
CONFIG
  chmod 644 "$CONFIG_DEST"
}

install_files() {
  install -Dm755 "$PACKAGE_DIR/vakt" "$BINARY_DEST"
  install -Dm644 "$PACKAGE_DIR/vakt.service" "$SERVICE_DEST"
  systemctl daemon-reload
  systemctl enable --now vakt.service
}

main() {
  parse_args "$@"
  require_root
  require_tools
  download_release
  configure_values
  write_config
  "$PACKAGE_DIR/vakt" validate-config --config "$CONFIG_DEST"
  install_files
  echo "vakt ${VERSION} installed and started"
}

main "$@"
