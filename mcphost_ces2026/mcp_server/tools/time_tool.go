package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Time API structures
type TimeData struct {
	CurrentTime     string            `json:"current_time"`
	Timezone        string            `json:"timezone"`
	UTCOffset       string            `json:"utc_offset"`
	UnixTimestamp   int64             `json:"unix_timestamp"`
	DayOfWeek       string            `json:"day_of_week"`
	DayOfYear       int               `json:"day_of_year"`
	WeekOfYear      int               `json:"week_of_year"`
	IsLeapYear      bool              `json:"is_leap_year"`
	Season          string            `json:"season"`
	FormattedTimes  map[string]string `json:"formatted_times"`
}

// AddTimeTools adds time-related tools to the MCP server
func AddTimeTools(s *server.MCPServer) {
	log.Printf("🔧 Tool Registry: Starting to register time tools...")
	
	// Get current time tool
	timeTool := mcp.NewTool("get_current_time",
		mcp.WithDescription("Get current time and date information"),
		mcp.WithString("timezone",
			mcp.Description("Timezone (e.g.: Asia/Taipei, America/New_York, Europe/London)"),
		),
		mcp.WithString("format",
			mcp.Description("Time format (iso8601, rfc3339, unix, date_only, time_only, datetime_24h, datetime_12h)"),
		),
		mcp.WithBoolean("include_details",
			mcp.Description("Whether to include detailed information (day of week, season, leap year, etc.)"),
		),
		mcp.WithString("test_mode",
			mcp.Description("Test mode: 'normal', 'invalid_timezone', 'future_time', 'past_time', 'edge_timezone', 'format_error'"),
		),
		mcp.WithBoolean("validate_input",
			mcp.Description("Whether to perform strict input validation (default: true)"),
		),
	)

	log.Printf("🔧 Tool Registry: Time tool created")
	log.Printf("   └─ Name: get_current_time")
	log.Printf("   └─ Description: Get current time and date information")
	log.Printf("   └─ Parameters: timezone (optional), format (optional), include_details (optional)")
	log.Printf("   └─ Supported Timezones: %d pre-mapped zones + IANA timezone support", 20)

	s.AddTool(timeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("🕐 MCP Server: Received get_current_time tool call request")

		timezone := request.GetString("timezone", "Asia/Taipei")
		format := request.GetString("format", "iso8601")
		includeDetails := request.GetBool("include_details", true)
		testMode := request.GetString("test_mode", "normal")
		validateInput := request.GetBool("validate_input", true)

		log.Printf("⏰ MCP Server: Processing time query - timezone: %s, format: %s, include details: %t, test mode: %s", 
			timezone, format, includeDetails, testMode)

		// Perform input validation if enabled
		if validateInput {
			if validationErr := validateTimeInput(timezone, format, testMode); validationErr != nil {
				log.Printf("❌ MCP Server: Time input validation failed: %v", validationErr)
				return mcp.NewToolResultError(fmt.Sprintf("Input validation failed: %v", validationErr)), nil
			}
		}

		// Handle test modes
		if testMode != "normal" {
			testResult := handleTimeTestMode(testMode, timezone, format, includeDetails)
			if testResult != nil {
				jsonData, _ := json.MarshalIndent(testResult, "", "  ")
				log.Printf("🧪 MCP Server: Time test mode executed - mode: %s", testMode)
				return mcp.NewToolResultText(string(jsonData)), nil
			}
		}

		// Generate time data
		timeData := getCurrentTimeData(timezone, format, includeDetails)

		// Convert structured data to JSON string
		jsonData, err := json.MarshalIndent(timeData, "", "  ")
		if err != nil {
			log.Printf("❌ MCP Server: Time data serialization failed: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Time data serialization failed: %v", err)), nil
		}

		log.Printf("✅ MCP Server: get_current_time tool call successful, returning %d bytes of time data", len(jsonData))
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	log.Printf("✅ Tool Registry: Successfully registered time tool 'get_current_time'")
	log.Printf("🔧 Tool Registry: Time tools registration completed (1 tool registered)")
}

// getCurrentTimeData generates time data for the specified timezone and format
func getCurrentTimeData(timezone, format string, includeDetails bool) TimeData {
	// Map common city names to standard timezones
	timezoneMap := map[string]string{
		"taipei":    "Asia/Taipei",
		"tokyo":     "Asia/Tokyo",
		"newyork":   "America/New_York",
		"new_york":  "America/New_York",
		"london":    "Europe/London",
		"paris":     "Europe/Paris",
		"berlin":    "Europe/Berlin",
		"moscow":    "Europe/Moscow",
		"sydney":    "Australia/Sydney",
		"shanghai":  "Asia/Shanghai",
		"singapore": "Asia/Singapore",
		"dubai":     "Asia/Dubai",
		"mumbai":    "Asia/Kolkata",
		"delhi":     "Asia/Kolkata",
		"bangkok":   "Asia/Bangkok",
		"jakarta":   "Asia/Jakarta",
		"manila":    "Asia/Manila",
		"seoul":     "Asia/Seoul",
		"hongkong":  "Asia/Hong_Kong",
		"utc":       "UTC",
	}

	// Normalize timezone input
	normalizedTz := strings.ToLower(strings.ReplaceAll(timezone, " ", ""))
	if mappedTz, exists := timezoneMap[normalizedTz]; exists {
		timezone = mappedTz
	}

	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Fallback to UTC if timezone is invalid
		loc = time.UTC
		timezone = "UTC"
	}

	// Get current time in the specified timezone
	now := time.Now().In(loc)

	// Calculate UTC offset
	_, offset := now.Zone()
	offsetHours := offset / 3600
	offsetMinutes := (offset % 3600) / 60
	var utcOffset string
	if offset == 0 {
		utcOffset = "Z"
	} else {
		utcOffset = fmt.Sprintf("%+03d:%02d", offsetHours, offsetMinutes)
	}

	// Create formatted times map
	formattedTimes := map[string]string{
		"iso8601":      now.Format("2006-01-02T15:04:05") + utcOffset,
		"rfc3339":      now.Format("2006-01-02T15:04:05") + utcOffset,
		"unix_time":    fmt.Sprintf("%d", now.Unix()),
		"date_only":    now.Format("2006-01-02"),
		"time_only":    now.Format("15:04:05"),
		"datetime_24h": now.Format("2006-01-02 15:04:05"),
		"datetime_12h": now.Format("2006-01-02 03:04:05 PM"),
		"relative_time": getRelativeTime(now),
	}

	// Calculate day of year and week of year
	dayOfYear := now.YearDay()
	_, weekOfYear := now.ISOWeek()

	// Determine season (Northern Hemisphere)
	season := getSeason(now)

	// Check if it's a leap year
	isLeapYear := time.Date(now.Year(), 2, 29, 0, 0, 0, 0, time.UTC).Month() == 2

	timeData := TimeData{
		CurrentTime:    now.Format("2006-01-02 15:04:05"),
		Timezone:       timezone,
		UTCOffset:      utcOffset,
		UnixTimestamp:  now.Unix(),
		DayOfWeek:      now.Weekday().String(),
		DayOfYear:      dayOfYear,
		WeekOfYear:     weekOfYear,
		IsLeapYear:     isLeapYear,
		Season:         season,
		FormattedTimes: formattedTimes,
	}

	return timeData
}

// getSeason determines the season based on the current date (Northern Hemisphere)
func getSeason(t time.Time) string {
	month := t.Month()
	day := t.Day()

	switch month {
	case 12:
		if day >= 21 {
			return "Winter"
		}
		return "Autumn"
	case 1, 2:
		return "Winter"
	case 3:
		if day >= 20 {
			return "Spring"
		}
		return "Winter"
	case 4, 5:
		return "Spring"
	case 6:
		if day >= 21 {
			return "Summer"
		}
		return "Spring"
	case 7, 8:
		return "Summer"
	case 9:
		if day >= 22 {
			return "Autumn"
		}
		return "Summer"
	case 10, 11:
		return "Autumn"
	default:
		return "Unknown"
	}
}

// getRelativeTime returns a human-readable relative time string
func getRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < 0 {
		diff = -diff
		if diff < time.Minute {
			return "in a few seconds"
		} else if diff < time.Hour {
			return fmt.Sprintf("in %d minutes", int(diff.Minutes()))
		} else if diff < 24*time.Hour {
			return fmt.Sprintf("in %d hours", int(diff.Hours()))
		} else {
			return fmt.Sprintf("in %d days", int(diff.Hours()/24))
		}
	} else {
		if diff < time.Minute {
			return "just now"
		} else if diff < time.Hour {
			return fmt.Sprintf("%d minutes ago", int(diff.Minutes()))
		} else if diff < 24*time.Hour {
			return fmt.Sprintf("%d hours ago", int(diff.Hours()))
		} else {
			return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
		}
	}
}

// validateTimeInput performs input validation for time queries
func validateTimeInput(timezone, format, testMode string) error {
	// Check for dangerous patterns in timezone
	if timezone != "" {
		dangerousPatterns := []string{
			"../", "..\\", "/etc/", "C:\\",
			"<script>", "</script>", "javascript:",
			"'", "\"", "--", "DROP", "DELETE", "INSERT", "UPDATE", "UNION", "SELECT",
			"|", "&", ";", "$(", "`", "rm -rf", "format c:",
		}

		timezoneLower := strings.ToLower(timezone)
		for _, pattern := range dangerousPatterns {
			if strings.Contains(timezoneLower, strings.ToLower(pattern)) {
				return fmt.Errorf("potentially dangerous input detected in timezone: contains '%s'", pattern)
			}
		}

		// Check timezone length
		if len(timezone) > 100 {
			return fmt.Errorf("timezone name too long (max 100 characters)")
		}
	}

	// Validate format parameter
	validFormats := []string{"iso8601", "rfc3339", "unix", "date_only", "time_only", "datetime_24h", "datetime_12h"}
	if format != "" {
		isValidFormat := false
		for _, validFormat := range validFormats {
			if format == validFormat {
				isValidFormat = true
				break
			}
		}
		if !isValidFormat {
			return fmt.Errorf("invalid format: must be one of %v", validFormats)
		}
	}

	// Validate test mode
	validTestModes := []string{"normal", "invalid_timezone", "future_time", "past_time", "edge_timezone", "format_error"}
	isValidTestMode := false
	for _, mode := range validTestModes {
		if testMode == mode {
			isValidTestMode = true
			break
		}
	}
	if !isValidTestMode {
		return fmt.Errorf("invalid test_mode: must be one of %v", validTestModes)
	}

	return nil
}

// handleTimeTestMode handles different test modes for time tool
func handleTimeTestMode(testMode, timezone, format string, includeDetails bool) interface{} {
	switch testMode {
	case "invalid_timezone":
		return map[string]interface{}{
			"error": "Invalid timezone",
			"code":  "INVALID_TIMEZONE",
			"message": fmt.Sprintf("Simulated invalid timezone error for: %s", timezone),
			"test_mode": testMode,
			"timezone": timezone,
			"recoverable": false,
			"suggested_timezones": []string{"America/New_York", "Europe/London", "Asia/Tokyo", "UTC"},
		}

	case "future_time":
		// Simulate future time (year 2038 problem testing)
		futureTime := time.Date(2038, 1, 20, 0, 0, 0, 0, time.UTC)
		return map[string]interface{}{
			"test_mode": testMode,
			"message": "Simulated future time - testing year 2038 edge case",
			"current_time": futureTime.Format(time.RFC3339),
			"timezone": timezone,
			"unix_timestamp": futureTime.Unix(),
			"warning": "This is beyond the 32-bit Unix timestamp limit",
			"is_edge_case": true,
		}

	case "past_time":
		// Simulate past time (Unix epoch edge case)
		pastTime := time.Date(1969, 12, 31, 23, 59, 59, 0, time.UTC)
		return map[string]interface{}{
			"test_mode": testMode,
			"message": "Simulated past time - testing pre-Unix epoch edge case",
			"current_time": pastTime.Format(time.RFC3339),
			"timezone": timezone,
			"unix_timestamp": pastTime.Unix(),
			"warning": "This is before Unix epoch (1970-01-01)",
			"is_edge_case": true,
		}

	case "edge_timezone":
		// Test with edge case timezone
		edgeTimezone := "Pacific/Kiritimati" // +14 UTC, furthest ahead
		location, err := time.LoadLocation(edgeTimezone)
		if err != nil {
			return map[string]interface{}{
				"error": "Edge timezone test failed",
				"code":  "EDGE_TIMEZONE_ERROR",
				"message": fmt.Sprintf("Could not load edge timezone: %s", edgeTimezone),
				"test_mode": testMode,
				"recoverable": false,
			}
		}
		
		now := time.Now().In(location)
		return map[string]interface{}{
			"test_mode": testMode,
			"message": "Simulated edge case timezone - Pacific/Kiritimati (UTC+14)",
			"current_time": now.Format(time.RFC3339),
			"timezone": edgeTimezone,
			"utc_offset": "+14:00",
			"unix_timestamp": now.Unix(),
			"is_edge_case": true,
			"note": "This is the furthest timezone ahead of UTC",
		}

	case "format_error":
		return map[string]interface{}{
			"error": "Format processing error",
			"code":  "FORMAT_ERROR",
			"message": fmt.Sprintf("Simulated format error for format: %s", format),
			"test_mode": testMode,
			"format": format,
			"recoverable": true,
			"suggested_formats": []string{"iso8601", "rfc3339", "unix", "date_only"},
		}

	default:
		return nil // Normal mode, proceed with real time data
	}
}