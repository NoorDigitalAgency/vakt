package monitor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Snapshot struct {
	Timestamp           time.Time
	Hostname            string
	PublicIP            string
	CPUUsagePercent     float64
	MemoryUsagePercent  float64
	MemoryUsedBytes     uint64
	MemoryTotalBytes    uint64
	StorageUsagePercent float64
	StorageUsedBytes    uint64
	StorageTotalBytes   uint64
	Load1               float64
	Load5               float64
	Load15              float64
	Uptime              time.Duration
	StoragePath         string
}

type Collector struct {
	hostname string
	path     string

	httpClient       *http.Client
	publicIPResolver func(context.Context, *http.Client) (string, error)

	mu              sync.Mutex
	prevCPU         *cpuSample
	publicIP        string
	publicIPChecked time.Time
}

type cpuSample struct {
	idle  uint64
	total uint64
}

const (
	publicIPLookupURL       = "https://api.ipify.org"
	publicIPRefreshInterval = 15 * time.Minute
)

func NewCollector(hostname, path string, httpTimeout time.Duration) *Collector {
	client := &http.Client{Timeout: httpTimeout}
	return &Collector{
		hostname:         hostname,
		path:             path,
		httpClient:       client,
		publicIPResolver: fetchPublicIP,
	}
}

func (c *Collector) Collect(now time.Time) (Snapshot, error) {
	c.mu.Lock()
	needsRefresh := c.publicIPChecked.IsZero() || now.Sub(c.publicIPChecked) >= publicIPRefreshInterval
	c.mu.Unlock()

	var freshIP string
	if needsRefresh {
		if ip, err := c.publicIPResolver(context.Background(), c.httpClient); err == nil {
			freshIP = ip
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if needsRefresh {
		if freshIP != "" {
			c.publicIP = freshIP
		}
		c.publicIPChecked = now
	}
	snapshot := Snapshot{Timestamp: now, Hostname: c.hostname, StoragePath: c.path, PublicIP: c.publicIP}

	cpuUsage, nextCPU, err := readCPUUsage(c.prevCPU)
	if err != nil {
		return Snapshot{}, err
	}
	c.prevCPU = nextCPU
	snapshot.CPUUsagePercent = cpuUsage

	memUsed, memTotal, memUsage, err := readMemory()
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.MemoryUsedBytes = memUsed
	snapshot.MemoryTotalBytes = memTotal
	snapshot.MemoryUsagePercent = memUsage

	storageUsed, storageTotal, storageUsage, err := readStorage(c.path)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.StorageUsedBytes = storageUsed
	snapshot.StorageTotalBytes = storageTotal
	snapshot.StorageUsagePercent = storageUsage

	load1, load5, load15, err := readLoad()
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.Load1 = load1
	snapshot.Load5 = load5
	snapshot.Load15 = load15

	uptime, err := readUptime()
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.Uptime = uptime

	return snapshot, nil
}

func readCPUUsage(prev *cpuSample) (float64, *cpuSample, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, nil, fmt.Errorf("open /proc/stat: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return 0, nil, fmt.Errorf("scan /proc/stat: %w", err)
		}
		return 0, nil, fmt.Errorf("read /proc/stat: missing cpu line")
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 8 || fields[0] != "cpu" {
		return 0, nil, fmt.Errorf("read /proc/stat: malformed cpu line")
	}

	values := make([]uint64, 0, len(fields)-1)
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, nil, fmt.Errorf("parse cpu stat: %w", err)
		}
		values = append(values, value)
	}

	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}

	var total uint64
	for _, value := range values {
		total += value
	}

	current := &cpuSample{idle: idle, total: total}
	if prev == nil || total <= prev.total || idle < prev.idle {
		return 0, current, nil
	}

	totalDelta := float64(total - prev.total)
	idleDelta := float64(idle - prev.idle)
	if totalDelta == 0 {
		return 0, current, nil
	}

	usage := 100 * (totalDelta - idleDelta) / totalDelta
	return clampPercent(usage), current, nil
}

func readMemory() (uint64, uint64, float64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("open /proc/meminfo: %w", err)
	}
	defer file.Close()

	var totalKB uint64
	var availableKB uint64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			totalKB, err = strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, 0, 0, fmt.Errorf("parse MemTotal: %w", err)
			}
		case "MemAvailable:":
			availableKB, err = strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, 0, 0, fmt.Errorf("parse MemAvailable: %w", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, 0, fmt.Errorf("scan /proc/meminfo: %w", err)
	}
	if totalKB == 0 {
		return 0, 0, 0, fmt.Errorf("read /proc/meminfo: MemTotal missing")
	}

	usedBytes := (totalKB - availableKB) * 1024
	totalBytes := totalKB * 1024
	usage := 100 * float64(totalKB-availableKB) / float64(totalKB)
	return usedBytes, totalBytes, clampPercent(usage), nil
}

func readStorage(path string) (uint64, uint64, float64, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return 0, 0, 0, fmt.Errorf("statfs %s: %w", path, err)
	}

	total := stats.Blocks * uint64(stats.Bsize)
	free := stats.Bfree * uint64(stats.Bsize)
	used := total - free
	if total == 0 {
		return 0, 0, 0, fmt.Errorf("statfs %s: reported zero size", path)
	}

	usage := 100 * float64(used) / float64(total)
	return used, total, clampPercent(usage), nil
}

func readLoad() (float64, float64, float64, error) {
	raw, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("read /proc/loadavg: %w", err)
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 3 {
		return 0, 0, 0, fmt.Errorf("read /proc/loadavg: malformed data")
	}

	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parse load1: %w", err)
	}
	load5, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parse load5: %w", err)
	}
	load15, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parse load15: %w", err)
	}

	return load1, load5, load15, nil
}

func readUptime() (time.Duration, error) {
	raw, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, fmt.Errorf("read /proc/uptime: %w", err)
	}
	fields := strings.Fields(string(raw))
	if len(fields) == 0 {
		return 0, fmt.Errorf("read /proc/uptime: malformed data")
	}

	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("parse uptime: %w", err)
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

func clampPercent(value float64) float64 {
	return math.Max(0, math.Min(100, value))
}

func fetchPublicIP(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, publicIPLookupURL, nil)
	if err != nil {
		return "", fmt.Errorf("create public ip request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("lookup public ip: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("lookup public ip: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", fmt.Errorf("read public ip response: %w", err)
	}

	publicIP := strings.TrimSpace(string(body))
	if parsed := net.ParseIP(publicIP); parsed == nil {
		return "", fmt.Errorf("lookup public ip: invalid response %q", publicIP)
	}

	return publicIP, nil
}
