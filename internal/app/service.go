package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/NoorDigitalAgency/vakt/internal/config"
	"github.com/NoorDigitalAgency/vakt/internal/monitor"
	"github.com/NoorDigitalAgency/vakt/internal/notify"
)

type Service struct {
	cfg       config.Config
	collector *monitor.Collector
	notifier  *notify.SlackNotifier
	trackers  []*monitor.ThresholdTracker
}

func NewService(cfg config.Config) *Service {
	hostname := cfg.Monitor.ServerName
	if hostname == "" {
		hostname = cfg.Monitor.HostAlias
	}
	if hostname == "" {
		resolved, err := os.Hostname()
		if err != nil {
			hostname = "unknown-host"
		} else {
			hostname = resolved
		}
	}

	return &Service{
		cfg:       cfg,
		collector: monitor.NewCollector(hostname, cfg.Monitor.StoragePath, cfg.Monitor.HTTPTimeout),
		notifier:  notify.NewSlackNotifier(cfg),
		trackers: []*monitor.ThresholdTracker{
			monitor.NewThresholdTracker("cpu", cfg.Thresholds.CPU),
			monitor.NewThresholdTracker("memory", cfg.Thresholds.Memory),
			monitor.NewThresholdTracker("storage", cfg.Thresholds.Storage),
		},
	}
}

func (s *Service) Run(ctx context.Context) error {
	if s.cfg.Monitor.SnapshotSchedule != "" {
		scheduler := cron.New()
		if _, err := scheduler.AddFunc(s.cfg.Monitor.SnapshotSchedule, func() {
			snapshotCtx, cancel := context.WithTimeout(context.Background(), s.cfg.Monitor.HTTPTimeout)
			defer cancel()
			if err := s.sendSnapshot(snapshotCtx, notify.KindScheduled); err != nil {
				log.Printf("scheduled snapshot failed: %v", err)
			}
		}); err != nil {
			return fmt.Errorf("register snapshot schedule: %w", err)
		}
		scheduler.Start()
		defer func() {
			stopCtx := scheduler.Stop()
			<-stopCtx.Done()
		}()
	}

	if err := s.pollOnce(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(s.cfg.Monitor.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			if err := s.processSample(ctx, now); err != nil {
				log.Printf("resource polling failed: %v", err)
			}
		}
	}
}

func (s *Service) SendManualSnapshot(ctx context.Context) error {
	return s.sendSnapshot(ctx, notify.KindManual)
}

func (s *Service) pollOnce(ctx context.Context) error {
	return s.processSample(ctx, time.Now())
}

func (s *Service) processSample(ctx context.Context, now time.Time) error {
	snapshot, err := s.collector.Collect(now)
	if err != nil {
		return err
	}

	values := map[string]float64{
		"cpu":     snapshot.CPUUsagePercent,
		"memory":  snapshot.MemoryUsagePercent,
		"storage": snapshot.StorageUsagePercent,
	}

	for _, tracker := range s.trackers {
		resource := tracker.ResourceName()
		event := tracker.Evaluate(snapshot.Timestamp, values[resource])
		if event == nil {
			continue
		}

		notification := notify.Notification{
			Kind:          kindFromEvent(event.Kind),
			Resource:      event.Resource,
			Current:       event.Current,
			PreviousAlert: event.PreviousAlert,
			Threshold:     event.Threshold.Percent,
			Snapshot:      snapshot,
		}
		if err := s.notifier.Send(ctx, notification); err != nil {
			log.Printf("send %s notification failed: %v", event.Resource, err)
		}
	}

	return nil
}

func (s *Service) sendSnapshot(ctx context.Context, kind notify.Kind) error {
	snapshot, err := s.collector.Collect(time.Now())
	if err != nil {
		return err
	}
	return s.notifier.Send(ctx, notify.Notification{Kind: kind, Snapshot: snapshot})
}

func kindFromEvent(kind monitor.EventKind) notify.Kind {
	switch kind {
	case monitor.EventAlert:
		return notify.KindAlert
	case monitor.EventFollowup:
		return notify.KindFollowup
	case monitor.EventReminder:
		return notify.KindReminder
	default:
		return notify.KindRecovery
	}
}
