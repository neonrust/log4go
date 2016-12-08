package log4go_test

import (
	"fmt"
	"os"
	"time"
	"bytes"
	"bufio"
	"strings"
	"testing"

	"../../log4go"
)


func TestOne(t *testing.T) {
	var buf bytes.Buffer

	log4go.BasicConfig(log4go.BasicConfigOpts{
		Level:  log4go.DEBUG,
		Writer: &buf,
	})

	log := log4go.GetLogger("test")

	for idx := 0; idx < 100; idx ++ {
		log.Info("test message %d", idx)
	}

	log4go.Shutdown()

	foundLast := false
	scanner := bufio.NewScanner(&buf)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "test message 99") {
			foundLast = true
		}
	}

	if ! foundLast {
		t.Error("last message not found")
	}
}

func TestLevelFilter(t *testing.T) {
	var buf bytes.Buffer

	log4go.BasicConfig(log4go.BasicConfigOpts{
		Level:  log4go.WARNING,
		Writer: &buf,
	})

	log := log4go.GetLogger("test")

	log.Info("this will never appear in the log")

	log4go.Shutdown()

	if buf.Len() != 0 {
		t.Errorf("expected empty log, got %d bytes", buf.Len())
	}
}

func TestNoHandlers(t *testing.T) {
	var buf bytes.Buffer

	log4go.BasicConfig(log4go.BasicConfigOpts{
		Level:  log4go.DEBUG,
		Writer: &buf,
	})

	log4go.GetLogger().RemoveHandlers()

	log := log4go.GetLogger("test")

	log.Info("this will never appear in the log")

	log4go.Shutdown()

	if buf.Len() != 0 {
		t.Errorf("expected empty log, got %d bytes", buf.Len())
	}
}


func TestMulti(t *testing.T) {
	var buf bytes.Buffer

	log4go.BasicConfig(log4go.BasicConfigOpts{
		Level:  log4go.DEBUG,
		Writer: &buf,
	})

	width := 100

	done := make(chan bool, width)

	for idx := 0; idx < width; idx++ {
		log := log4go.GetLogger(fmt.Sprintf("test%d", idx))

		go func(log *log4go.Logger) {
			for idx := 0; idx < width; idx++ {
				log.Info("test message %d", idx)
			}
			done <- true
		}(log)
	}

	for idx := 0; idx < width; idx++ {
		<-done
	}

	log4go.Shutdown()

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
	log4go.BasicConfig(log4go.BasicConfigOpts{
		Level:    log4go.WARNING,  // thus all info-logs below will not be output
		FileName: "/dev/null",
	})

	log := log4go.GetLogger("test")

	//startTime := time.Now()

	for idx := 0; idx < b.N; idx++ {
		log.Info("test message %d", idx)
	}

	//duration := time.Now().Sub(startTime)
	log4go.Shutdown()

	//printPerf(b.N, duration)
}

func BenchmarkNoneLogged(b *testing.B) {
	log4go.BasicConfig(log4go.BasicConfigOpts{
		Level:    log4go.WARNING,  // thus all info-logs below will not be output
		FileName: "/dev/null",
	})

	log := log4go.GetLogger("test")

	//startTime := time.Now()

	for idx := 0; idx < b.N; idx++ {
		log.Info("test message %d", idx)
	}

	log4go.Shutdown()
	//duration := time.Now().Sub(startTime)

	//printPerf(b.N, duration)
}

func BenchmarkMultiAllLogged(b *testing.B) {
	log4go.BasicConfig(log4go.BasicConfigOpts{
		Level:    log4go.DEBUG,
		FileName: "/dev/null",
	})

	startTime := time.Now()

	width := 100

	done := make(chan bool, width)

	for idx := 0; idx < width; idx++ {
		log := log4go.GetLogger(fmt.Sprintf("test%d", idx))

		go func(log *log4go.Logger) {
			for idx := 0; idx < b.N; idx++ {
				log.Info("test message %d", idx)
			}
			done <- true
		}(log)
	}

	for idx := 0; idx < width; idx++ {
		<-done
	}

	log4go.Shutdown()
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