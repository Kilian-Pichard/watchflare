package services

import (
	"errors"
	"fmt"
	"time"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"gorm.io/gorm"
)

// SensorDataPoint groups all sensor readings for a single timestamp bucket.
type SensorDataPoint struct {
	Timestamp      time.Time             `json:"timestamp"`
	SensorReadings models.SensorReadings `json:"sensor_readings"`
}

// sensorRow is used to scan flat rows from the DB before grouping.
type sensorRow struct {
	Timestamp   time.Time `gorm:"column:ts"`
	SensorKey   string    `gorm:"column:sensor_key"`
	Temperature float64   `gorm:"column:temperature"`
}

// GetSensorReadings retrieves per-sensor temperature data for a server.
//
// For 1h/12h/24h: queries metrics.sensor_readings (JSONB) — always available.
// For 7d/30d:     queries sensor_metrics hypertable — accumulates over time.
func GetSensorReadings(serverID string, start, end time.Time, interval string) ([]SensorDataPoint, error) {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServerNotFound
		}
		return nil, err
	}

	var rows []sensorRow
	var err error

	switch interval {
	case "", "10m", "15m":
		// 1h / 12h / 24h — expand JSONB from the metrics table (retained 24h)
		// bucket comes from a fixed map — no SQL injection risk.
		bucket := map[string]string{"": "30 seconds", "10m": "10 minutes", "15m": "15 minutes"}[interval]
		err = database.DB.Raw(fmt.Sprintf(`
			SELECT time_bucket('%s', timestamp) AS ts,
			       sr->>'key' AS sensor_key,
			       AVG((sr->>'temperature_celsius')::double precision) AS temperature
			FROM metrics,
			     LATERAL jsonb_array_elements(sensor_readings) AS sr
			WHERE server_id = $1
			  AND timestamp >= $2
			  AND timestamp <= $3
			  AND sensor_readings IS NOT NULL
			  AND jsonb_array_length(sensor_readings) > 0
			GROUP BY 1, 2
			ORDER BY 1 ASC, 2 ASC
		`, bucket), serverID, start, end).Scan(&rows).Error

	case "2h", "8h":
		// 7d / 30d — dedicated sensor_metrics hypertable
		// bucket comes from a fixed map — no SQL injection risk.
		bucket := map[string]string{"2h": "2 hours", "8h": "8 hours"}[interval]
		err = database.DB.Raw(fmt.Sprintf(`
			SELECT time_bucket('%s', time) AS ts,
			       sensor_key,
			       AVG(temperature) AS temperature
			FROM sensor_metrics
			WHERE server_id = $1
			  AND time >= $2
			  AND time <= $3
			GROUP BY 1, 2
			ORDER BY 1 ASC, 2 ASC
		`, bucket), serverID, start, end).Scan(&rows).Error

	default:
		return nil, fmt.Errorf("invalid interval: %s", interval)
	}

	if err != nil {
		return nil, err
	}

	return groupSensorRows(rows), nil
}

// groupSensorRows converts flat (ts, key, temp) rows into SensorDataPoints.
func groupSensorRows(rows []sensorRow) []SensorDataPoint {
	if len(rows) == 0 {
		return nil
	}

	var result []SensorDataPoint
	var current *SensorDataPoint

	for _, r := range rows {
		if current == nil || !current.Timestamp.Equal(r.Timestamp) {
			result = append(result, SensorDataPoint{
				Timestamp:      r.Timestamp,
				SensorReadings: models.SensorReadings{},
			})
			current = &result[len(result)-1]
		}
		current.SensorReadings = append(current.SensorReadings, models.SensorReading{
			Key:                r.SensorKey,
			TemperatureCelsius: r.Temperature,
		})
	}

	return result
}
