package log4go

import "fmt"

// Level is a typed logging level.
type Level int

// Log levels.
const (
	// INHERIT inherits from parent logger.
	INHERIT Level = iota
	// TRACE log level.
	TRACE
	// DEBUG log level.
	DEBUG
	// INFO log level.
	INFO
	// WARNING log level.
	WARNING
	// ERROR log level.
	ERROR
	// FATAL log level - globally unrecoverable error (also does os.Exit(1)).
	FATAL
)

var levelToName = map[Level]string{
	INHERIT: "INHERIT",
	TRACE:   "TRACE",
	DEBUG:   "DEBUG",
	INFO:    "INFO",
	WARNING: "WARNING",
	ERROR:   "ERROR",
	FATAL:   "FATAL",
}

// LevelName returns the textual representation of the level.
func LevelName(l Level) string {
	if name, ok := levelToName[l]; ok {
		return name
	}
	return fmt.Sprintf("<Level:%d>", l)
}
