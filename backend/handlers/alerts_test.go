package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"watchflare/backend/database"
	"watchflare/backend/middleware"
	"watchflare/backend/models"
	"watchflare/backend/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAlertsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	protected := r.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/settings/alerts", GetAlertRules)
		protected.PUT("/settings/alerts", UpdateAlertRules)
		protected.GET("/settings/alerts/active", GetActiveIncidents)
		protected.GET("/servers/:id/alerts", GetServerAlertRules)
		protected.PUT("/servers/:id/alerts/:metric_type", UpsertServerAlertRule)
		protected.DELETE("/servers/:id/alerts/:metric_type", DeleteServerAlertRule)
		protected.GET("/servers/:id/incidents", GetServerIncidents)
	}
	return r
}

func teardownAlertData() {
	database.DB.Exec("DELETE FROM alert_incidents")
	database.DB.Exec("DELETE FROM server_alert_rules")
	database.DB.Exec("DELETE FROM alert_rules")
	database.DB.Exec("DELETE FROM servers")
	database.DB.Exec("DELETE FROM users")
}

func seedGlobalAlertRules(t *testing.T) {
	t.Helper()
	require.NoError(t, services.UpdateAlertRules([]services.AlertRuleInput{
		{MetricType: models.MetricTypeCPUUsage, Enabled: true, Threshold: 80.0, DurationMinutes: 5},
		{MetricType: models.MetricTypeMemoryUsage, Enabled: false, Threshold: 90.0, DurationMinutes: 10},
	}))
}

func TestGetAlertRules(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	cookie := registerAndGetCookie(t, "alerts@test.com")

	seedGlobalAlertRules(t)

	req, _ := http.NewRequest("GET", "/settings/alerts", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	rules, ok := resp["rules"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(rules), 2)
}

func TestGetAlertRules_Unauthenticated(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	req, _ := http.NewRequest("GET", "/settings/alerts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateAlertRules(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	cookie := registerAndGetCookie(t, "alerts2@test.com")

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Success - valid rules",
			payload: map[string]interface{}{
				"rules": []map[string]interface{}{
					{"metric_type": models.MetricTypeCPUUsage, "enabled": true, "threshold": 85.0, "duration_minutes": 5},
					{"metric_type": models.MetricTypeMemoryUsage, "enabled": false, "threshold": 90.0, "duration_minutes": 10},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Fail - invalid metric_type",
			payload: map[string]interface{}{
				"rules": []map[string]interface{}{
					{"metric_type": "invalid_metric", "enabled": true, "threshold": 80.0, "duration_minutes": 5},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Fail - duration_minutes less than 1",
			payload: map[string]interface{}{
				"rules": []map[string]interface{}{
					{"metric_type": models.MetricTypeCPUUsage, "enabled": true, "threshold": 80.0, "duration_minutes": 0},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest("PUT", "/settings/alerts", bytes.NewBuffer(b))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(cookie)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestUpdateAlertRules_Unauthenticated(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	b, _ := json.Marshal(map[string]interface{}{"rules": []interface{}{}})
	req, _ := http.NewRequest("PUT", "/settings/alerts", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetServerAlertRules(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	cookie := registerAndGetCookie(t, "alerts3@test.com")

	seedGlobalAlertRules(t)

	req, _ := http.NewRequest("GET", "/servers/server-abc/alerts", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	rules, ok := resp["rules"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(rules), 2)
}

func TestUpsertServerAlertRule(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	cookie := registerAndGetCookie(t, "alerts4@test.com")

	seedGlobalAlertRules(t)

	tests := []struct {
		name           string
		serverID       string
		metricType     string
		payload        map[string]interface{}
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:       "Success - valid upsert",
			serverID:   "server-abc",
			metricType: models.MetricTypeCPUUsage,
			payload:    map[string]interface{}{"enabled": true, "threshold": 95.0, "duration_minutes": 3},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "server alert rule saved", resp["message"])
			},
		},
		{
			name:       "Fail - invalid metric_type",
			serverID:   "server-abc",
			metricType: "not_a_real_metric",
			payload:    map[string]interface{}{"enabled": true, "threshold": 95.0, "duration_minutes": 3},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
		{
			name:       "Fail - duration_minutes less than 1",
			serverID:   "server-abc",
			metricType: models.MetricTypeCPUUsage,
			payload:    map[string]interface{}{"enabled": true, "threshold": 95.0, "duration_minutes": 0},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(tt.payload)
			url := "/servers/" + tt.serverID + "/alerts/" + tt.metricType
			req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(b))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(cookie)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestDeleteServerAlertRule(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	cookie := registerAndGetCookie(t, "alerts5@test.com")

	seedGlobalAlertRules(t)

	// Create an override first.
	b, _ := json.Marshal(map[string]interface{}{"enabled": true, "threshold": 75.0, "duration_minutes": 2})
	req, _ := http.NewRequest("PUT", "/servers/server-del/alerts/"+models.MetricTypeCPUUsage, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Delete it.
	req, _ = http.NewRequest("DELETE", "/servers/server-del/alerts/"+models.MetricTypeCPUUsage, nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "server alert rule deleted", resp["message"])
}

func TestDeleteServerAlertRule_InvalidMetricType(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	cookie := registerAndGetCookie(t, "alerts6@test.com")

	req, _ := http.NewRequest("DELETE", "/servers/server-abc/alerts/not_a_real_metric", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteServerAlertRule_Unauthenticated(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	req, _ := http.NewRequest("DELETE", "/servers/server-abc/alerts/"+models.MetricTypeCPUUsage, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetActiveIncidents(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	cookie := registerAndGetCookie(t, "incidents@test.com")

	// Seed a server and an active incident
	server := models.Server{ID: "server-inc-1", Name: "test-server", Status: "offline"}
	require.NoError(t, database.DB.Create(&server).Error)

	incident := models.AlertIncident{
		ServerID:       server.ID,
		MetricType:     models.MetricTypeServerDown,
		StartedAt:      time.Now().Add(-5 * time.Minute),
		ThresholdValue: 0,
		CurrentValue:   0,
	}
	require.NoError(t, database.DB.Create(&incident).Error)

	req, _ := http.NewRequest("GET", "/settings/alerts/active", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	incidents, ok := resp["incidents"].([]interface{})
	require.True(t, ok)
	require.Len(t, incidents, 1)

	item := incidents[0].(map[string]interface{})
	assert.Equal(t, server.ID, item["server_id"])
	assert.Equal(t, "test-server", item["server_name"])
	assert.Equal(t, models.MetricTypeServerDown, item["metric_type"])
}

func TestGetActiveIncidents_Empty(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	cookie := registerAndGetCookie(t, "incidents2@test.com")

	req, _ := http.NewRequest("GET", "/settings/alerts/active", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	incidents, ok := resp["incidents"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, incidents)
}

func TestGetServerIncidents(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	cookie := registerAndGetCookie(t, "sinc@test.com")

	server := models.Server{ID: "server-sinc", Name: "sinc-server", Status: "offline"}
	require.NoError(t, database.DB.Create(&server).Error)

	// Active incident
	active := models.AlertIncident{
		ServerID:   server.ID,
		MetricType: models.MetricTypeServerDown,
		StartedAt:  time.Now().Add(-5 * time.Minute),
	}
	require.NoError(t, database.DB.Create(&active).Error)

	// Resolved incident
	resolved := models.AlertIncident{
		ServerID:   server.ID,
		MetricType: models.MetricTypeCPUUsage,
		StartedAt:  time.Now().Add(-30 * time.Minute),
	}
	require.NoError(t, database.DB.Create(&resolved).Error)
	resolvedAt := time.Now().Add(-20 * time.Minute)
	require.NoError(t, database.DB.Model(&resolved).Update("resolved_at", resolvedAt).Error)

	req, _ := http.NewRequest("GET", "/servers/server-sinc/incidents", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	incidents, ok := resp["incidents"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), resp["total_count"])
	assert.Len(t, incidents, 2)
}

func TestGetServerIncidents_StatusFilter(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	cookie := registerAndGetCookie(t, "sinc2@test.com")

	server := models.Server{ID: "server-sinc2", Name: "sinc2-server", Status: "offline"}
	require.NoError(t, database.DB.Create(&server).Error)

	active := models.AlertIncident{
		ServerID:   server.ID,
		MetricType: models.MetricTypeServerDown,
		StartedAt:  time.Now().Add(-5 * time.Minute),
	}
	require.NoError(t, database.DB.Create(&active).Error)

	resolved := models.AlertIncident{
		ServerID:   server.ID,
		MetricType: models.MetricTypeCPUUsage,
		StartedAt:  time.Now().Add(-30 * time.Minute),
	}
	require.NoError(t, database.DB.Create(&resolved).Error)
	require.NoError(t, database.DB.Model(&resolved).Update("resolved_at", time.Now()).Error)

	// Filter: active only
	req, _ := http.NewRequest("GET", "/servers/server-sinc2/incidents?status=active", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total_count"])

	// Filter: resolved only
	req, _ = http.NewRequest("GET", "/servers/server-sinc2/incidents?status=resolved", nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total_count"])
}

func TestGetServerIncidents_Empty(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	cookie := registerAndGetCookie(t, "sinc3@test.com")

	req, _ := http.NewRequest("GET", "/servers/no-such-server/incidents", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	incidents, ok := resp["incidents"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, incidents)
	assert.Equal(t, float64(0), resp["total_count"])
}

func TestGetServerIncidents_Unauthenticated(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	req, _ := http.NewRequest("GET", "/servers/server-abc/incidents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetActiveIncidents_Unauthenticated(t *testing.T) {
	setupTestDB(t)
	defer teardownAlertData()

	r := setupAlertsRouter()
	req, _ := http.NewRequest("GET", "/settings/alerts/active", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
