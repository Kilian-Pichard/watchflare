package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"watchflare/backend/config"
	"watchflare/backend/database"
	"watchflare/backend/middleware"
	"watchflare/backend/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	config.AppConfig = &config.Config{
		JWTSecret: "test-secret-key",
	}
	if err := database.Connect(); err != nil {
		t.Skipf("skipping test: database unavailable: %v", err)
	}
}

// teardownTestDB cleans up the test database
func teardownTestDB() {
	database.DB.Exec("DELETE FROM servers")
	database.DB.Exec("DELETE FROM users")
}

// setupRouter creates a test router with routes
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Auth routes (public)
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", Register)
		authGroup.POST("/login", Login)
		authGroup.POST("/logout", Logout)
	}

	// Protected routes
	protectedGroup := router.Group("/auth")
	protectedGroup.Use(middleware.AuthMiddleware())
	{
		protectedGroup.PUT("/change-password", ChangePassword)
	}

	return router
}

// TestRegister tests user registration
func TestRegister(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupRouter()

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "Success - First admin registration",
			payload: map[string]string{
				"email":    "admin@test.com",
				"password": "password123",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "User registered successfully", resp["message"])
				assert.NotNil(t, resp["user"])
			},
		},
		{
			name: "Fail - Second admin registration",
			payload: map[string]string{
				"email":    "admin2@test.com",
				"password": "password456",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp["error"], "admin user already exists")
			},
		},
		{
			name: "Fail - Invalid email",
			payload: map[string]string{
				"email":    "invalid-email",
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
		{
			name: "Fail - Password too short",
			payload: map[string]string{
				"email":    "test@test.com",
				"password": "short",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)
		})
	}
}

// TestLogin tests user login
func TestLogin(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupRouter()

	// Create a test user
	testUser := &models.User{
		Email: "test@test.com",
	}
	testUser.HashPassword("password123")
	database.DB.Create(testUser)

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder, map[string]interface{})
	}{
		{
			name: "Success - Valid credentials",
			payload: map[string]string{
				"email":    "test@test.com",
				"password": "password123",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, resp map[string]interface{}) {
				assert.Equal(t, "Login successful", resp["message"])
				// Check cookie
				cookies := w.Result().Cookies()
				assert.NotEmpty(t, cookies)
				var jwtCookie *http.Cookie
				for _, cookie := range cookies {
					if cookie.Name == "jwt_token" {
						jwtCookie = cookie
						break
					}
				}
				assert.NotNil(t, jwtCookie)
				assert.NotEmpty(t, jwtCookie.Value)
				assert.True(t, jwtCookie.HttpOnly)
			},
		},
		{
			name: "Fail - Wrong password",
			payload: map[string]string{
				"email":    "test@test.com",
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, resp map[string]interface{}) {
				assert.Contains(t, resp["error"], "invalid credentials")
			},
		},
		{
			name: "Fail - User not found",
			payload: map[string]string{
				"email":    "notfound@test.com",
				"password": "password123",
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, resp map[string]interface{}) {
				assert.Contains(t, resp["error"], "invalid credentials")
			},
		},
		{
			name: "Fail - Invalid email format",
			payload: map[string]string{
				"email":    "invalid-email",
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, w, response)
		})
	}
}

// TestLogout tests user logout
func TestLogout(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupRouter()

	req, _ := http.NewRequest("POST", "/auth/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Logout successful", response["message"])

	// Check cookie is cleared
	cookies := w.Result().Cookies()
	var jwtCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "jwt_token" {
			jwtCookie = cookie
			break
		}
	}
	assert.NotNil(t, jwtCookie)
	assert.Equal(t, "", jwtCookie.Value)
	assert.Equal(t, -1, jwtCookie.MaxAge)
}

// TestChangePassword tests password change
func TestChangePassword(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	router := setupRouter()

	// Create a test user and login
	testUser := &models.User{
		Email: "test@test.com",
	}
	testUser.HashPassword("oldpassword123")
	database.DB.Create(testUser)

	// Login to get JWT cookie
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "test@test.com",
		"password": "oldpassword123",
	})
	loginReq, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	// Extract JWT cookie
	var jwtCookie *http.Cookie
	for _, cookie := range loginW.Result().Cookies() {
		if cookie.Name == "jwt_token" {
			jwtCookie = cookie
			break
		}
	}

	tests := []struct {
		name           string
		payload        map[string]string
		withCookie     bool
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "Success - Valid password change",
			payload: map[string]string{
				"current_password": "oldpassword123",
				"new_password":     "newpassword456",
			},
			withCookie:     true,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "Password changed successfully", resp["message"])
			},
		},
		{
			name: "Fail - Wrong current password",
			payload: map[string]string{
				"current_password": "wrongpassword",
				"new_password":     "newpassword456",
			},
			withCookie:     true,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp["error"], "current password is incorrect")
			},
		},
		{
			name: "Fail - New password too short",
			payload: map[string]string{
				"current_password": "oldpassword123",
				"new_password":     "short",
			},
			withCookie:     true,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.NotNil(t, resp["error"])
			},
		},
		{
			name: "Fail - No authentication",
			payload: map[string]string{
				"current_password": "oldpassword123",
				"new_password":     "newpassword456",
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
			req, _ := http.NewRequest("PUT", "/auth/change-password", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			if tt.withCookie {
				req.AddCookie(jwtCookie)
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
