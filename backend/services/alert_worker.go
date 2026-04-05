package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"watchflare/backend/cache"
	"watchflare/backend/config"
	"watchflare/backend/database"
	"watchflare/backend/encryption"
	"watchflare/backend/models"

	"gorm.io/gorm"
)

// AlertWorker evaluates alert rules every interval and fires email notifications.
type AlertWorker struct {
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewAlertWorker(interval time.Duration) *AlertWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &AlertWorker{
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start runs the worker loop. Call in a goroutine.
func (w *AlertWorker) Start() {
	slog.Info("alert worker starting", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.evaluate()
		case <-w.ctx.Done():
			slog.Info("alert worker stopped")
			return
		}
	}
}

func (w *AlertWorker) Stop() {
	w.cancel()
}

// evaluate runs one evaluation cycle across all monitored servers.
func (w *AlertWorker) evaluate() {
	// Load all servers that should be monitored (skip paused and expired)
	var servers []models.Server
	if err := database.DB.
		Where("status NOT IN ?", []string{models.StatusPaused, models.StatusExpired, models.StatusPending}).
		Find(&servers).Error; err != nil {
		slog.Error("alert worker: failed to load servers", "error", err)
		return
	}
	if len(servers) == 0 {
		return
	}

	// Load all global rules once
	globalRules, err := GetAlertRules()
	if err != nil {
		slog.Error("alert worker: failed to load alert rules", "error", err)
		return
	}
	if len(globalRules) == 0 {
		return
	}

	// Check whether any rule is enabled before doing further work
	anyEnabled := false
	for _, r := range globalRules {
		if r.Enabled {
			anyEnabled = true
			break
		}
	}
	if !anyEnabled {
		// Check server-level overrides
		var count int64
		if err := database.DB.Model(&models.ServerAlertRule{}).Where("enabled = true").Count(&count).Error; err != nil {
			slog.Error("alert worker: failed to count server overrides", "error", err)
			return
		}
		if count == 0 {
			return
		}
	}

	// Load first user as notification recipient
	recipient, err := firstUserEmail()
	if err != nil {
		slog.Error("alert worker: failed to get notification recipient", "error", err)
		return
	}

	// Build a global rule map for quick lookup
	globalMap := make(map[string]models.AlertRule, len(globalRules))
	for _, r := range globalRules {
		globalMap[r.MetricType] = r
	}

	hbCache := cache.GetCache()
	now := time.Now()

	for i := range servers {
		server := &servers[i]
		w.evaluateServer(server, globalMap, hbCache, recipient, now)
	}
}

func (w *AlertWorker) evaluateServer(
	server *models.Server,
	globalMap map[string]models.AlertRule,
	hbCache *cache.HeartbeatCache,
	recipient string,
	now time.Time,
) {
	// Load server-level overrides
	var overrides []models.ServerAlertRule
	if err := database.DB.Where("server_id = ?", server.ID).Find(&overrides).Error; err != nil {
		slog.Error("alert worker: failed to load server overrides", "server_id", server.ID, "error", err)
		return
	}
	overrideMap := make(map[string]models.ServerAlertRule, len(overrides))
	for _, o := range overrides {
		overrideMap[o.MetricType] = o
	}

	// Get the server's current status: prefer HeartbeatCache, fall back to DB value
	var currentStatus string
	if hb, ok := hbCache.Get(server.AgentID); ok {
		currentStatus = hb.Status
	} else {
		currentStatus = server.Status
	}

	// Load the latest metric row (used for all non-server_down metrics)
	var latestMetric *models.Metric
	var m models.Metric
	err := database.DB.
		Where("server_id = ?", server.ID).
		Order("timestamp DESC").
		First(&m).Error
	if err == nil {
		latestMetric = &m
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.Error("alert worker: failed to load latest metric", "server_id", server.ID, "error", err)
		return
	}

	for _, metricType := range models.AllMetricTypes {
		// Resolve effective rule (server override > global)
		var enabled bool
		var threshold float64
		var durationMinutes int

		if o, ok := overrideMap[metricType]; ok {
			enabled = o.Enabled
			threshold = o.Threshold
			durationMinutes = o.DurationMinutes
		} else if g, ok := globalMap[metricType]; ok {
			enabled = g.Enabled
			threshold = g.Threshold
			durationMinutes = g.DurationMinutes
		} else {
			continue
		}

		if !enabled {
			// Rule disabled — resolve any open incident silently
			resolveIncident(server.ID, metricType, now)
			continue
		}

		// Evaluate threshold
		breaching, currentValue := evaluateMetric(metricType, threshold, currentStatus, latestMetric)

		// Find open incident
		var incident models.AlertIncident
		hasIncident := true
		if err := database.DB.
			Where("server_id = ? AND metric_type = ? AND resolved_at IS NULL", server.ID, metricType).
			First(&incident).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				hasIncident = false
			} else {
				slog.Error("alert worker: failed to query incident", "server_id", server.ID, "metric_type", metricType, "error", err)
				continue
			}
		}

		if breaching {
			if !hasIncident {
				// For server_down, backdate the incident start to last_seen so the
				// duration counts from when the server actually went offline, not
				// from when the worker first noticed it.
				// Priority: HeartbeatCache (real-time) > server.LastSeen (DB, ~5min lag) > now
				startedAt := now
				if metricType == models.MetricTypeServerDown {
					if hb, ok := hbCache.Get(server.AgentID); ok && hb.Status == models.StatusOffline {
						startedAt = hb.LastSeen
					} else if server.LastSeen != nil {
						startedAt = *server.LastSeen
					}
				}

				incident = models.AlertIncident{
					ServerID:       server.ID,
					MetricType:     metricType,
					StartedAt:      startedAt,
					ThresholdValue: threshold,
					CurrentValue:   currentValue,
				}
				if err := database.DB.Create(&incident).Error; err != nil {
					slog.Error("alert worker: failed to create incident", "server_id", server.ID, "metric_type", metricType, "error", err)
					continue
				}
				// Fall through to immediately check duration (avoids waiting one
				// extra tick when the condition has already been met).
			} else {
				// Update current value on existing incident
				if err := database.DB.Model(&incident).Update("current_value", currentValue).Error; err != nil {
					slog.Error("alert worker: failed to update incident value", "server_id", server.ID, "metric_type", metricType, "error", err)
				}
			}

			// Fire notification if duration exceeded and not yet notified
			if !incident.Notified && now.Sub(incident.StartedAt) >= time.Duration(durationMinutes)*time.Minute {
				if err := sendAlertEmail(server, metricType, threshold, currentValue, incident.StartedAt, recipient); err != nil {
					slog.Error("alert worker: failed to send alert email",
						"server_id", server.ID, "metric_type", metricType, "error", err)
				} else {
					if err := database.DB.Model(&incident).Update("notified", true).Error; err != nil {
						slog.Error("alert worker: failed to mark incident notified",
							"server_id", server.ID, "metric_type", metricType, "error", err)
					} else {
						slog.Info("alert fired",
							"server", server.Name, "metric_type", metricType,
							"current_value", currentValue, "threshold", threshold)
					}
				}
			}
		} else {
			// Not breaching — resolve open incident if any
			if hasIncident {
				if err := database.DB.Model(&incident).Update("resolved_at", now).Error; err != nil {
					slog.Error("alert worker: failed to resolve incident",
						"server_id", server.ID, "metric_type", metricType, "error", err)
				} else {
					slog.Info("alert resolved", "server", server.Name, "metric_type", metricType)
				}
			}
		}
	}
}

// evaluateMetric returns whether the metric is breaching the threshold and its current value.
func evaluateMetric(
	metricType string,
	threshold float64,
	status string,
	m *models.Metric,
) (breaching bool, currentValue float64) {
	switch metricType {
	case models.MetricTypeServerDown:
		if status == models.StatusOffline {
			return true, 0
		}
		return false, 0

	case models.MetricTypeCPUUsage:
		if m == nil {
			return false, 0
		}
		return m.CPUUsagePercent >= threshold, m.CPUUsagePercent

	case models.MetricTypeMemoryUsage:
		if m == nil || m.MemoryTotalBytes == 0 {
			return false, 0
		}
		pct := float64(m.MemoryUsedBytes) / float64(m.MemoryTotalBytes) * 100
		return pct >= threshold, pct

	case models.MetricTypeDiskUsage:
		if m == nil || m.DiskTotalBytes == 0 {
			return false, 0
		}
		pct := float64(m.DiskUsedBytes) / float64(m.DiskTotalBytes) * 100
		return pct >= threshold, pct

	case models.MetricTypeLoadAvg:
		if m == nil {
			return false, 0
		}
		return m.LoadAvg1Min >= threshold, m.LoadAvg1Min

	case models.MetricTypeLoadAvg5:
		if m == nil {
			return false, 0
		}
		return m.LoadAvg5Min >= threshold, m.LoadAvg5Min

	case models.MetricTypeLoadAvg15:
		if m == nil {
			return false, 0
		}
		return m.LoadAvg15Min >= threshold, m.LoadAvg15Min

	case models.MetricTypeTemperature:
		if m == nil || m.CPUTemperatureCelsius == 0 {
			return false, 0
		}
		return m.CPUTemperatureCelsius >= threshold, m.CPUTemperatureCelsius
	}
	return false, 0
}

// resolveIncident silently resolves any open incident for the given server + metric type.
func resolveIncident(serverID, metricType string, now time.Time) {
	database.DB.Model(&models.AlertIncident{}).
		Where("server_id = ? AND metric_type = ? AND resolved_at IS NULL", serverID, metricType).
		Update("resolved_at", now)
}

// sendAlertEmail delivers an alert notification email.
func sendAlertEmail(server *models.Server, metricType string, threshold, currentValue float64, startedAt time.Time, recipient string) error {
	var s models.SmtpSettings
	if err := database.DB.First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // SMTP not configured — skip silently
		}
		return err
	}
	if !s.Enabled {
		return nil // SMTP disabled — skip silently
	}

	var plainPassword string
	if s.EncryptedPassword != "" {
		if config.AppConfig.SMTPEncryptionKey == "" {
			return errors.New("SMTP_ENCRYPTION_KEY is not configured")
		}
		var err error
		plainPassword, err = encryption.Decrypt(s.EncryptedPassword, config.AppConfig.SMTPEncryptionKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt SMTP password: %w", err)
		}
	}

	subject, body := buildAlertEmailContent(server.Name, metricType, threshold, currentValue, startedAt)
	return sendEmail(&s, plainPassword, recipient, subject, body)
}

func buildAlertEmailContent(serverName, metricType string, threshold, currentValue float64, startedAt time.Time) (subject, body string) {
	var metricLabel, valueDesc string

	switch metricType {
	case models.MetricTypeServerDown:
		subject = fmt.Sprintf("[Watchflare Alert] %s is offline", serverName)
		body = fmt.Sprintf("Server %q has been offline since %s.\n\nThis alert was triggered by Watchflare.",
			serverName, startedAt.Format(time.RFC1123))
		return

	case models.MetricTypeCPUUsage:
		metricLabel = "CPU usage"
		valueDesc = fmt.Sprintf("%.1f%% (threshold: %.0f%%)", currentValue, threshold)

	case models.MetricTypeMemoryUsage:
		metricLabel = "Memory usage"
		valueDesc = fmt.Sprintf("%.1f%% (threshold: %.0f%%)", currentValue, threshold)

	case models.MetricTypeDiskUsage:
		metricLabel = "Disk usage"
		valueDesc = fmt.Sprintf("%.1f%% (threshold: %.0f%%)", currentValue, threshold)

	case models.MetricTypeLoadAvg:
		metricLabel = "Load average (1m)"
		valueDesc = fmt.Sprintf("%.2f (threshold: %.2f)", currentValue, threshold)

	case models.MetricTypeLoadAvg5:
		metricLabel = "Load average (5m)"
		valueDesc = fmt.Sprintf("%.2f (threshold: %.2f)", currentValue, threshold)

	case models.MetricTypeLoadAvg15:
		metricLabel = "Load average (15m)"
		valueDesc = fmt.Sprintf("%.2f (threshold: %.2f)", currentValue, threshold)

	case models.MetricTypeTemperature:
		metricLabel = "CPU temperature"
		valueDesc = fmt.Sprintf("%.1f°C (threshold: %.0f°C)", currentValue, threshold)

	default:
		metricLabel = metricType
		valueDesc = fmt.Sprintf("%.2f (threshold: %.2f)", currentValue, threshold)
	}

	subject = fmt.Sprintf("[Watchflare Alert] %s — %s exceeded", serverName, metricLabel)
	body = fmt.Sprintf("An alert has been triggered for server %q.\n\n%s: %s\n\nThis alert started at %s.\n\nThis notification was sent by Watchflare.",
		serverName, metricLabel, valueDesc, startedAt.Format(time.RFC1123))
	return
}

// firstUserEmail returns the email of the first registered user.
func firstUserEmail() (string, error) {
	var user models.User
	if err := database.DB.Order("created_at ASC").First(&user).Error; err != nil {
		return "", err
	}
	return user.Email, nil
}
