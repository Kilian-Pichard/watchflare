package handlers

import (
	"bytes"
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
)

// setupServerRouter creates a test router with server routes
func setupServerRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	serverGroup := router.Group("/servers")
	serverGroup.Use(middleware.AuthMiddleware())
	{
		serverGroup.POST("", CreateAgent)
		serverGroup.GET("", ListServers)
		serverGroup.GET("/:id", GetServer)
		serverGroup.PUT("/:id/validate-ip", ValidateIP)
		serverGroup.PUT("/:id/change-ip", UpdateConfiguredIP)
		serverGroup.POST("/:id/regenerate-token", RegenerateToken)
		serverGroup.DELETE("/:id", DeleteServer)
	}

	return router
}

// createTestUser creates a test user and returns JWT cookie
func createTestUser(t *testing.T) *http.Cookie {
	testUser := &models.User{
		Email: "test@test.com",
	}
	testUser.HashPassword("password123")
	database.DB.Create(testUser)

	// Generate JWT
	token, _ := services.Login("test@test.com", "password123")

	return &http.Cookie{
		Name:  "jwt_token",
		Value: token,
	}
}

func TestCreateAgent(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupServerRouter()
	cookie := createTestUser(t)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		withCookie     bool
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "Success - Create pending server",
			payload: map[string]interface{}{
				"name":          "server01",
				"type":          "vm",
				"configured_ip": "192.168.1.100",
				"allow_any_ip":  false,
			},
			withCookie:     true,
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "Server created successfully", resp["message"])
				assert.NotNil(t, resp["server"])
				assert.NotNil(t, resp["token"])
				assert.NotNil(t, resp["agent_key"])

				server := resp["server"].(map[string]interface{})
				assert.Equal(t, "server01", server["name"])
				assert.Equal(t, "vm", server["type"])
				assert.Equal(t, "pending", server["status"])
			},
		},
		{
			name: "Fail - Invalid server type",
			payload: map[string]interface{}{
				"name":          "server02",
				"type":          "invalid_type",
				"configured_ip": "192.168.1.101",
				"allow_any_ip":  false,
			},
			withCookie:     true,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
		{
			name: "Fail - Missing required fields",
			payload: map[string]interface{}{
				"name": "server03",
			},
			withCookie:     true,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
		{
			name: "Fail - No authentication",
			payload: map[string]interface{}{
				"name":          "server04",
				"type":          "vm",
				"configured_ip": "192.168.1.102",
			},
			withCookie:     false,
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest("POST", "/servers", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			if tt.withCookie {
				req.AddCookie(cookie)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)
		})
	}
}

func TestListServers(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupServerRouter()
	cookie := createTestUser(t)

	// Create test servers
	server1, _, _, _ := services.CreateAgent("server01", "vm", "192.168.1.100", false)
	server2, _, _, _ := services.CreateAgent("server02", "physical", "192.168.1.101", true)

	req, _ := http.NewRequest("GET", "/servers", nil)
	req.AddCookie(cookie)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	servers := response["servers"].([]interface{})
	assert.Len(t, servers, 2)

	// Verify server data
	firstServer := servers[0].(map[string]interface{})
	assert.Equal(t, float64(server1.ID), firstServer["id"])
	assert.Equal(t, "server01", firstServer["name"])
	assert.Equal(t, "pending", firstServer["status"])

	secondServer := servers[1].(map[string]interface{})
	assert.Equal(t, float64(server2.ID), secondServer["id"])
	assert.Equal(t, "server02", secondServer["name"])
}

func TestGetServer(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupServerRouter()
	cookie := createTestUser(t)

	// Create test server
	server, _, _, _ := services.CreateAgent("server01", "vm", "192.168.1.100", false)

	tests := []struct {
		name           string
		serverID       string
		withCookie     bool
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "Success - Get existing server",
			serverID:       fmt.Sprintf("%d", server.ID),
			withCookie:     true,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				serverData := resp["server"].(map[string]interface{})
				assert.Equal(t, "server01", serverData["name"])
				assert.Equal(t, "vm", serverData["type"])
				assert.Equal(t, "pending", serverData["status"])
			},
		},
		{
			name:           "Fail - Server not found",
			serverID:       "999",
			withCookie:     true,
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp["error"], "not found")
			},
		},
		{
			name:           "Fail - Invalid server ID",
			serverID:       "invalid",
			withCookie:     true,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/servers/"+tt.serverID, nil)

			if tt.withCookie {
				req.AddCookie(cookie)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)
		})
	}
}

func TestRegenerateToken(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupServerRouter()
	cookie := createTestUser(t)

	// Create test server
	server, _, _, _ := services.CreateAgent("server01", "vm", "192.168.1.100", false)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/servers/%d/regenerate-token", server.ID), nil)
	req.AddCookie(cookie)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "Token regenerated successfully", response["message"])
	assert.NotNil(t, response["token"])

	// Verify token format
	token := response["token"].(string)
	assert.Contains(t, token, "wf_reg_")
}

func TestDeleteServer(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupServerRouter()
	cookie := createTestUser(t)

	tests := []struct {
		name           string
		setupServer    func() uint
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "Success - Delete pending server",
			setupServer: func() uint {
				server, _, _, _ := services.CreateAgent("server01", "vm", "192.168.1.100", false)
				return server.ID
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "Server deleted successfully", resp["message"])
			},
		},
		{
			name: "Fail - Delete non-existent server",
			setupServer: func() uint {
				return 999
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp["error"], "not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverID := tt.setupServer()

			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/servers/%d", serverID), nil)
			req.AddCookie(cookie)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)
		})
	}
}

func TestValidateIP(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupServerRouter()
	cookie := createTestUser(t)

	// Create test server
	server, _, _, _ := services.CreateAgent("server01", "vm", "192.168.1.100", false)

	payload := map[string]string{
		"selected_ip": "192.168.1.100",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/servers/%d/validate-ip", server.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "IP validated successfully", response["message"])
}

func TestUpdateConfiguredIP(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupServerRouter()
	cookie := createTestUser(t)

	// Create test server
	server, _, _, _ := services.CreateAgent("server01", "vm", "192.168.1.100", false)

	payload := map[string]string{
		"new_ip": "192.168.1.200",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/servers/%d/change-ip", server.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Configured IP updated successfully", response["message"])

	// Verify IP was updated
	updatedServer, _ := services.GetServer(server.ID)
	assert.Equal(t, "192.168.1.200", *updatedServer.ConfiguredIP)
}
