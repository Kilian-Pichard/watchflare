package services

import (
	"testing"
	"time"
	"watchflare/backend/models"
)

func TestGroupSensorRows_Empty(t *testing.T) {
	result := groupSensorRows(nil)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestGroupSensorRows_SingleTimestamp(t *testing.T) {
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := []sensorRow{
		{Timestamp: ts, SensorKey: "cpu", Temperature: 65.0},
		{Timestamp: ts, SensorKey: "gpu", Temperature: 72.0},
	}

	result := groupSensorRows(rows)

	if len(result) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result))
	}
	if !result[0].Timestamp.Equal(ts) {
		t.Errorf("timestamp: got %v, want %v", result[0].Timestamp, ts)
	}
	if len(result[0].SensorReadings) != 2 {
		t.Errorf("expected 2 sensor readings, got %d", len(result[0].SensorReadings))
	}
}

func TestGroupSensorRows_MultipleTimestamps(t *testing.T) {
	ts1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC)
	rows := []sensorRow{
		{Timestamp: ts1, SensorKey: "cpu", Temperature: 65.0},
		{Timestamp: ts2, SensorKey: "cpu", Temperature: 66.0},
		{Timestamp: ts2, SensorKey: "gpu", Temperature: 73.0},
	}

	result := groupSensorRows(rows)

	if len(result) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(result))
	}
	if len(result[0].SensorReadings) != 1 {
		t.Errorf("group 0: expected 1 reading, got %d", len(result[0].SensorReadings))
	}
	if len(result[1].SensorReadings) != 2 {
		t.Errorf("group 1: expected 2 readings, got %d", len(result[1].SensorReadings))
	}
}

func TestGroupSensorRows_CorrectValues(t *testing.T) {
	ts := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	rows := []sensorRow{
		{Timestamp: ts, SensorKey: "cpu_core0", Temperature: 71.5},
	}

	result := groupSensorRows(rows)

	reading := result[0].SensorReadings[0]
	if reading.Key != "cpu_core0" {
		t.Errorf("key: got %s, want cpu_core0", reading.Key)
	}
	if reading.TemperatureCelsius != 71.5 {
		t.Errorf("temperature: got %f, want 71.5", reading.TemperatureCelsius)
	}
}

func TestGroupSensorRows_PreservesOrder(t *testing.T) {
	ts1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC)
	ts3 := time.Date(2024, 1, 1, 0, 2, 0, 0, time.UTC)
	rows := []sensorRow{
		{Timestamp: ts1, SensorKey: "cpu", Temperature: 60.0},
		{Timestamp: ts2, SensorKey: "cpu", Temperature: 61.0},
		{Timestamp: ts3, SensorKey: "cpu", Temperature: 62.0},
	}

	result := groupSensorRows(rows)

	if len(result) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(result))
	}
	expected := []models.SensorReadings{
		{{Key: "cpu", TemperatureCelsius: 60.0}},
		{{Key: "cpu", TemperatureCelsius: 61.0}},
		{{Key: "cpu", TemperatureCelsius: 62.0}},
	}
	for i, group := range result {
		if group.SensorReadings[0].TemperatureCelsius != expected[i][0].TemperatureCelsius {
			t.Errorf("group %d: temperature got %f, want %f", i,
				group.SensorReadings[0].TemperatureCelsius, expected[i][0].TemperatureCelsius)
		}
	}
}
