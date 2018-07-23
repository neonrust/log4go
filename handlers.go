package log4go

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

// Handler handles the formatted log events.
type Handler interface {
	Handle(rec *Record) error
	SetFormatter(formatter Formatter)
	Formatter() Formatter
	SetLevel(level Level)
	Level() Level
	Shutdown()
}

// StreamHandler handles stream-based output.
type StreamHandler struct {
	writer        io.Writer
	formatter     Formatter
	level         Level
	commitChannel chan Record
	committerStop chan struct{}
	shutdown      bool
}

// NewStreamHandler returns a new StreamHandler instance using the specified writer.
func NewStreamHandler(w io.Writer) (*StreamHandler, error) {
	handler := &StreamHandler{
		writer:        w,
		commitChannel: make(chan Record, 100),
		committerStop: make(chan struct{}),
		shutdown:      false,
	}

	go handler.committer()

	return handler, nil
}

// NewFileHandler returns a new StreamHandler instance writing to the specified file name.
func NewFileHandler(filename string, append bool) (*StreamHandler, error) {
	flags := os.O_WRONLY | os.O_CREATE
	if append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	writer, err := os.OpenFile(filename, flags, 0664)
	if err != nil {
		return nil, err
	}
	return NewStreamHandler(writer)
}

// SetLevel sets the level the handler will (at least) handle.
func (h *StreamHandler) SetLevel(level Level) {
	h.level = level
}

// Level returns the level previously set (or NOTSET if not set).
func (h *StreamHandler) Level() Level {
	return h.level
}

// Handle handles the formatted message.
func (h *StreamHandler) Handle(rec *Record) error {
	if !h.shutdown { // TODO: should use mutex (to avoid writing to closed channel)
		h.commitChannel <- *rec
	}
	return nil
}

// Shutdown shuts down the handler.
func (h *StreamHandler) Shutdown() {
	if !h.shutdown {
		h.shutdown = true
		h.committerStop <- struct{}{} // unbuffered; when this returns the committer has stopped

		close(h.committerStop)
		close(h.commitChannel) // TODO: see Handle() or never close this channel?
	}
}

func (h *StreamHandler) onPreWrite() {
	// default does nothing
}

func (h *StreamHandler) committer() {
	for {
		select {
		case rec := <-h.commitChannel:
			msg, err := ff.Format(&rec)
			if err != nil {
				fmt.Fprintf(os.Stderr, "log4go.StreamHandler: formatter error %v\n", err)
				continue
			}

			msg = append(msg, '\n')

			h.onPreWrite()

			if _, err = h.writer.Write(msg); err != nil {
				fmt.Fprintf(os.Stderr, "log4go.StreamHandler: write error: %v\n", err)
			}

		case <-h.committerStop:
			break
		}
	}
}

// SetFormatter sets the handler's Formatter.
func (h *StreamHandler) SetFormatter(formatter Formatter) {
	if formatter == nil {
		fmt.Fprintln(os.Stderr, "log4go.StreamHandler: setting nil formatter")
	}

	h.formatter = formatter
}

// Formatter resutns the handler's Formatter.
func (h *StreamHandler) Formatter() Formatter {
	return h.formatter
}

// WatchedFileHandler watches the log file: if file is moved the filename is re-opened.
type WatchedFileHandler struct {
	*StreamHandler

	fp       *os.File // we want to use Sync()
	filename string
	append   bool
	inode    uint64
	dev      uint64
}

// NewWatchedFileHandler returns a new WatchedFileHandler instance writing to the specified file name.
func NewWatchedFileHandler(filename string, append bool) (*WatchedFileHandler, error) {
	wfh := &WatchedFileHandler{
		filename: filename,
		append:   append,
	}

	if err := wfh.open(); err != nil {
		return nil, err
	}

	s, err := NewStreamHandler(wfh.fp)
	if err != nil {
		return nil, err
	}

	wfh.StreamHandler = s

	return wfh, nil
}

// called when committer is about to write a message
func (h *WatchedFileHandler) onPreWrite() {
	if h.fileHasMoved() {
		// just re-open, with same filename
		h.close()
		h.open()
		h.writer = h.fp
	}
}

func (h WatchedFileHandler) fileHasMoved() bool {
	// TODO: use fsnotify to detect when the file has moved?

	dev, ino := h.statFile()
	// in case statFile() returns (0, 0) this will return true also
	return dev != h.dev || ino != h.inode
}

func (h *WatchedFileHandler) close() {
	if h.fp != nil {
		h.fp.Sync()
		h.fp.Close()
		h.fp = nil
		h.writer = nil
	}
}

func (h *WatchedFileHandler) open() error {
	flags := os.O_WRONLY | os.O_CREATE
	if h.append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	fp, err := os.OpenFile(h.filename, flags, 0664)
	if err != nil {
		return err
	}
	h.writer = fp

	h.dev, h.inode = h.statFile()

	return nil
}

func (h WatchedFileHandler) statFile() (uint64, uint64) {
	info, _ := os.Stat(h.filename)
	if stat, ok := info.Sys().(*syscall.Stat_t); !ok {
		return 0, 0
	} else {
		return stat.Dev, stat.Ino
	}
}
