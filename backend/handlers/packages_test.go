package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"watchflare/backend/database"
	"watchflare/backend/middleware"
	"watchflare/backend/models"
	"watchflare/backend/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func setupGlobalPackagesRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/packages")
	group.Use(middleware.AuthMiddleware())
	group.GET("", ListAllPackages)
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
	pagination := resp["pagination"].(map[string]interface{})
	assert.Equal(t, float64(0), pagination["total"])
}

func TestGetHostPackages_LimitClamped(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupPackagesRouter()
	cookie := createTestUser(t)

	host, _, _, _ := services.CreateAgent("pkg-host", "10.0.0.1", false)

	// limit=0 → clamped to default 25
	req, _ := http.NewRequest("GET", fmt.Sprintf("/hosts/%s/packages?limit=0", host.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	pagination := resp["pagination"].(map[string]interface{})
	assert.Equal(t, float64(25), pagination["limit"])
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
	pagination := resp["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pagination["page"])
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

// ===== ListAllPackages tests =====

func TestListAllPackages_Unauthenticated(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupGlobalPackagesRouter()
	req, _ := http.NewRequest("GET", "/packages", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListAllPackages_Empty(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupGlobalPackagesRouter()
	cookie := createTestUser(t)

	req, _ := http.NewRequest("GET", "/packages", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["total_packages"])
	assert.Equal(t, float64(0), resp["outdated_count"])
	assert.Equal(t, float64(0), resp["security_count"])
	pagination := resp["pagination"].(map[string]interface{})
	assert.Equal(t, float64(0), pagination["total"])
	packages := resp["packages"].([]interface{})
	assert.Len(t, packages, 0)
}

func TestListAllPackages_DeduplicatesAcrossHosts(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	host1, _, _, _ := services.CreateAgent("host-a", "10.0.0.1", false)
	host2, _, _, _ := services.CreateAgent("host-b", "10.0.0.2", false)

	// curl installed on both hosts
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host1.ID, Name: "curl", Version: "7.88.0", PackageManager: "dpkg",
	}).Error)
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host2.ID, Name: "curl", Version: "7.88.0", PackageManager: "dpkg",
	}).Error)
	// bash only on host1
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host1.ID, Name: "bash", Version: "5.2.0", PackageManager: "dpkg",
	}).Error)

	router := setupGlobalPackagesRouter()
	cookie := createTestUser(t)

	req, _ := http.NewRequest("GET", "/packages", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	// 2 distinct (name, package_manager) pairs
	pagination := resp["pagination"].(map[string]interface{})
	assert.Equal(t, float64(2), pagination["total"])
	packages := resp["packages"].([]interface{})
	assert.Len(t, packages, 2)

	// Find curl and check host_count = 2
	var curlPkg map[string]interface{}
	for _, p := range packages {
		pkg := p.(map[string]interface{})
		if pkg["name"] == "curl" {
			curlPkg = pkg
			break
		}
	}
	require.NotNil(t, curlPkg)
	assert.Equal(t, float64(2), curlPkg["host_count"])
}

func TestListAllPackages_StatusFilter(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	host, _, _, _ := services.CreateAgent("host-status", "10.0.0.1", false)

	// Security update
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host.ID, Name: "openssl", Version: "3.0.0", PackageManager: "dpkg",
		AvailableVersion: "3.0.1", HasSecurityUpdate: true, UpdateChecked: true,
	}).Error)
	// Outdated (no security)
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host.ID, Name: "curl", Version: "7.88.0", PackageManager: "dpkg",
		AvailableVersion: "8.0.0", HasSecurityUpdate: false, UpdateChecked: true,
	}).Error)
	// Up to date
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host.ID, Name: "bash", Version: "5.2.0", PackageManager: "dpkg",
		AvailableVersion: "", HasSecurityUpdate: false, UpdateChecked: true,
	}).Error)
	// Not checked
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host.ID, Name: "myapp", Version: "1.0.0", PackageManager: "cargo",
		AvailableVersion: "", HasSecurityUpdate: false, UpdateChecked: false,
	}).Error)

	router := setupGlobalPackagesRouter()
	cookie := createTestUser(t)

	tests := []struct {
		status        string
		expectedCount float64
		expectedName  string
	}{
		{"security", 1, "openssl"},
		{"outdated", 1, "curl"},
		{"up_to_date", 1, "bash"},
		{"not_checked", 1, "myapp"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("/packages?status=%s", tt.status), nil)
			req.AddCookie(cookie)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			var resp map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			pagination := resp["pagination"].(map[string]interface{})
			assert.Equal(t, tt.expectedCount, pagination["total"], "status=%s", tt.status)
			// Global stats should always show all 4
			assert.Equal(t, float64(4), resp["total_packages"])
		})
	}
}

func TestListAllPackages_SearchFilter(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	host, _, _, _ := services.CreateAgent("host-search", "10.0.0.1", false)
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host.ID, Name: "curl", Version: "7.88.0", PackageManager: "dpkg",
	}).Error)
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host.ID, Name: "libcurl4", Version: "7.88.0", PackageManager: "dpkg",
	}).Error)
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host.ID, Name: "bash", Version: "5.2.0", PackageManager: "dpkg",
	}).Error)

	router := setupGlobalPackagesRouter()
	cookie := createTestUser(t)

	req, _ := http.NewRequest("GET", "/packages?q=curl", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	pagination := resp["pagination"].(map[string]interface{})
	assert.Equal(t, float64(2), pagination["total"])     // curl + libcurl4
	assert.Equal(t, float64(3), resp["total_packages"]) // global stat, unfiltered
}

func TestListAllPackages_ManagerFilter(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	host, _, _, _ := services.CreateAgent("host-mgr", "10.0.0.1", false)
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host.ID, Name: "curl", Version: "7.88.0", PackageManager: "dpkg",
	}).Error)
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host.ID, Name: "requests", Version: "2.31.0", PackageManager: "pip",
	}).Error)

	router := setupGlobalPackagesRouter()
	cookie := createTestUser(t)

	req, _ := http.NewRequest("GET", "/packages?manager=dpkg", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	pagination := resp["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pagination["total"])
	packages := resp["packages"].([]interface{})
	assert.Equal(t, "curl", packages[0].(map[string]interface{})["name"])
}

func TestListAllPackages_Pagination(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	host, _, _, _ := services.CreateAgent("host-page", "10.0.0.1", false)
	for _, name := range []string{"aaa", "bbb", "ccc", "ddd", "eee"} {
		require.NoError(t, database.DB.Create(&models.Package{
			HostID: host.ID, Name: name, Version: "1.0.0", PackageManager: "dpkg",
		}).Error)
	}

	router := setupGlobalPackagesRouter()
	cookie := createTestUser(t)

	req, _ := http.NewRequest("GET", "/packages?limit=2&offset=2&sort_by=name&sort_order=asc", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	pagination := resp["pagination"].(map[string]interface{})
	assert.Equal(t, float64(5), pagination["total"])
	packages := resp["packages"].([]interface{})
	assert.Len(t, packages, 2)
	assert.Equal(t, "ccc", packages[0].(map[string]interface{})["name"])
	assert.Equal(t, "ddd", packages[1].(map[string]interface{})["name"])
}

func TestListAllPackages_GlobalStatsUnaffectedByFilter(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	host, _, _, _ := services.CreateAgent("host-stats", "10.0.0.1", false)
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host.ID, Name: "openssl", Version: "3.0.0", PackageManager: "dpkg",
		AvailableVersion: "3.0.1", HasSecurityUpdate: true,
	}).Error)
	require.NoError(t, database.DB.Create(&models.Package{
		HostID: host.ID, Name: "curl", Version: "7.88.0", PackageManager: "dpkg",
	}).Error)

	router := setupGlobalPackagesRouter()
	cookie := createTestUser(t)

	// Filter by security — only 1 result, but global stats show total=2, security=1
	req, _ := http.NewRequest("GET", "/packages?status=security", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	pagination := resp["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pagination["total"])     // filtered
	assert.Equal(t, float64(2), resp["total_packages"])  // global, unfiltered
	assert.Equal(t, float64(1), resp["security_count"])  // global, unfiltered
}
