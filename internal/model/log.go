package model

import "time"

// LogLevel represents the severity of a log line.
type LogLevel string

const (
	LevelDebug   LogLevel = "DEBUG"
	LevelInfo    LogLevel = "INFO"
	LevelWarn    LogLevel = "WARN"
	LevelError   LogLevel = "ERROR"
	LevelUnknown LogLevel = ""
)

// LogEntry represents a single log line from a container.
type LogEntry struct {
	Timestamp   time.Time
	Container   *Container
	Level       LogLevel
	Message     string
	RawLine     string
}

// ParseLevel attempts to detect log level from a message string.
func ParseLevel(msg string) LogLevel {
	// Simple heuristic: look for common level indicators in the first 50 chars.
	prefix := msg
	if len(prefix) > 50 {
		prefix = prefix[:50]
	}

	for i := 0; i < len(prefix)-4; i++ {
		chunk := prefix[i : i+5]
		switch {
		case matches(chunk, "ERROR"), matches(chunk, "FATAL"), matches(chunk, "PANIC"):
			return LevelError
		case matches(chunk, "WARN"), matches(chunk, "WARNI"):
			return LevelWarn
		case matches(chunk, "DEBUG"), matches(chunk, "DEBU"):
			return LevelDebug
		case matches(chunk, "INFO"):
			return LevelInfo
		}
	}

	return LevelUnknown
}

func matches(s, target string) bool {
	if len(s) < len(target) {
		return false
	}
	for i := 0; i < len(target); i++ {
		c := s[i]
		t := target[i]
		// case-insensitive
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		if t >= 'a' && t <= 'z' {
			t -= 32
		}
		if c != t {
			return false
		}
	}
	return true
}
