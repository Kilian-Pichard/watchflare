package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"watchflare/backend/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func setupTestConfig() {
	config.AppConfig = &config.Config{
		JWTSecret: "test-secret-key",
	}
}

func generateTestJWT(userID uint, secret string, expired bool) string {
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
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupRequest   func(*http.Request)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Success - Valid JWT token",
			setupRequest: func(req *http.Request) {
				token := generateTestJWT(1, config.AppConfig.JWTSecret, false)
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
				token := generateTestJWT(1, config.AppConfig.JWTSecret, true)
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
				token := generateTestJWT(1, "wrong-secret", false)
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
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		assert.True(t, exists, "user_id should exist in context")
		assert.Equal(t, uint(42), userID, "user_id should be 42")
		c.String(http.StatusOK, "ok")
	})

	token := generateTestJWT(42, config.AppConfig.JWTSecret, false)
	req, _ := http.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "jwt_token",
		Value: token,
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
