package services

import (
	"testing"
)

func TestGetContinuousAggregateTable(t *testing.T) {
	cases := []struct {
		interval string
		want     string
	}{
		{"10m", "metrics_10min"},
		{"15m", "metrics_15min"},
		{"2h", "metrics_2h"},
		{"8h", "metrics_8h"},
	}

	for _, tc := range cases {
		t.Run(tc.interval, func(t *testing.T) {
			got, err := getContinuousAggregateTable(tc.interval)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestGetContinuousAggregateTable_InvalidInterval(t *testing.T) {
	_, err := getContinuousAggregateTable("5m")
	if err == nil {
		t.Error("expected error for invalid interval")
	}
}

func TestGetContainerAggregateTable(t *testing.T) {
	cases := []struct {
		interval string
		want     string
	}{
		{"10m", "container_metrics_10min"},
		{"15m", "container_metrics_15min"},
		{"2h", "container_metrics_2h"},
		{"8h", "container_metrics_8h"},
	}

	for _, tc := range cases {
		t.Run(tc.interval, func(t *testing.T) {
			got, err := getContainerAggregateTable(tc.interval)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestGetContainerAggregateTable_InvalidInterval(t *testing.T) {
	_, err := getContainerAggregateTable("1h")
	if err == nil {
		t.Error("expected error for invalid interval")
	}
}
