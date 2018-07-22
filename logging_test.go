package log4go

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOne(t *testing.T) {
	var buf bytes.Buffer

	BasicConfig(BasicConfigOpts{
		Level:  DEBUG,
		Writer: &buf,
	})

	log := GetLogger("test")

	for idx := 0; idx < 100; idx++ {
		log.Info("test message %d", idx)
	}

	Shutdown()

	foundLast := false
	scanner := bufio.NewScanner(&buf)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "test message 99") {
			foundLast = true
		}
	}

	if !foundLast {
		t.Errorf("last message not found (output len: %d)", buf.Len())
	}
}

func TestOnlyChildLogger(t *testing.T) {

	GetLogger().RemoveHandlers() // no logging from root logger

	var buf bytes.Buffer
	fp := &buf
	//fp, _ := os.OpenFile("TestOnlyChildLogger.log", os.O_CREATE | os.O_TRUNC, 0664)
	handler, _ := NewStreamHandler(fp)
	fmt, _ := NewTemplateFormatter("{name} {level} {message}")
	handler.SetFormatter(fmt)
	log := GetLogger("test")
	log.AddHandler(handler)
	log.SetLevel(INFO) // otherwise it will inherit root's WARNING (the default)

	log.Info("test message 99")

	Shutdown()

	foundLast := false
	scanner := bufio.NewScanner(&buf)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "test message 99") {
			foundLast = true
		}
	}

	if !foundLast {
		t.Errorf("last message not found (output len: %d)", buf.Len())
	}
}

func TestLevelFilter(t *testing.T) {
	var buf bytes.Buffer

	BasicConfig(BasicConfigOpts{
		Level:  WARNING,
		Writer: &buf,
	})

	log := GetLogger("test")

	log.Info("this will never appear in the log")

	Shutdown()

	if buf.Len() != 0 {
		t.Errorf("expected empty log, got %d bytes", buf.Len())
	}
}

func TestNoHandlers(t *testing.T) {
	var buf bytes.Buffer

	BasicConfig(BasicConfigOpts{
		Level:  DEBUG,
		Writer: &buf,
	})

	GetLogger().RemoveHandlers()

	log := GetLogger("test")

	log.Info("this will never appear in the log")

	Shutdown()

	if buf.Len() != 0 {
		t.Errorf("expected empty log, got %d bytes", buf.Len())
	}
}

func TestMulti(t *testing.T) {
	var buf bytes.Buffer

	BasicConfig(BasicConfigOpts{
		Level:  DEBUG,
		Writer: &buf,
	})

	width := 100

	done := make(chan bool, width)

	for idx := 0; idx < width; idx++ {
		log := GetLogger(fmt.Sprintf("test%d", idx))

		go func(log *Logger) {
			for idx := 0; idx < width; idx++ {
				log.Info("test message %d", idx)
			}
			done <- true
		}(log)
	}

	for idx := 0; idx < width; idx++ {
		<-done
	}

	Shutdown()

	var foundLast int
	scanner := bufio.NewScanner(&buf)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "test message 99") {
			foundLast++
		}
	}

	if foundLast != width {
		t.Errorf("found %d last messages, expected %d", foundLast, width)
	}
}

func BenchmarkAllLogged(b *testing.B) {
	BasicConfig(BasicConfigOpts{
		Level:    WARNING, // thus all info-logs below will not be output
		FileName: "/dev/null",
	})

	log := GetLogger("test")

	//startTime := time.Now()

	for idx := 0; idx < b.N; idx++ {
		log.Info("test message %d", idx)
	}

	//duration := time.Now().Sub(startTime)
	Shutdown()

	//printPerf(b.N, duration)
}

func BenchmarkNoneLogged(b *testing.B) {
	BasicConfig(BasicConfigOpts{
		Level:    WARNING, // thus all info-logs below will not be output
		FileName: "/dev/null",
	})

	log := GetLogger("test")

	//startTime := time.Now()

	for idx := 0; idx < b.N; idx++ {
		log.Info("test message %d", idx)
	}

	Shutdown()
	//duration := time.Now().Sub(startTime)

	//printPerf(b.N, duration)
}

func BenchmarkMultiAllLogged(b *testing.B) {
	BasicConfig(BasicConfigOpts{
		Level:    DEBUG,
		FileName: "/dev/null",
	})

	startTime := time.Now()

	width := 100

	done := make(chan bool, width)

	for idx := 0; idx < width; idx++ {
		log := GetLogger(fmt.Sprintf("test%d", idx))

		go func(log *Logger) {
			for idx := 0; idx < b.N; idx++ {
				log.Info("test message %d", idx)
			}
			done <- true
		}(log)
	}

	for idx := 0; idx < width; idx++ {
		<-done
	}

	Shutdown()
	duration := time.Now().Sub(startTime)

	printPerf(width*b.N, duration)
}

func printPerf(n int, d time.Duration) {
	secs := d.Seconds()

	fmt.Fprintf(os.Stderr, "%d logs in %.3f ms -> %.0f logs/s  avg: %.3f µs/msg\n",
		n,
		secs*1e3,
		float64(n)/secs,
		secs*1e6/float64(n),
	)

}
