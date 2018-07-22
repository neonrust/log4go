// Package log4go provides a simple, tree-like logging facility.
package log4go

import (
	"io"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// BasicConfigOpts is used to supply options to BasicConfig.
type BasicConfigOpts struct {
	FileName   string
	FileAppend interface{}
	WatchFile  bool
	Writer     io.Writer
	Format     string
	Level      Level
	Handlers   []Handler
}

var rootLogger *Logger
var loggersLock = &sync.Mutex{}
var loggers map[string]*Logger

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

		if opts.WatchFile && opts.Writer != nil {
			opts.WatchFile = false
		}

		if opts.Writer != nil {
			defHandler, err = NewStreamHandler(opts.Writer)
		} else if len(opts.FileName) > 0 {
			append := opts.FileAppend == nil || opts.FileAppend.(bool)

			if opts.WatchFile {
				defHandler, err = NewWatchedFileHandler(opts.FileName, append)
			} else {
				defHandler, err = NewFileHandler(opts.FileName, append)
			}
		} else {
			defHandler, err = NewStreamHandler(os.Stderr)
		}
		if err != nil {
			return err
		}
		opts.Handlers = []Handler{defHandler}
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

// Shutdown shuts down all internals of log4go.
func Shutdown() {
	// close all commit channels (depth-first), then wait for the commiters to finish (somehow) ?

	shutdownHandlers(rootLogger)

	runtime.Gosched()
	runtime.GC()
	syscall.Sync()

	// nice synchronization there, uncle Bob!
	time.Sleep(100 * time.Millisecond)
}

func shutdownHandlers(log *Logger) {
	// TODO: this will call handler.Shutdown() multiple times on the same handler
	//   if the same handler is used multiple times
	//   this should be 2-pass:
	//     1. collect all unique handlers
	//     2. then call Shutdown() on those

	if log == nil {
		return
	}
	if log.children != nil {
		// recurse down children, depth-first
		for _, child := range log.children {
			shutdownHandlers(child)
		}
	}

	if log.handlers != nil {
		for _, handler := range log.handlers {
			handler.Shutdown()
		}
	}
}

// GetLogger returns the root logger while GetLogger(name) calls GetLogger(name) on the root logger.
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

func createRootLogger(handlers ...Handler) *Logger {
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