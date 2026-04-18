package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"watchflare/backend/config"
	"watchflare/backend/sse"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupSSERouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/events", HostEvents)
	router.GET("/:id/events", HostDetailEvents)
	return router
}

// runSSE runs the SSE handler in a goroutine, cancels after delay, and returns
// the recorder once the handler exits.
func runSSE(t *testing.T, req *http.Request, cancel context.CancelFunc) *httptest.ResponseRecorder {
	t.Helper()
	router := setupSSERouter()
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(w, req)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE handler did not return after context cancel")
	}
	return w
}

func TestHostEvents_CORSAllowedOrigin(t *testing.T) {
	config.AppConfig = &config.Config{
		CORSOrigins: []string{"http://localhost:5173"},
		JWTSecret:   "test-secret-key-must-be-32-chars!!",
	}

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "/events", nil)
	req.Header.Set("Origin", "http://localhost:5173")

	w := runSSE(t, req, cancel)

	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestHostEvents_CORSUnknownOrigin(t *testing.T) {
	config.AppConfig = &config.Config{
		CORSOrigins: []string{"http://localhost:5173"},
		JWTSecret:   "test-secret-key-must-be-32-chars!!",
	}

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "/events", nil)
	req.Header.Set("Origin", "http://evil.example.com")

	w := runSSE(t, req, cancel)

	// Unrecognized origin must not be reflected.
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestHostEvents_CORSNoOrigin(t *testing.T) {
	config.AppConfig = &config.Config{
		CORSOrigins: []string{"http://localhost:5173"},
		JWTSecret:   "test-secret-key-must-be-32-chars!!",
	}

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "/events", nil)
	// No Origin header.

	w := runSSE(t, req, cancel)

	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestHostEvents_SSEHeaders(t *testing.T) {
	config.AppConfig = &config.Config{
		CORSOrigins: []string{"http://localhost:5173"},
		JWTSecret:   "test-secret-key-must-be-32-chars!!",
	}

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "/events", nil)

	w := runSSE(t, req, cancel)

	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "no", w.Header().Get("X-Accel-Buffering"))
}

func TestHostEvents_ConnectedEvent(t *testing.T) {
	config.AppConfig = &config.Config{
		CORSOrigins: []string{"http://localhost:5173"},
		JWTSecret:   "test-secret-key-must-be-32-chars!!",
	}

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "/events", nil)

	w := runSSE(t, req, cancel)

	body := w.Body.String()
	assert.Contains(t, body, sse.EventTypeConnected)
	assert.Contains(t, body, "client_id")
}

func TestHostEvents_DisconnectsCleanly(t *testing.T) {
	config.AppConfig = &config.Config{
		CORSOrigins: []string{},
		JWTSecret:   "test-secret-key-must-be-32-chars!!",
	}

	broker := sse.GetBroker()
	clientsBefore := broker.ClientCount()

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "/events", nil)

	runSSE(t, req, cancel)

	// Client must be removed from broker after disconnect.
	assert.Equal(t, clientsBefore, broker.ClientCount())
}

// ===== HostDetailEvents =====

func TestHostDetailEvents_ConnectedEvent(t *testing.T) {
	config.AppConfig = &config.Config{
		CORSOrigins: []string{"http://localhost:5173"},
		JWTSecret:   "test-secret-key-must-be-32-chars!!",
	}

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "/host-abc/events", nil)

	w := runSSE(t, req, cancel)

	body := w.Body.String()
	assert.Contains(t, body, sse.EventTypeConnected)
	assert.Contains(t, body, "client_id")
}

func TestHostDetailEvents_DisconnectsCleanly(t *testing.T) {
	config.AppConfig = &config.Config{
		CORSOrigins: []string{},
		JWTSecret:   "test-secret-key-must-be-32-chars!!",
	}

	broker := sse.GetBroker()
	clientsBefore := broker.ClientCount()

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", "/host-abc/events", nil)

	runSSE(t, req, cancel)

	assert.Equal(t, clientsBefore, broker.ClientCount())
}
