package log4go

import (
	"io"
	"os"
	"fmt"
	"runtime"
)

// Handler handles the formatted log events.
type Handler interface {
	Handle(rec *Record) error
	SetFormatter(formatter Formatter)
	Formatter() Formatter
	SetLevel(level int)
	Level() int
	Shutdown()
}

// StreamHandler handles file-based output
type StreamHandler struct {
	writer        io.Writer
	formatter     Formatter
	level         int
	commitChannel chan Record
	committerDone chan bool
	shutdown      bool
}

// NewStreamHandler returns a new StreamHandler instance using the specified writer.
func NewStreamHandler(writer io.Writer) (*StreamHandler, error) {
	handler := &StreamHandler{
		writer:        writer,
		commitChannel: make(chan Record, 100),
		committerDone: make(chan bool),
		shutdown:      false,
	}
	//fmt.Printf("[%p] NewStreamHandler\n", handler)

	go handler.committer()

	return handler, nil
}

// NewFileHandler returns a new StreamHandler instance writing to the specified file name.
func NewFileHandler(fileName string, append bool) (*StreamHandler, error) {
	flags := os.O_WRONLY | os.O_CREATE
	if append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	writer, err := os.OpenFile(fileName, flags, 0664)
	if err != nil {
		return nil, err
	}
	return NewStreamHandler(writer)
}

func (h *StreamHandler) SetLevel(level int) {
	h.level = level
}

func (h *StreamHandler) Level() int {
	return h.level
}

// Handle handles the formatted message.
func (h *StreamHandler) Handle(rec *Record) error {
	if ! h.shutdown {
		h.commitChannel <- *rec
		//fmt.Printf("[%p] SteamHandler.Handle(): sent record: '%s'\n", h, rec.Message)
	}
	return nil
}

func (h *StreamHandler) Shutdown() {
	if ! h.shutdown {
		h.shutdown = true
		close(h.commitChannel)
		<-h.committerDone
		runtime.GC()
	}
}

func (h *StreamHandler) committer() {
	//fmt.Printf("[%p] SteamHandler.committer(): running, ch = %p\n", h, h.commitChannel)

	for rec := range h.commitChannel {
		//fmt.Printf("[%p] SteamHandler.committer(): got record, message: %s\n", h, rec.Message)

		msg, err := h.Formatter().Format(&rec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "log4go.SteamHandler: formatter error %v\n", err)
			return
		}

		msg = append(msg, '\n')
		if _, err = h.writer.Write(msg); err != nil {
			fmt.Fprintf(os.Stderr, "log4go.SteamHandler: write error: %v\n", err)
		}

		//fmt.Printf("[%p] SteamHandler.comitter(): wrote message: '%s'\n", h, rec.Message)
	}
	//fmt.Printf("[%p] SteamHandler.comitter(): exit\n", h)
	h.committerDone <- true
}

// SetFormatter sets the handler's Formatter.
func (h *StreamHandler) SetFormatter(formatter Formatter) {
	h.formatter = formatter
}

// Formatter resutns the handler's Formatter.
func (h *StreamHandler) Formatter() Formatter {
	return h.formatter
}
