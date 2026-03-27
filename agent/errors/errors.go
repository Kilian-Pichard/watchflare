package errors

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsTimestampError checks if an error is a timestamp synchronization issue.
// The backend returns codes.InvalidArgument with a specific message when the
// agent clock is more than 5 minutes out of sync.
func IsTimestampError(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	return st.Code() == codes.InvalidArgument &&
		strings.Contains(st.Message(), "Timestamp outside acceptable window")
}

// FormatError formats an error with helpful context
func FormatError(err error, context string) string {
	if IsTimestampError(err) {
		return fmt.Sprintf("%s failed: CLOCK SYNC ERROR - System time is out of sync with the backend (>5min difference). "+
			"Ensure the system clock is synchronized and restart the agent. Original error: %v", context, err)
	}
	return fmt.Sprintf("%s failed: %v", context, err)
}
