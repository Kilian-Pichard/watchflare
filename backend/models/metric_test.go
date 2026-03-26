package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSensorReadings_Roundtrip(t *testing.T) {
	t.Run("Value then Scan roundtrip", func(t *testing.T) {
		original := SensorReadings{
			{Key: "cpu_temp", TemperatureCelsius: 55.5},
			{Key: "gpu_temp", TemperatureCelsius: 72.0},
		}

		val, err := original.Value()
		require.NoError(t, err)
		require.NotNil(t, val)

		var restored SensorReadings
		require.NoError(t, restored.Scan(val))
		assert.Equal(t, original, restored)
	})

	t.Run("empty slice produces nil Value", func(t *testing.T) {
		s := SensorReadings{}
		val, err := s.Value()
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("Scan nil sets nil slice", func(t *testing.T) {
		var s SensorReadings
		require.NoError(t, s.Scan(nil))
		assert.Nil(t, s)
	})

	t.Run("Scan string input", func(t *testing.T) {
		var s SensorReadings
		require.NoError(t, s.Scan(`[{"key":"cpu","temperature_celsius":45.0}]`))
		require.Len(t, s, 1)
		assert.Equal(t, "cpu", s[0].Key)
		assert.Equal(t, 45.0, s[0].TemperatureCelsius)
	})

	t.Run("Scan unknown type returns error", func(t *testing.T) {
		var s SensorReadings
		assert.Error(t, s.Scan(12345))
	})
}
