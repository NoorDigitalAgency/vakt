package monitor

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestCollectorCollectIncludesResolvedPublicIP(t *testing.T) {
	t.Parallel()

	collector := NewCollector("server-1", "/", time.Second)
	collector.publicIPResolver = func(context.Context, *http.Client) (string, error) {
		return "203.0.113.10", nil
	}

	snapshot, err := collector.Collect(time.Now())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if snapshot.Hostname != "server-1" {
		t.Fatalf("snapshot.Hostname = %q, want %q", snapshot.Hostname, "server-1")
	}
	if snapshot.PublicIP != "203.0.113.10" {
		t.Fatalf("snapshot.PublicIP = %q, want %q", snapshot.PublicIP, "203.0.113.10")
	}
}
