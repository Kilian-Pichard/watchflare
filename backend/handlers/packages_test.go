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
	group := router.Group("/servers")
	group.Use(middleware.AuthMiddleware())
	{
		group.GET("/:id/packages", GetServerPackages)
		group.GET("/:id/packages/history", GetServerPackageHistory)
		group.GET("/:id/packages/collections", GetServerPackageCollections)
		group.GET("/:id/packages/stats", GetPackageStats)
	}
	return router
}

func TestGetServerPackages_Unauthenticated(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	req, _ := http.NewRequest("GET", "/servers/some-id/packages", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetServerPackages_ReturnsEmptyList(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	cookie := createTestUser(t)

	server, _, _, _ := services.CreateAgent("pkg-server", "10.0.0.1", false)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/servers/%s/packages", server.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotNil(t, resp["packages"])
	assert.Equal(t, float64(0), resp["total_count"])
}

func TestGetServerPackages_LimitClamped(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	cookie := createTestUser(t)

	server, _, _, _ := services.CreateAgent("pkg-server", "10.0.0.1", false)

	// limit=0 → clamped to 1000
	req, _ := http.NewRequest("GET", fmt.Sprintf("/servers/%s/packages?limit=0", server.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(1000), resp["limit"])
}

func TestGetServerPackageHistory_InvalidChangeType(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	cookie := createTestUser(t)

	server, _, _, _ := services.CreateAgent("pkg-server", "10.0.0.1", false)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/servers/%s/packages/history?change_type=invalid", server.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "invalid change_type")
}

func TestGetServerPackageHistory_ValidChangeTypes(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	cookie := createTestUser(t)

	server, _, _, _ := services.CreateAgent("pkg-server", "10.0.0.1", false)

	for _, ct := range []string{"added", "removed", "updated", "initial"} {
		t.Run(ct, func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("/servers/%s/packages/history?change_type=%s", server.ID, ct), nil)
			req.AddCookie(cookie)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestGetServerPackageCollections_OffsetClamped(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	cookie := createTestUser(t)

	server, _, _, _ := services.CreateAgent("pkg-server", "10.0.0.1", false)

	// offset=-5 should not cause an error
	req, _ := http.NewRequest("GET", fmt.Sprintf("/servers/%s/packages/collections?offset=-5", server.ID), nil)
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

	server, _, _, _ := services.CreateAgent("pkg-server", "10.0.0.1", false)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/servers/%s/packages/stats", server.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotNil(t, resp["total_packages"])
	assert.NotNil(t, resp["by_package_manager"])
	assert.NotNil(t, resp["recent_changes"])
}
