package config

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	t.Run("returns env var when set", func(t *testing.T) {
		os.Setenv("TEST_KEY", "test_value")
		defer os.Unsetenv("TEST_KEY")
		assert.Equal(t, "test_value", getEnv("TEST_KEY", "default"))
	})

	t.Run("returns default when not set", func(t *testing.T) {
		os.Unsetenv("TEST_KEY")
		assert.Equal(t, "default", getEnv("TEST_KEY", "default"))
	})
}

func TestGetIntEnv(t *testing.T) {
	t.Run("returns parsed int", func(t *testing.T) {
		os.Setenv("TEST_INT", "42")
		defer os.Unsetenv("TEST_INT")
		assert.Equal(t, 42, getIntEnv("TEST_INT", 0))
	})

	t.Run("returns default on invalid value", func(t *testing.T) {
		os.Setenv("TEST_INT", "not_a_number")
		defer os.Unsetenv("TEST_INT")
		assert.Equal(t, 10, getIntEnv("TEST_INT", 10))
	})

	t.Run("returns default when not set", func(t *testing.T) {
		os.Unsetenv("TEST_INT")
		assert.Equal(t, 5, getIntEnv("TEST_INT", 5))
	})
}

func TestParseOrigins(t *testing.T) {
	t.Run("splits multiple origins", func(t *testing.T) {
		result := parseOrigins("http://localhost:5173,http://localhost:3000")
		assert.Equal(t, []string{"http://localhost:5173", "http://localhost:3000"}, result)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		result := parseOrigins("http://localhost:5173, http://localhost:3000")
		assert.Equal(t, []string{"http://localhost:5173", "http://localhost:3000"}, result)
	})

	t.Run("returns empty slice for empty string", func(t *testing.T) {
		result := parseOrigins("")
		assert.Equal(t, []string{}, result)
	})

	t.Run("single origin", func(t *testing.T) {
		result := parseOrigins("http://localhost:5173")
		assert.Equal(t, []string{"http://localhost:5173"}, result)
	})
}

func boolPtr(b bool) *bool { return &b }

func TestCookieSecure(t *testing.T) {
	AppConfig = &Config{TrustedProxies: []string{"127.0.0.1", "::1"}}

	t.Run("override true forces secure regardless of connection", func(t *testing.T) {
		AppConfig.CookieSecureOverride = boolPtr(true)
		assert.True(t, CookieSecure(false, "10.0.0.1:1234", ""))
	})

	t.Run("override false forces insecure regardless of TLS", func(t *testing.T) {
		AppConfig.CookieSecureOverride = boolPtr(false)
		assert.False(t, CookieSecure(true, "127.0.0.1:1234", "https"))
	})

	t.Run("direct TLS returns true when no override", func(t *testing.T) {
		AppConfig.CookieSecureOverride = nil
		assert.True(t, CookieSecure(true, "10.0.0.5:1234", ""))
	})

	t.Run("trusted proxy with X-Forwarded-Proto https returns true", func(t *testing.T) {
		AppConfig.CookieSecureOverride = nil
		assert.True(t, CookieSecure(false, "127.0.0.1:54321", "https"))
	})

	t.Run("trusted proxy with X-Forwarded-Proto http returns false", func(t *testing.T) {
		AppConfig.CookieSecureOverride = nil
		assert.False(t, CookieSecure(false, "127.0.0.1:54321", "http"))
	})

	t.Run("untrusted proxy header is ignored", func(t *testing.T) {
		AppConfig.CookieSecureOverride = nil
		assert.False(t, CookieSecure(false, "10.0.0.99:1234", "https"))
	})

	t.Run("plain HTTP no proxy returns false", func(t *testing.T) {
		AppConfig.CookieSecureOverride = nil
		assert.False(t, CookieSecure(false, "192.168.1.5:1234", ""))
	})

	t.Run("ipv6 loopback trusted proxy", func(t *testing.T) {
		AppConfig.CookieSecureOverride = nil
		assert.True(t, CookieSecure(false, "[::1]:54321", "https"))
	})
}

func TestParseProxies(t *testing.T) {
	t.Run("parses comma-separated IPs", func(t *testing.T) {
		result := parseProxies("127.0.0.1,::1")
		assert.Equal(t, []string{"127.0.0.1", "::1"}, result)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		result := parseProxies("127.0.0.1, ::1")
		assert.Equal(t, []string{"127.0.0.1", "::1"}, result)
	})

	t.Run("returns empty slice for empty string", func(t *testing.T) {
		result := parseProxies("")
		assert.Equal(t, []string{}, result)
	})
}

func TestLoad_ValidConfig(t *testing.T) {
	os.Setenv("JWT_SECRET", "a-valid-secret-that-is-at-least-32-chars!")
	defer os.Unsetenv("JWT_SECRET")

	Load()

	assert.NotNil(t, AppConfig)
	assert.Equal(t, "50051", AppConfig.GRPCPort)
	assert.Equal(t, 300, AppConfig.GRPCTimestampWindow)
}

// TestLoad_EmptyJWTSecret and TestLoad_ShortJWTSecret use the subprocess pattern
// to test os.Exit behavior without killing the test process.

func TestLoad_EmptyJWTSecret(t *testing.T) {
	if os.Getenv("TEST_CRASHER") == "empty_jwt" {
		os.Unsetenv("JWT_SECRET")
		Load()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestLoad_EmptyJWTSecret")
	cmd.Env = append(os.Environ(), "TEST_CRASHER=empty_jwt")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatal("expected os.Exit(1) for empty JWT_SECRET")
}

func TestLoad_ShortJWTSecret(t *testing.T) {
	if os.Getenv("TEST_CRASHER") == "short_jwt" {
		os.Setenv("JWT_SECRET", "tooshort")
		Load()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestLoad_ShortJWTSecret")
	cmd.Env = append(os.Environ(), "TEST_CRASHER=short_jwt")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatal("expected os.Exit(1) for JWT_SECRET shorter than 32 chars")
}
