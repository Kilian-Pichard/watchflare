package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"watchflare/backend/config"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func setupTestConfig() {
	config.AppConfig = &config.Config{
		JWTSecret: "test-secret-key",
	}
}

func setupTestDB(t *testing.T) {
	t.Helper()
	if err := database.Connect(); err != nil {
		t.Skipf("skipping test: database unavailable: %v", err)
	}
}

func teardownTestDB() {
	database.DB.Exec("DELETE FROM users")
}

func generateTestJWT(userID string, secret string, expired bool) string {
	var exp time.Time
	if expired {
		exp = time.Now().Add(-time.Hour) // Expired 1 hour ago
	} else {
		exp = time.Now().Add(time.Hour * 24) // Valid for 24 hours
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     exp.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestAuthMiddleware(t *testing.T) {
	setupTestConfig()
	setupTestDB(t)
	defer teardownTestDB()
	gin.SetMode(gin.TestMode)

	// Create a test user
	testUser := &models.User{
		ID:       "550e8400-e29b-41d4-a716-446655440001",
		Email:    "test@example.com",
		Password: "hashedpassword",
	}
	database.DB.Create(testUser)

	tests := []struct {
		name           string
		setupRequest   func(*http.Request)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Success - Valid JWT token",
			setupRequest: func(req *http.Request) {
				token := generateTestJWT("550e8400-e29b-41d4-a716-446655440001", config.AppConfig.JWTSecret, false)
				req.AddCookie(&http.Cookie{
					Name:  "jwt_token",
					Value: token,
				})
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "protected content", w.Body.String())
			},
		},
		{
			name: "Fail - User not found in database",
			setupRequest: func(req *http.Request) {
				// Use a valid JWT but for a user that doesn't exist
				token := generateTestJWT("00000000-0000-0000-0000-000000000000", config.AppConfig.JWTSecret, false)
				req.AddCookie(&http.Cookie{
					Name:  "jwt_token",
					Value: token,
				})
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "User not found")
			},
		},
		{
			name: "Fail - No JWT token",
			setupRequest: func(req *http.Request) {
				// No cookie
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Authentication required")
			},
		},
		{
			name: "Fail - Expired JWT token",
			setupRequest: func(req *http.Request) {
				token := generateTestJWT("550e8400-e29b-41d4-a716-446655440001", config.AppConfig.JWTSecret, true)
				req.AddCookie(&http.Cookie{
					Name:  "jwt_token",
					Value: token,
				})
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Invalid or expired token")
			},
		},
		{
			name: "Fail - Invalid JWT secret",
			setupRequest: func(req *http.Request) {
				token := generateTestJWT("550e8400-e29b-41d4-a716-446655440001", "wrong-secret", false)
				req.AddCookie(&http.Cookie{
					Name:  "jwt_token",
					Value: token,
				})
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Invalid or expired token")
			},
		},
		{
			name: "Fail - Malformed JWT token",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "jwt_token",
					Value: "malformed.token.here",
				})
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Invalid or expired token")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware())
			router.GET("/protected", func(c *gin.Context) {
				userID, exists := c.Get("user_id")
				assert.True(t, exists)
				assert.NotNil(t, userID)
				c.String(http.StatusOK, "protected content")
			})

			req, _ := http.NewRequest("GET", "/protected", nil)
			tt.setupRequest(req)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)
		})
	}
}

func TestAuthMiddleware_UserIDInContext(t *testing.T) {
	setupTestConfig()
	setupTestDB(t)
	defer teardownTestDB()
	gin.SetMode(gin.TestMode)

	testUserID := "550e8400-e29b-41d4-a716-446655440042"

	// Create a test user
	testUser := &models.User{
		ID:       testUserID,
		Email:    "test2@example.com",
		Password: "hashedpassword",
	}
	database.DB.Create(testUser)

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		assert.True(t, exists, "user_id should exist in context")
		assert.Equal(t, testUserID, userID, "user_id should match")
		c.String(http.StatusOK, "ok")
	})

	token := generateTestJWT(testUserID, config.AppConfig.JWTSecret, false)
	req, _ := http.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "jwt_token",
		Value: token,
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
