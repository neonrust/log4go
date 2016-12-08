// Package log4go provides a simple, tree-like logging facility.

package log4go

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
	"runtime"
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
	FileName   string
	FileAppend interface{}
	Writer     io.Writer
	Format     string
	Level      int
	Handlers   []Handler
}

// Logger objects.
type Logger struct {
	name           string
	level          int
	handlers       []Handler
	parent         *Logger
	children       []*Logger
}

// Log levels.
const (
	// NOTSET log level (inherits from parent).
	NOTSET = 0
	// FATAL log level (also does os.Exit(1)).
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
	NOTSET:     "NOTSET",
	FATAL:      "FATAL",
	ERROR:      "ERROR",
	WARNING:    "WARNING",
	INFO:       "INFO",
	DEBUG:      "DEBUG",
}


func LevelName(level int) string {
	if name, exists := levelToName[level]; ! exists {
		return fmt.Sprintf("%d", level)
	} else {
		return name
	}
}

var loggers map[string]*Logger
var rootLogger *Logger
var loggersLock = &sync.Mutex{}

var recordPool sync.Pool

func init() {
	recordPool = sync.Pool{
		New: func() interface{} {
			return &Record{}
		},
	}

	loggers = make(map[string]*Logger)
}

// BasicConfig sets up a simple configuration of the logging system.
func BasicConfig(opts BasicConfigOpts) error {
	loggersLock.Lock()
	defer loggersLock.Unlock()

	// remove any/all created Logger, Handler and Formatter instances
	Shutdown()
	loggers = map[string]*Logger{}
	rootLogger = nil

	var err error

	if opts.Level == 0 {
		opts.Level = WARNING
	}
	if len(opts.Format) == 0 {
		opts.Format = "{time} {name<20} {level<8} {message}"
	}

	if len(opts.Handlers) == 0 {
		var defHandler Handler
		var err error

		if opts.Writer != nil {
			defHandler, err = NewStreamHandler(opts.Writer)
		} else if len(opts.FileName) > 0 {
			append := opts.FileAppend == nil || opts.FileAppend.(bool)
			defHandler, err = NewFileHandler(opts.FileName, append)
		} else {
			defHandler, err = NewStreamHandler(os.Stderr)
		}
		if err != nil {
			return err
		}
		opts.Handlers = []Handler { defHandler }
	}

	// use a default formatter if the specified handler(s) has none
	var defFormatter Formatter
	for _, handler := range opts.Handlers {
		if handler.Formatter() == nil {
			if defFormatter == nil { // create a default formatter
				defFormatter, err = NewTemplateFormatter(opts.Format)
				if err != nil {
					return err
				}
			}
			handler.SetFormatter(defFormatter)
		}
	}

	rootLogger = createRootLogger(opts.Handlers...)
	rootLogger.SetLevel(opts.Level)

	return nil
}

func Shutdown() {
	// close all commit channels (depth-first), then wait for the commiters to finish (somehow) ?

	shutdownHandlers(rootLogger)

	runtime.Gosched()
	runtime.GC()

	time.Sleep(100*time.Millisecond)
}

func shutdownHandlers(log *Logger) {
	if log == nil {
		return
	}
	if log.children != nil {
		for _, child := range log.children {
			shutdownHandlers(child)
		}
	}

	if log.handlers != nil {
		for _, handler := range log.handlers {
			//fmt.Fprintf(os.Stderr, "[%s] shutting down handler: %p\n", log.name, handler)
			handler.Shutdown()
		}
	} else {
		//fmt.Fprintf(os.Stderr, "[%s] no handlers to shut down\n", log.name)
	}
}

// GetLogger() returns the root logger while GetLogger(name) calls GetLogger(name) on the root logger.
func GetLogger(name ...string) *Logger {
	if len(name) > 0 && !(len(name) == 1 && name[0] == "root") {
		return GetLogger().GetLogger(name[0])
	}

	// get/create the root logger
	loggersLock.Lock()
	defer loggersLock.Unlock()

	if rootLogger == nil {
		rootLogger = createRootLogger()
	}

	return rootLogger
}


func newLogger(parent *Logger, name string, level int, handlers... Handler) *Logger {
	// use: sync.Pool ?
	log := &Logger{
		name:  name,
		level: level,
	}
	if parent != nil {
		log.parent = parent
		if parent.children == nil {
			parent.children = make([]*Logger, 0, 5)
		}
		parent.children = append(parent.children, log)
	}

	if len(handlers) > 0 {
		log.handlers = handlers
	}

	return log
}

func createRootLogger(handlers... Handler) *Logger {
	//fmt.Println("creating root logger: %d handlers", len(handlers))

	if len(handlers) == 0 {
		handler, _ := NewStreamHandler(os.Stderr)
		formatter, _ := NewTemplateFormatter("{time} {name} {level} {message}")
		handler.SetFormatter(formatter)
		handlers = []Handler{handler}
	}

	//fmt.Printf("root logger, h = %p\n", handler)

	logger := newLogger(nil, "", WARNING, handlers...)

	return logger
}

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
		//fmt.Printf("'%s' creating sub-logger: %s\n", l.name, loggerName)

		// create sub-logger
		logger = newLogger(l, loggerName, NOTSET)

		loggers[loggerName] = logger
	}

	return logger
}

// SetLevel sets the logging level of the logger.
func (l *Logger) SetLevel(level int) {
	l.level = level
}

// Level returns the loggers (effective) level.
func (l *Logger) Level() int {
	for l.level == NOTSET {
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
	l.handlers = append(l.handlers, handler)
}

// ReplaceHandlers replaces all added handler with a new handler.
func (l *Logger) ReplaceHandlers(handler Handler) {
	l.RemoveHandlers()
	l.AddHandler(handler)
}

func (l *Logger) RemoveHandlers() {
	l.handlers = []Handler{}
}

// Handlers returns all handlers used by this logger (i.e. this and all its parents' handlers).
func (l *Logger) Handlers() []Handler {
	handlers := make([]Handler, 0, 10)
	logger := l
	for logger != nil {
		if logger.handlers != nil && len(logger.handlers) > 0 {
			handlers = append(handlers, logger.handlers...)
		}
		logger = logger.parent
	}
	return handlers
}

// Log submits a log message using specific level and message.
func (l *Logger) Log(level int, message string, args ...interface{}) {
	ourLevel := l.Level()

	if level < ourLevel {
		return
	}

	var record *Record

	// traverse up this logger's ancestors, calling all handlers along the way
	logger := l
	for logger != nil {
		if len(logger.handlers) > 0 { // we need handlers!
			if record == nil {
				record = recordPool.Get().(*Record)

				record.Time = time.Now()
				record.Name = l.name
				record.Level = level
				record.Message = fmt.Sprintf(message, args...)
			}

			// invoke all handlers
			for _, handler := range logger.handlers {
				handler.Handle(record)
			}
		}
		logger = logger.parent
	}

	if record != nil {
		recordPool.Put(record)
	}
}

// Fatal logs message with FATAL level (also does os.Exit(1))
func (l *Logger) Fatal(message string, args ...interface{}) {
	l.Log(FATAL, message, args...)

	Shutdown()
	os.Exit(1)
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

	Shutdown()

	os.Exit(1)
}
