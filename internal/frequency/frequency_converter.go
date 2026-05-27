package frequency

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// ConvertibleFrequency represents frequencies that can be converted to cron expressions
type ConvertibleFrequency string

const (
	FrequencyDaily   ConvertibleFrequency = "Daily"
	FrequencyWeekly  ConvertibleFrequency = "Weekly"  // Future implementation
	FrequencyMonthly ConvertibleFrequency = "Monthly" // Future implementation
)

// FrequencyToCron converts user-friendly frequency + startDateTime to cron expression
// Example: "Daily" + 2024-01-01T09:30:00Z → "30 9 * * *"
func FrequencyToCron(frequency string, startDateTime time.Time) (string, error) {
	freq := ConvertibleFrequency(frequency)

	switch freq {
	case FrequencyDaily:
		return dailyToCron(startDateTime), nil
	case FrequencyWeekly:
		return "", fmt.Errorf("weekly frequency not yet implemented")
	case FrequencyMonthly:
		return "", fmt.Errorf("monthly frequency not yet implemented")
	default:
		// Already a cron expression or invalid - check if it's a valid cron expression
		if IsValidCronExpression(frequency) {
			return frequency, nil
		}
		return "", fmt.Errorf("unsupported frequency: %s (must be 'Daily' or valid cron expression)", frequency)
	}
}

// dailyToCron converts "Daily" + startDateTime to cron expression
// Uses UTC for consistent scheduling across timezones
func dailyToCron(startDateTime time.Time) string {
	// Convert to UTC to ensure consistent scheduling
	utcTime := startDateTime.UTC()
	minute := utcTime.Minute()
	hour := utcTime.Hour()
	return fmt.Sprintf("%d %d * * *", minute, hour)
}

// IsConvertibleFrequency checks if a frequency string can be converted (e.g., "Daily", "Weekly")
func IsConvertibleFrequency(frequency string) bool {
	switch ConvertibleFrequency(frequency) {
	case FrequencyDaily, FrequencyWeekly, FrequencyMonthly:
		return true
	default:
		return false
	}
}

// IsValidCronExpression checks if a string is a valid cron expression
func IsValidCronExpression(schedule string) bool {
	_, err := cron.ParseStandard(schedule)
	return err == nil
}
