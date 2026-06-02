package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/NoorDigitalAgency/vakt/internal/config"
	"github.com/NoorDigitalAgency/vakt/internal/monitor"
)

type Kind string

const (
	KindAlert     Kind = "alert"
	KindFollowup  Kind = "followup"
	KindRecovery  Kind = "recovery"
	KindScheduled Kind = "scheduled"
	KindManual    Kind = "manual"
)

type Notification struct {
	Kind          Kind
	Resource      string
	Current       float64
	PreviousAlert float64
	Threshold     float64
	Snapshot      monitor.Snapshot
}

type SlackNotifier struct {
	webhookURL string
	channel    string
	username   string
	iconEmoji  string
	client     *http.Client
}

func NewSlackNotifier(cfg config.Config) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: cfg.Notifications.Slack.WebhookURL,
		channel:    cfg.Notifications.Slack.Channel,
		username:   cfg.Notifications.Slack.Username,
		iconEmoji:  cfg.Notifications.Slack.IconEmoji,
		client: &http.Client{
			Timeout: cfg.Monitor.HTTPTimeout,
		},
	}
}

func (s *SlackNotifier) Send(ctx context.Context, notification Notification) error {
	payload := s.buildPayload(notification)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send slack request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("slack webhook returned %s", resp.Status)
	}
	return nil
}

type slackPayload struct {
	Text      string       `json:"text"`
	Channel   string       `json:"channel,omitempty"`
	Username  string       `json:"username,omitempty"`
	IconEmoji string       `json:"icon_emoji,omitempty"`
	Blocks    []slackBlock `json:"blocks,omitempty"`
}

type slackBlock struct {
	Type   string      `json:"type"`
	Text   *slackText  `json:"text,omitempty"`
	Fields []slackText `json:"fields,omitempty"`
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *SlackNotifier) buildPayload(notification Notification) slackPayload {
	title, intro := messageHeader(notification)
	snapshot := notification.Snapshot
	metrics := []slackText{
		{Type: "mrkdwn", Text: fmt.Sprintf("*CPU*\n%.2f%%", snapshot.CPUUsagePercent)},
		{Type: "mrkdwn", Text: fmt.Sprintf("*Memory*\n%.2f%% (%s / %s)", snapshot.MemoryUsagePercent, humanBytes(snapshot.MemoryUsedBytes), humanBytes(snapshot.MemoryTotalBytes))},
		{Type: "mrkdwn", Text: fmt.Sprintf("*Storage*\n%.2f%% (%s / %s on `%s`)", snapshot.StorageUsagePercent, humanBytes(snapshot.StorageUsedBytes), humanBytes(snapshot.StorageTotalBytes), snapshot.StoragePath)},
		{Type: "mrkdwn", Text: fmt.Sprintf("*Load*\n%.2f / %.2f / %.2f", snapshot.Load1, snapshot.Load5, snapshot.Load15)},
		{Type: "mrkdwn", Text: fmt.Sprintf("*Server*\n`%s`", snapshot.Hostname)},
		{Type: "mrkdwn", Text: fmt.Sprintf("*Public IP*\n`%s`", publicIPText(snapshot.PublicIP))},
		{Type: "mrkdwn", Text: fmt.Sprintf("*Uptime*\n%s", snapshot.Uptime.Round(time.Second))},
	}

	blocks := []slackBlock{
		{
			Type: "section",
			Text: &slackText{Type: "mrkdwn", Text: fmt.Sprintf("*%s*\n%s", title, intro)},
		},
		{
			Type:   "section",
			Fields: metrics,
		},
	}

	return slackPayload{
		Text:      strings.TrimSpace(title + " - " + intro),
		Channel:   s.channel,
		Username:  s.username,
		IconEmoji: s.iconEmoji,
		Blocks:    blocks,
	}
}

func messageHeader(notification Notification) (string, string) {
	switch notification.Kind {
	case KindAlert:
		return strings.ToUpper(notification.Resource) + " threshold breached", fmt.Sprintf("Usage reached %.2f%% and stayed above the %.2f%% threshold.", notification.Current, notification.Threshold)
	case KindFollowup:
		return strings.ToUpper(notification.Resource) + " usage increased", fmt.Sprintf("Usage climbed from %.2f%% to %.2f%% while still above the %.2f%% threshold.", notification.PreviousAlert, notification.Current, notification.Threshold)
	case KindRecovery:
		return strings.ToUpper(notification.Resource) + " recovered", fmt.Sprintf("Usage fell to %.2f%% and remained below the %.2f%% threshold.", notification.Current, notification.Threshold)
	case KindManual:
		return "Manual server snapshot", "Snapshot requested manually from the CLI."
	default:
		return "Scheduled server snapshot", "Snapshot requested by the configured cron schedule."
	}
}

func humanBytes(value uint64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	size := float64(value)
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", value, units[unit])
	}
	return fmt.Sprintf("%.2f %s", size, units[unit])
}

func publicIPText(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unavailable"
	}
	return value
}
