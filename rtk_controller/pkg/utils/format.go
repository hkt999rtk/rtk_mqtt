package utils

import (
	"fmt"
	"strings"
	"time"
)

// FormatTimeAgo formats a time as "X ago" format
func FormatTimeAgo(t time.Time) string {
	duration := time.Since(t)
	
	if duration < time.Minute {
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	} else {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
}

// FormatDuration formats a duration in human readable format
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	} else {
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd%dh", days, hours)
	}
}

// FormatBytes formats bytes in human readable format
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}


// TruncateString truncates a string to maxLen characters
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	
	if maxLen <= 3 {
		return s[:maxLen]
	}
	
	return s[:maxLen-3] + "..."
}

// FormatJSON formats JSON string with proper indentation
func FormatJSON(data []byte) ([]byte, error) {
	// This would use json.Indent in a real implementation
	return data, nil
}

// ParseKeyValue parses key=value strings into a map
func ParseKeyValue(args []string) map[string]string {
	result := make(map[string]string)
	
	for _, arg := range args {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			key := strings.TrimPrefix(parts[0], "--")
			result[key] = parts[1]
		}
	}
	
	return result
}

// FormatTable creates a simple table format
func FormatTable(headers []string, rows [][]string) string {
	if len(rows) == 0 {
		return "No data to display"
	}
	
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	
	// Build table
	var result strings.Builder
	
	// Header
	for i, header := range headers {
		if i > 0 {
			result.WriteString("  ")
		}
		result.WriteString(fmt.Sprintf("%-*s", widths[i], header))
	}
	result.WriteString("\n")
	
	// Separator
	for i, width := range widths {
		if i > 0 {
			result.WriteString("  ")
		}
		result.WriteString(strings.Repeat("-", width))
	}
	result.WriteString("\n")
	
	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				result.WriteString("  ")
			}
			if i < len(widths) {
				result.WriteString(fmt.Sprintf("%-*s", widths[i], cell))
			} else {
				result.WriteString(cell)
			}
		}
		result.WriteString("\n")
	}
	
	return result.String()
}