package models

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	t.Run("produces a valid bcrypt hash", func(t *testing.T) {
		u := &User{}
		require.NoError(t, u.HashPassword("mysecretpassword"))
		assert.NotEmpty(t, u.Password)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("mysecretpassword")))
	})

	t.Run("wrong password does not match", func(t *testing.T) {
		u := &User{}
		require.NoError(t, u.HashPassword("correctpassword"))
		assert.Error(t, bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("wrongpassword")))
	})

	t.Run("stores hash not plaintext", func(t *testing.T) {
		u := &User{}
		require.NoError(t, u.HashPassword("plaintext"))
		assert.NotEqual(t, "plaintext", u.Password)
	})
}

func TestUserBeforeCreate(t *testing.T) {
	t.Run("generates UUID when ID is empty", func(t *testing.T) {
		u := &User{}
		require.NoError(t, u.BeforeCreate(nil))
		assert.NotEmpty(t, u.ID)
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		u := &User{ID: "existing-id"}
		require.NoError(t, u.BeforeCreate(nil))
		assert.Equal(t, "existing-id", u.ID)
	})

	t.Run("sets all defaults when fields are empty", func(t *testing.T) {
		u := &User{}
		require.NoError(t, u.BeforeCreate(nil))
		assert.Equal(t, "1h", u.DefaultTimeRange)
		assert.Equal(t, "system", u.Theme)
		assert.Equal(t, "24h", u.TimeFormat)
		assert.Equal(t, "celsius", u.TemperatureUnit)
		assert.Equal(t, "bytes", u.NetworkUnit)
		assert.Equal(t, "bytes", u.DiskUnit)
		assert.Equal(t, 70, u.GaugeWarningThreshold)
		assert.Equal(t, 90, u.GaugeCriticalThreshold)
	})

	t.Run("preserves existing field values", func(t *testing.T) {
		u := &User{
			DefaultTimeRange:       "7d",
			Theme:                  "dark",
			TimeFormat:             "12h",
			TemperatureUnit:        "fahrenheit",
			NetworkUnit:            "bits",
			DiskUnit:               "bits",
			GaugeWarningThreshold:  50,
			GaugeCriticalThreshold: 80,
		}
		require.NoError(t, u.BeforeCreate(nil))
		assert.Equal(t, "7d", u.DefaultTimeRange)
		assert.Equal(t, "dark", u.Theme)
		assert.Equal(t, "12h", u.TimeFormat)
		assert.Equal(t, "fahrenheit", u.TemperatureUnit)
		assert.Equal(t, "bits", u.NetworkUnit)
		assert.Equal(t, "bits", u.DiskUnit)
		assert.Equal(t, 50, u.GaugeWarningThreshold)
		assert.Equal(t, 80, u.GaugeCriticalThreshold)
	})
}
