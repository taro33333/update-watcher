// Package util provides utility functions for date parsing,
// text formatting, and other common operations.
package util

import (
	"fmt"
	"log"
	"time"
)

// SupportedDateFormats contains all date formats we can parse
var SupportedDateFormats = []string{
	time.RFC3339,
	time.RFC1123Z,
	time.RFC1123,
	"2006-01-02T15:04:05.999999999Z07:00", // RFC3339 with nanoseconds
	"2006-01-02T15:04:05.999",             // ISO 8601 with milliseconds (NVD format)
	"2006-01-02T15:04:05Z",
	"2006-01-02", // Date only format (used by Debian)
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"Mon, 02 Jan 2006 15:04:05 MST",
}

// ParseDate attempts to parse a date string using multiple formats
func ParseDate(dateStr string) (time.Time, error) {
	for _, format := range SupportedDateFormats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse date: %s", dateStr)
}

// IsRecent checks if a date is within the specified hours ago
func IsRecent(dateStr string, hoursAgo int) bool {
	parsedTime, err := ParseDate(dateStr)
	if err != nil {
		log.Printf("Warning: %v", err)
		return false
	}

	cutoff := time.Now().Add(-time.Duration(hoursAgo) * time.Hour)
	return parsedTime.After(cutoff)
}
