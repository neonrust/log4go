package log4go

import (
	"io"
	"os"
)

// Handler handles the formatted log events.
type Handler interface {
	Handle(rec *Record) error
	SetFormatter(formatter Formatter)
	Formatter() Formatter
	// TODO: SetLevel(level int)  (restrict a handler to a level range)
}

// StreamHandler handles file-based output
type StreamHandler struct {
	writer       io.Writer
	formatter    Formatter
	// TODO: level int   (restrict a handler to a level range)
	//commitChannel chan *[]byte
}

// NewStreamHandler returns a new StreamHandler instance using the specified writer.
func NewStreamHandler(writer io.Writer) (Handler, error) {
	handler := &StreamHandler{
		writer:       writer,
		//commitChannel: make(chan *[]byte, 100),
	}

	//go handler.committer(handler.commitChannel)

	return handler, nil
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
	return NewStreamHandler(writer)
}

// Handle handles the formatted message.
func (h *StreamHandler) Handle(rec *Record) error {
	msg, err := h.Formatter().Format(rec)
	if err != nil {
		return err
	}
/*
	out.Printf("writing message: '%s'\n", msg)
	h.commitChannel <- &msg
	out.Printf("message written")
	return nil
}

func (h *StreamHandler) committer(ch <-chan *[]byte) {
	out.Println("waiting for messages...")

	for msg := range ch {
		if msg == nil { // got "kill pill"
			break
		}

		out.Println("committing message")
*/
		msg = append(msg, '\n')
		_, err = h.writer.Write(msg)
/*
	}
	out.Println("committer exit")
*/
return nil
}

// SetFormatter sets the handler's Formatter.
func (h *StreamHandler) SetFormatter(formatter Formatter) {
	h.formatter = formatter
}

// Formatter resutns the handler's Formatter.
func (h *StreamHandler) Formatter() Formatter {
	return h.formatter
}
