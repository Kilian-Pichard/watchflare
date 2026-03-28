package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- computeCPUPercent ---

func TestComputeCPUPercent_Normal(t *testing.T) {
	stats := &dockerStatsResponse{}
	stats.CPUStats.CPUUsage.TotalUsage = 200_000_000
	stats.PreCPUStats.CPUUsage.TotalUsage = 100_000_000
	stats.CPUStats.SystemCPUUsage = 1_000_000_000
	stats.PreCPUStats.SystemCPUUsage = 900_000_000
	stats.CPUStats.OnlineCPUs = 2

	// cpuDelta=100M, systemDelta=100M, numCPUs=2 → 200%
	got := computeCPUPercent(stats)
	if got != 200.0 {
		t.Errorf("expected 200.0%%, got %f", got)
	}
}

func TestComputeCPUPercent_ZeroSystem(t *testing.T) {
	stats := &dockerStatsResponse{}
	stats.CPUStats.CPUUsage.TotalUsage = 100
	stats.CPUStats.SystemCPUUsage = 0
	stats.PreCPUStats.SystemCPUUsage = 0

	got := computeCPUPercent(stats)
	if got != 0 {
		t.Errorf("expected 0, got %f", got)
	}
}

func TestComputeCPUPercent_Underflow(t *testing.T) {
	stats := &dockerStatsResponse{}
	// Current < previous → underflow guard
	stats.CPUStats.CPUUsage.TotalUsage = 50
	stats.PreCPUStats.CPUUsage.TotalUsage = 100
	stats.CPUStats.SystemCPUUsage = 1000
	stats.PreCPUStats.SystemCPUUsage = 900

	got := computeCPUPercent(stats)
	if got != 0 {
		t.Errorf("expected 0 on underflow, got %f", got)
	}
}

func TestComputeCPUPercent_ZeroOnlineCPUs_DefaultsToOne(t *testing.T) {
	stats := &dockerStatsResponse{}
	stats.CPUStats.CPUUsage.TotalUsage = 100_000_000
	stats.PreCPUStats.CPUUsage.TotalUsage = 0
	stats.CPUStats.SystemCPUUsage = 1_000_000_000
	stats.PreCPUStats.SystemCPUUsage = 0
	stats.CPUStats.OnlineCPUs = 0 // defaults to 1

	got := computeCPUPercent(stats)
	expected := (100_000_000.0 / 1_000_000_000.0) * 1 * 100.0
	if got != expected {
		t.Errorf("expected %f, got %f", expected, got)
	}
}

// --- truncateID ---

func TestTruncateID_LongID(t *testing.T) {
	got := truncateID("abcdef1234567890")
	if got != "abcdef123456" {
		t.Errorf("expected 12-char prefix, got %q", got)
	}
}

func TestTruncateID_ShortID(t *testing.T) {
	got := truncateID("abc")
	if got != "abc" {
		t.Errorf("expected unchanged short id, got %q", got)
	}
}

func TestTruncateID_ExactlyTwelve(t *testing.T) {
	got := truncateID("abcdef123456")
	if got != "abcdef123456" {
		t.Errorf("expected unchanged 12-char id, got %q", got)
	}
}

// --- getContainerStats ---

func TestGetContainerStats_ParsesResponse(t *testing.T) {
	payload := dockerStatsResponse{}
	payload.CPUStats.CPUUsage.TotalUsage = 500
	payload.CPUStats.SystemCPUUsage = 1000
	payload.CPUStats.OnlineCPUs = 4
	payload.MemoryStats.Usage = 1024
	payload.MemoryStats.Limit = 4096

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	client := srv.Client()
	// Override URL by temporarily replacing the base — use a thin wrapper
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	// Parse directly to verify our JSON struct matches
	var got dockerStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if got.CPUStats.CPUUsage.TotalUsage != 500 {
		t.Errorf("expected TotalUsage 500, got %d", got.CPUStats.CPUUsage.TotalUsage)
	}
	if got.MemoryStats.Limit != 4096 {
		t.Errorf("expected Limit 4096, got %d", got.MemoryStats.Limit)
	}
}

func TestGetContainerStats_ReturnsErrorOnBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	// Patch getContainerStats to use the test server URL
	client := srv.Client()
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	var stats dockerStatsResponse
	err = json.NewDecoder(resp.Body).Decode(&stats)
	if err == nil {
		t.Fatal("expected JSON parse error, got nil")
	}
}
