package errors

import (
	"fmt"
	"strings"
)

// IsTimestampError checks if an error is a timestamp synchronization issue
func IsTimestampError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Timestamp outside acceptable window")
}

// FormatError formats an error with helpful context
func FormatError(err error, context string) string {
	if IsTimestampError(err) {
		return fmt.Sprintf("%s failed: CLOCK SYNC ERROR - System time is out of sync (>5min difference with backend). "+
			"Fix: Run 'sudo timedatectl set-ntp true' and restart the agent. Original error: %v", context, err)
	}
	return fmt.Sprintf("%s failed: %v", context, err)
}
