package services

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildAggregatedQuery_ContainsTableAndInterval(t *testing.T) {
	cases := []struct {
		table    string
		interval string
	}{
		{"metrics_10min", "10 minutes"},
		{"metrics_15min", "15 minutes"},
		{"metrics_2h", "2 hours"},
		{"metrics_8h", "8 hours"},
	}

	for _, tc := range cases {
		t.Run(tc.table, func(t *testing.T) {
			q := buildAggregatedQuery(tc.table, tc.interval)
			if !strings.Contains(q, tc.table) {
				t.Errorf("query does not contain table %q", tc.table)
			}
			if !strings.Contains(q, tc.interval) {
				t.Errorf("query does not contain interval %q", tc.interval)
			}
		})
	}
}

func TestBuildAggregatedQuery_HasRequiredClauses(t *testing.T) {
	q := buildAggregatedQuery("metrics_10min", "10 minutes")
	for _, clause := range []string{"WITH", "UNION ALL", "GROUP BY", "ORDER BY"} {
		if !strings.Contains(q, clause) {
			t.Errorf("query missing expected clause %q", clause)
		}
	}
}

func TestGetAggregatedMetrics_InvalidTimeRange(t *testing.T) {
	_, err := GetAggregatedMetrics("99y")
	if !errors.Is(err, ErrInvalidTimeRange) {
		t.Errorf("expected ErrInvalidTimeRange, got %v", err)
	}
}

func TestAggregateConfigs_KnownTimeRanges(t *testing.T) {
	// Verify that all expected time ranges are present in aggregateConfigs.
	for _, tr := range []string{"12h", "24h", "7d", "30d"} {
		if _, ok := aggregateConfigs[tr]; !ok {
			t.Errorf("aggregateConfigs missing expected time range %q", tr)
		}
	}
}
