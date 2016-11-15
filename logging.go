package log4go

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

/*

 - write to file and optionally stdout/stderr
 - global level
 - specific format; essentially: time, name, level, message.
 - multiple loggers with:
    - name
    - level

*/

// BasicConfigOpts is used to supply options to BasicConfig.
type BasicConfigOpts struct {
	FileName string
	Writer   io.Writer
	Format   string
	Level    int
	Handlers []Handler
}

var loggers map[string]*Logger
var loggersLock = &sync.Mutex{}

// BasicConfig sets up a simple configuration of the logging system.
func BasicConfig(opts BasicConfigOpts) error {
	loggersLock.Lock()
	defer loggersLock.Unlock()

	loggers = map[string]*Logger{} // replace any already created Logger

	var err error

	if opts.Writer == nil {
		if len(opts.FileName) == 0 {
			opts.Writer = os.Stdout
		} else {
			opts.Writer, err = os.OpenFile(opts.FileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0664)
			if err != nil {
				return err
			}
		}
	}
	if opts.Level == 0 {
		opts.Level = WARNING
	}
	if len(opts.Format) == 0 {
		opts.Format = "{time} {name<20} {level<8} {message}"
	}
	var formatter Formatter
	formatter, err = NewTemplateFormatter(opts.Format)
	if err != nil {
		return err
	}

	if len(opts.Handlers) == 0 {
		defHandler := &StreamHandler{
			writer:    opts.Writer,
			formatter: formatter,
		}
		opts.Handlers = append(opts.Handlers, defHandler)
	} else {
		// if any specified handler has no formatter, use the one created above
		for _, handler := range opts.Handlers {
			if handler.GetFormatter() == nil {
				handler.SetFormatter(formatter)
			}
		}
	}

	logger := &Logger{
		name:     "", // i.e. root
		level:    opts.Level,
		handlers: opts.Handlers,
	}

	loggers[""] = logger

	return nil
}

// GetLogger returns the root logger.
func GetLogger(name ...string) *Logger {
	if len(name) > 0 && !(len(name) == 1 && name[0] == "root") {
		return GetLogger().GetLogger(name[0])
	}

	loggerName := "" // i.e. root

	// get/create the root logger
	loggersLock.Lock()
	defer loggersLock.Unlock()

	logger, exist := loggers[loggerName]
	if !exist {
		//out.Println("creating root logger")
		defHandler := &StreamHandler{writer: os.Stdout}
		formatter, _ := NewTemplateFormatter("{time} {name} {level} {message}")
		defHandler.SetFormatter(formatter)

		logger = &Logger{
			name:           loggerName,
			level:          WARNING,
			defaultHandler: true,
		}
		logger.handlers = append(logger.handlers, defHandler)

		loggers[""] = logger

		//} else {
		//	out.Println("getting root logger")
	}

	return logger
}

const lvlInherit = 0

// GetLogger returns a sub-logger (inherits traits from parent).
func (l *Logger) GetLogger(subName string) *Logger {
	// get/create a sub-logger

	loggerName := l.name
	if len(loggerName) > 0 {
		loggerName += "/"
	}
	loggerName += subName

	loggersLock.Lock()
	defer loggersLock.Unlock()

	logger, exists := loggers[loggerName]
	if !exists {
		//out.Printf("creating sub logger: %s\n", loggerName)
		logger = &Logger{
			name:   loggerName,
			level:  lvlInherit,
			parent: l,
		}
		//} else {
		//	out.Printf("getting sub logger: %s\n", loggerName)
	}
	loggers[loggerName] = logger

	return logger
}

// SetLevel sets the logging level of the logger.
func (l *Logger) SetLevel(level int) {
	l.level = level
}

// GetLevel returns the loggers (effective) level.
func (l *Logger) GetLevel() int {
	for l.level == lvlInherit {
		if l.parent != nil {
			l = l.parent
		}
		if l == nil {
			return 0
		}
	}
	return l.level
}

// AddHandler adds a log record handler.
func (l *Logger) AddHandler(handler Handler) {
	if l.defaultHandler {
		l.defaultHandler = false
		l.handlers = []Handler{}
	}
	l.handlers = append(l.handlers, handler)
}

// ReplaceHandlers replaces all added handler with a new handler.
func (l *Logger) ReplaceHandlers(handler Handler) {
	if l.defaultHandler {
		l.defaultHandler = false
		l.handlers = []Handler{}
	}
	l.AddHandler(handler)
}

// GetHandlers return a Loggers handlers
// func (l *Logger) GetHandlers() []Handler {
// 	return l.handlers
// }

// Logger objects.
type Logger struct {
	name           string
	level          int
	defaultHandler bool
	handlers       []Handler
	parent         *Logger
}

// Log submits a log message using specific level and message.
func (l *Logger) Log(level int, message string, args ...interface{}) {
	ourLevel := l.GetLevel()

	if level >= ourLevel {
		message = fmt.Sprintf(message, args...)

		record := &Record{
			Name:    l.name,
			Time:    time.Now(),
			Level:   level,
			Message: message,
		}

		// build a Logger tree, starting with ourselves
		loggers := []*Logger{l}
		logger := l
		for logger.parent != nil {
			logger = logger.parent
			loggers = append([]*Logger{logger}, loggers...)
		}

		for _, logger := range loggers {
			for _, handler := range logger.handlers {
				handler.Handle(record)
			}
		}
	}

	if level == FATAL {
		os.Exit(1)
	}
}

// Fatal logs message with FATAL level (also does os.Exit(1))
func (l *Logger) Fatal(message string, args ...interface{}) {
	l.Log(FATAL, message, args...)
}

// Error logs message with ERROR level.
func (l *Logger) Error(message string, args ...interface{}) {
	l.Log(ERROR, message, args...)
}

// Warning logs message with WARNING level.
func (l *Logger) Warning(message string, args ...interface{}) {
	l.Log(WARNING, message, args...)
}

// Info logs message with INFO level.
func (l *Logger) Info(message string, args ...interface{}) {
	l.Log(INFO, message, args...)
}

// Debug logs message with DEBUG level.
func (l *Logger) Debug(message string, args ...interface{}) {
	l.Log(DEBUG, message, args...)
}

func (l *Logger) Crash(err interface{}, stack []byte, buildPath string) {
	// stack will always contain "useless" levels, e.g.:
	// runtime/debug.Stack(0xc4200ed7f0, 0x6aeee0, 0xc420101120)
	//    (location of call to debug.Stack())
	// main.main.func1(0xc420016f00)
	//    (location of deferred function that called recover())
	// panic(0x6aeee0, 0xc420101120)
	//    (location of call of panic())

	lines := make([]string, 0, 10)
	skipped := 0

	// skip until we find "panic("
	reader := strings.NewReader(string(stack))
	for scanner := bufio.NewScanner(reader); scanner.Scan(); {
		line := scanner.Text()
		if skipped > 0 || strings.HasPrefix(line, "panic(") {
			skipped++
			if skipped >= 3 {
				if strings.HasPrefix(line, "\t"+buildPath) {
					line = "   " + line[2+len(buildPath):]
				}
				lines = append(lines, line)
			}
		}
	}

	l.Error("CRASH: %v", err)
	for _, line := range lines {
		l.Error(line)
	}
	os.Exit(1)
}

const (
	// FATAL log level. (also does os.Exit(1))
	FATAL = 50
	// ERROR log level.
	ERROR = 40
	// WARNING log level.
	WARNING = 30
	// INFO log level.
	INFO = 20
	// DEBUG log level.
	DEBUG = 10
)

var levelToName = map[int]string{
	FATAL:      "FATAL",
	ERROR:      "ERROR",
	WARNING:    "WARNING",
	INFO:       "INFO",
	DEBUG:      "DEBUG",
	lvlInherit: "UNSET",
}

// Record is a log message container.
type Record struct {
	Time    time.Time
	Name    string
	Level   int
	Message string
}
