package frequency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrequencyToCron_Daily(t *testing.T) {
	tests := []struct {
		name          string
		startDateTime time.Time
		expectedCron  string
	}{
		{
			name:          "Daily at 9:30 AM",
			startDateTime: time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC),
			expectedCron:  "30 9 * * *",
		},
		{
			name:          "Daily at 2:00 PM",
			startDateTime: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			expectedCron:  "0 14 * * *",
		},
		{
			name:          "Daily at midnight",
			startDateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedCron:  "0 0 * * *",
		},
		{
			name:          "Daily at 11:59 PM",
			startDateTime: time.Date(2024, 1, 1, 23, 59, 0, 0, time.UTC),
			expectedCron:  "59 23 * * *",
		},
		{
			name:          "Daily at 8:05 AM",
			startDateTime: time.Date(2024, 1, 1, 8, 5, 0, 0, time.UTC),
			expectedCron:  "5 8 * * *",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FrequencyToCron("Daily", tt.startDateTime)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCron, result)
		})
	}
}

func TestFrequencyToCron_TimezoneHandling(t *testing.T) {
	// Test that different timezones are converted to UTC
	tests := []struct {
		name          string
		startDateTime time.Time
		expectedCron  string
	}{
		{
			name:          "UTC timezone",
			startDateTime: time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC),
			expectedCron:  "30 9 * * *",
		},
		{
			name:          "EST timezone (UTC-5)",
			startDateTime: time.Date(2024, 1, 1, 9, 30, 0, 0, time.FixedZone("EST", -5*3600)),
			expectedCron:  "30 14 * * *", // 9:30 AM EST = 14:30 UTC
		},
		{
			name:          "PST timezone (UTC-8)",
			startDateTime: time.Date(2024, 1, 1, 9, 30, 0, 0, time.FixedZone("PST", -8*3600)),
			expectedCron:  "30 17 * * *", // 9:30 AM PST = 17:30 UTC
		},
		{
			name:          "JST timezone (UTC+9)",
			startDateTime: time.Date(2024, 1, 1, 9, 30, 0, 0, time.FixedZone("JST", 9*3600)),
			expectedCron:  "30 0 * * *", // 9:30 AM JST = 00:30 UTC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FrequencyToCron("Daily", tt.startDateTime)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCron, result)
		})
	}
}

func TestFrequencyToCron_Unsupported(t *testing.T) {
	tests := []struct {
		name          string
		frequency     string
		expectedError string
	}{
		{
			name:          "Hourly frequency",
			frequency:     "Hourly",
			expectedError: "unsupported frequency",
		},
		{
			name:          "Invalid frequency",
			frequency:     "InvalidFreq",
			expectedError: "unsupported frequency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FrequencyToCron(tt.frequency, time.Now())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestFrequencyToCron_FutureFrequencies(t *testing.T) {
	tests := []struct {
		name          string
		frequency     string
		expectedError string
	}{
		{
			name:          "Weekly frequency",
			frequency:     "Weekly",
			expectedError: "not yet implemented",
		},
		{
			name:          "Monthly frequency",
			frequency:     "Monthly",
			expectedError: "not yet implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FrequencyToCron(tt.frequency, time.Now())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestFrequencyToCron_ValidCronExpression(t *testing.T) {
	validCronExpressions := []string{
		"0 9 * * *",
		"30 14 * * *",
		"0 */2 * * *",
		"0 9 * * 1-5",
	}

	for _, cronExpr := range validCronExpressions {
		t.Run(cronExpr, func(t *testing.T) {
			result, err := FrequencyToCron(cronExpr, time.Now())
			require.NoError(t, err)
			assert.Equal(t, cronExpr, result, "Valid cron expressions should be returned as-is")
		})
	}
}

func TestFrequencyToCron_InvalidCronExpression(t *testing.T) {
	invalidCronExpressions := []string{
		"invalid",
		"a b c d e",
		"* 25 * * *", // Invalid hour (hour 25 doesn't exist)
		"* 61 * * *", // Invalid minute (minute 61 doesn't exist)
	}

	for _, cronExpr := range invalidCronExpressions {
		t.Run(cronExpr, func(t *testing.T) {
			_, err := FrequencyToCron(cronExpr, time.Now())
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported frequency")
		})
	}
}

func TestIsConvertibleFrequency(t *testing.T) {
	tests := []struct {
		name      string
		frequency string
		expected  bool
	}{
		{
			name:      "Daily is convertible",
			frequency: "Daily",
			expected:  true,
		},
		{
			name:      "Weekly is convertible",
			frequency: "Weekly",
			expected:  true,
		},
		{
			name:      "Monthly is convertible",
			frequency: "Monthly",
			expected:  true,
		},
		{
			name:      "Cron expression is not convertible",
			frequency: "0 9 * * *",
			expected:  false,
		},
		{
			name:      "Random string is not convertible",
			frequency: "random",
			expected:  false,
		},
		{
			name:      "Hourly is not convertible",
			frequency: "Hourly",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConvertibleFrequency(tt.frequency)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidCronExpression(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		{
			name:     "Valid daily cron",
			expr:     "0 9 * * *",
			expected: true,
		},
		{
			name:     "Valid hourly cron",
			expr:     "0 */2 * * *",
			expected: true,
		},
		{
			name:     "Valid weekdays cron",
			expr:     "0 9 * * 1-5",
			expected: true,
		},
		{
			name:     "Invalid expression",
			expr:     "invalid",
			expected: false,
		},
		{
			name:     "User-friendly frequency",
			expr:     "Daily",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidCronExpression(tt.expr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
