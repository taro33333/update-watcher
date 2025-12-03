package util

import (
	"strings"
)

// TruncateText truncates text to maxLength and adds ellipsis
func TruncateText(text string, maxLength int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}

// FormatMultilineText formats multiline text, limiting to maxLines
func FormatMultilineText(text string, maxLength, maxLines int) string {
	text = strings.TrimSpace(text)
	if len(text) > maxLength {
		text = text[:maxLength] + "..."
	}

	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	if len(lines) > maxLines {
		return strings.Join(lines[:maxLines], "\n") + "..."
	}

	return text
}

// GetSeverityEmoji returns the emoji for a given severity level
func GetSeverityEmoji(severity string, emojiMap map[string]string) string {
	if emoji, ok := emojiMap[strings.ToLower(severity)]; ok {
		return emoji
	}
	return "⚠️" // default
}
