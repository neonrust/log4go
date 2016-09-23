package log4go

import (
	"io"
	"os"
)

// Handler handles the formatted log events.
type Handler interface {
	Handle(rec *Record) error
	SetFormatter(formatter Formatter)
	GetFormatter() Formatter
	// TODO: SetLevel(level int)  (restrict a handler to a level range)
}

// StreamHandler handles file-based output
type StreamHandler struct {
	writer    io.Writer
	formatter Formatter
	// TODO: level int   (restrict a handler to a level range)
}

// NewStreamHandler returns a new StreamHandler instance using the specified writer.
func NewStreamHandler(writer io.Writer) (Handler, error) {
	return &StreamHandler{
		writer: writer,
	}, nil
}

// NewFileHandler returns a new StreamHandler instance writing to the specified file name.
func NewFileHandler(fileName string, append bool) (Handler, error) {
	flags := os.O_WRONLY | os.O_CREATE
	if append {
		flags |= os.O_APPEND
	}
	writer, err := os.OpenFile(fileName, flags, 0664)
	if err != nil {
		return nil, err
	}
	return &StreamHandler{
		writer: writer,
	}, nil
}

// Handle handles the formatted message.
func (h *StreamHandler) Handle(rec *Record) error {
	formatter := h.GetFormatter()
	message, err := formatter.Format(rec)
	if err != nil {
		return err
	}

	_, err = io.WriteString(h.writer, message)
	if err == nil {
		_, err = io.WriteString(h.writer, "\n")
		if err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}

// SetFormatter sets the handler's Formatter.
func (h *StreamHandler) SetFormatter(formatter Formatter) {
	h.formatter = formatter
}

// GetFormatter resutns the handler's Formatter.
func (h *StreamHandler) GetFormatter() Formatter {
	return h.formatter
}
