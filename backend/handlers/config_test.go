package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"watchflare/backend/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupConfigRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/config", GetAppConfig)
	return r
}

func TestGetAppConfig_CookieSecure(t *testing.T) {
	config.AppConfig = &config.Config{
		TrustedProxies: []string{"127.0.0.1", "::1"},
	}

	t.Run("plain HTTP — cookie_secure false", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()
		setupConfigRouter().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, false, resp["cookie_secure"])
	})

	t.Run("trusted proxy with X-Forwarded-Proto https — cookie_secure true", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/config", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.RemoteAddr = "127.0.0.1:54321"
		w := httptest.NewRecorder()
		setupConfigRouter().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, true, resp["cookie_secure"])
	})

	t.Run("untrusted source with X-Forwarded-Proto https — cookie_secure false", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/config", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.RemoteAddr = "10.0.0.99:54321"
		w := httptest.NewRecorder()
		setupConfigRouter().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, false, resp["cookie_secure"])
	})

	t.Run("COOKIE_SECURE override true — cookie_secure always true", func(t *testing.T) {
		override := true
		config.AppConfig.CookieSecureOverride = &override
		defer func() { config.AppConfig.CookieSecureOverride = nil }()

		req, _ := http.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()
		setupConfigRouter().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, true, resp["cookie_secure"])
	})
}
