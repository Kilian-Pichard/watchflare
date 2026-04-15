package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"watchflare/backend/middleware"
	"watchflare/backend/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupPackagesRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/hosts")
	group.Use(middleware.AuthMiddleware())
	{
		group.GET("/:id/packages", GetHostPackages)
		group.GET("/:id/packages/history", GetHostPackageHistory)
		group.GET("/:id/packages/collections", GetHostPackageCollections)
		group.GET("/:id/packages/stats", GetPackageStats)
	}
	return router
}

func TestGetHostPackages_Unauthenticated(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	req, _ := http.NewRequest("GET", "/hosts/some-id/packages", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetHostPackages_ReturnsEmptyList(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	cookie := createTestUser(t)

	host, _, _, _ := services.CreateAgent("pkg-host", "10.0.0.1", false)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/hosts/%s/packages", host.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotNil(t, resp["packages"])
	assert.Equal(t, float64(0), resp["total_count"])
}

func TestGetHostPackages_LimitClamped(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	cookie := createTestUser(t)

	host, _, _, _ := services.CreateAgent("pkg-host", "10.0.0.1", false)

	// limit=0 → clamped to 1000
	req, _ := http.NewRequest("GET", fmt.Sprintf("/hosts/%s/packages?limit=0", host.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(1000), resp["limit"])
}

func TestGetHostPackageHistory_InvalidChangeType(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	cookie := createTestUser(t)

	host, _, _, _ := services.CreateAgent("pkg-host", "10.0.0.1", false)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/hosts/%s/packages/history?change_type=invalid", host.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "invalid change_type")
}

func TestGetHostPackageHistory_ValidChangeTypes(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	cookie := createTestUser(t)

	host, _, _, _ := services.CreateAgent("pkg-host", "10.0.0.1", false)

	for _, ct := range []string{"added", "removed", "updated", "initial"} {
		t.Run(ct, func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("/hosts/%s/packages/history?change_type=%s", host.ID, ct), nil)
			req.AddCookie(cookie)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestGetHostPackageCollections_OffsetClamped(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	cookie := createTestUser(t)

	host, _, _, _ := services.CreateAgent("pkg-host", "10.0.0.1", false)

	// offset=-5 should not cause an error
	req, _ := http.NewRequest("GET", fmt.Sprintf("/hosts/%s/packages/collections?offset=-5", host.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["offset"])
}

func TestGetPackageStats_ReturnsStats(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	cookie := createTestUser(t)

	host, _, _, _ := services.CreateAgent("pkg-host", "10.0.0.1", false)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/hosts/%s/packages/stats", host.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotNil(t, resp["total_packages"])
	assert.NotNil(t, resp["by_package_manager"])
	assert.NotNil(t, resp["recent_changes"])
	assert.NotNil(t, resp["outdated_count"])
	assert.NotNil(t, resp["security_updates_count"])
}
