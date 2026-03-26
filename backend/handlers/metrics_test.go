package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"watchflare/backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupMetricsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/servers")
	group.Use(middleware.AuthMiddleware())
	{
		group.GET("/:id/metrics", GetMetrics)
		group.GET("/:id/metrics/containers", GetContainerMetrics)
		group.GET("/:id/metrics/sensors", GetSensorReadings)
	}
	return router
}

// --- resolveTimeRange ---

func TestResolveTimeRange_ValidRanges(t *testing.T) {
	cases := []struct {
		tr       string
		wantOK   bool
		interval string
	}{
		{"1h", true, ""},
		{"12h", true, "10m"},
		{"24h", true, "15m"},
		{"7d", true, "2h"},
		{"30d", true, "8h"},
		{"99y", false, ""},
		{"", false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.tr, func(t *testing.T) {
			_, _, interval, ok := resolveTimeRange(tc.tr)
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, tc.interval, interval)
			}
		})
	}
}

func TestResolveTimeRange_TimeWindow(t *testing.T) {
	before := time.Now()
	start, end, _, ok := resolveTimeRange("1h")
	after := time.Now()

	assert.True(t, ok)
	assert.True(t, end.After(before) || end.Equal(before))
	assert.True(t, end.Before(after) || end.Equal(after))
	assert.WithinDuration(t, end.Add(-time.Hour), start, time.Second)
}

// --- parseTime ---

func TestParseTime_RFC3339(t *testing.T) {
	input := "2024-01-15T10:30:00Z"
	t1, err := parseTime(input)
	assert.NoError(t, err)
	assert.Equal(t, "2024-01-15T10:30:00Z", t1.UTC().Format(time.RFC3339))
}

func TestParseTime_UnixTimestamp(t *testing.T) {
	t1, err := parseTime("1705312200")
	assert.NoError(t, err)
	assert.Equal(t, int64(1705312200), t1.Unix())
}

func TestParseTime_Invalid(t *testing.T) {
	_, err := parseTime("not-a-time")
	assert.Error(t, err)
}

// --- HTTP handlers (DB-dependent, skipped if unavailable) ---

func TestGetMetrics_InvalidTimeRange(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupMetricsRouter()
	cookie := createTestUser(t)

	req, _ := http.NewRequest("GET", "/servers/some-id/metrics?time_range=invalid", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetMetrics_StartAfterEnd(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupMetricsRouter()
	cookie := createTestUser(t)

	// start after end
	req, _ := http.NewRequest("GET", "/servers/some-id/metrics?start=2024-01-15T12:00:00Z&end=2024-01-15T10:00:00Z", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetMetrics_Unauthenticated(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupMetricsRouter()

	req, _ := http.NewRequest("GET", "/servers/some-id/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetContainerMetrics_InvalidTimeRange(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupMetricsRouter()
	cookie := createTestUser(t)

	req, _ := http.NewRequest("GET", "/servers/some-id/metrics/containers?time_range=bad", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetSensorReadings_InvalidTimeRange(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupMetricsRouter()
	cookie := createTestUser(t)

	req, _ := http.NewRequest("GET", "/servers/some-id/metrics/sensors?time_range=bad", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
