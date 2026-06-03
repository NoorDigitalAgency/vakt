package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Monitor       MonitorConfig       `yaml:"monitor"`
	Notifications NotificationsConfig `yaml:"notifications"`
	Thresholds    ThresholdsConfig    `yaml:"thresholds"`
}

type MonitorConfig struct {
	PollInterval     time.Duration `yaml:"poll_interval"`
	SnapshotSchedule string        `yaml:"snapshot_schedule"`
	StoragePath      string        `yaml:"storage_path"`
	ServerName       string        `yaml:"server_name"`
	HostAlias        string        `yaml:"host_alias"`
	HTTPTimeout      time.Duration `yaml:"http_timeout"`
}

type NotificationsConfig struct {
	Slack SlackConfig `yaml:"slack"`
}

type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
	Username   string `yaml:"username"`
	IconEmoji  string `yaml:"icon_emoji"`
}

type ThresholdsConfig struct {
	CPU     ResourceThreshold `yaml:"cpu"`
	Memory  ResourceThreshold `yaml:"memory"`
	Storage ResourceThreshold `yaml:"storage"`
}

type ResourceThreshold struct {
	Percent             float64       `yaml:"percent"`
	SustainFor          time.Duration `yaml:"sustain_for"`
	ClearFor            time.Duration `yaml:"clear_for"`
	FollowupStepPercent float64       `yaml:"followup_step_percent"`
	Cooldown            time.Duration `yaml:"cooldown"`
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	cfg := defaultConfig()
	decoder := yaml.NewDecoder(strings.NewReader(string(raw)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Monitor: MonitorConfig{
			PollInterval: 15 * time.Second,
			StoragePath:  "/",
			HTTPTimeout:  10 * time.Second,
		},
		Notifications: NotificationsConfig{
			Slack: SlackConfig{},
		},
		Thresholds: ThresholdsConfig{
			CPU: ResourceThreshold{
				SustainFor:          2 * time.Minute,
				ClearFor:            5 * time.Minute,
				FollowupStepPercent: 5,
				Cooldown:            15 * time.Minute,
			},
			Memory: ResourceThreshold{
				SustainFor:          2 * time.Minute,
				ClearFor:            5 * time.Minute,
				FollowupStepPercent: 5,
				Cooldown:            15 * time.Minute,
			},
			Storage: ResourceThreshold{
				SustainFor:          2 * time.Minute,
				ClearFor:            5 * time.Minute,
				FollowupStepPercent: 3,
				Cooldown:            30 * time.Minute,
			},
		},
	}
}

func (c Config) Validate() error {
	if c.Monitor.PollInterval <= 0 {
		return fmt.Errorf("monitor.poll_interval must be greater than zero")
	}
	if c.Monitor.StoragePath == "" {
		return fmt.Errorf("monitor.storage_path must not be empty")
	}
	if c.Monitor.HTTPTimeout <= 0 {
		return fmt.Errorf("monitor.http_timeout must be greater than zero")
	}
	if c.Monitor.SnapshotSchedule != "" {
		if _, err := cron.ParseStandard(c.Monitor.SnapshotSchedule); err != nil {
			return fmt.Errorf("monitor.snapshot_schedule: %w", err)
		}
	}

	if err := validateWebhook(c.Notifications.Slack.WebhookURL); err != nil {
		return err
	}

	if err := validateThreshold("thresholds.cpu", c.Thresholds.CPU); err != nil {
		return err
	}
	if err := validateThreshold("thresholds.memory", c.Thresholds.Memory); err != nil {
		return err
	}
	if err := validateThreshold("thresholds.storage", c.Thresholds.Storage); err != nil {
		return err
	}

	return nil
}

func validateWebhook(raw string) error {
	if raw == "" {
		return fmt.Errorf("notifications.slack.webhook_url must not be empty")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("notifications.slack.webhook_url: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("notifications.slack.webhook_url must use https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("notifications.slack.webhook_url must include a host")
	}
	return nil
}

func validateThreshold(name string, threshold ResourceThreshold) error {
	if threshold.Percent <= 0 || threshold.Percent >= 100 {
		return fmt.Errorf("%s.percent must be between 0 and 100", name)
	}
	if threshold.SustainFor <= 0 {
		return fmt.Errorf("%s.sustain_for must be greater than zero", name)
	}
	if threshold.ClearFor <= 0 {
		return fmt.Errorf("%s.clear_for must be greater than zero", name)
	}
	if threshold.FollowupStepPercent <= 0 {
		return fmt.Errorf("%s.followup_step_percent must be greater than zero", name)
	}
	if threshold.Cooldown <= 0 {
		return fmt.Errorf("%s.cooldown must be greater than zero", name)
	}
	return nil
}
