package log4go

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
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

	for idx := 1; idx <= 100; idx++ {
		log.Info("test message %d", idx)
	}

	Shutdown()

	scanner := bufio.NewScanner(&buf)
	foundLast := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "test message 100") {
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

	text := "test message 99"
	log.Info(text)

	Shutdown()

	scanner := bufio.NewScanner(&buf)
	foundLast := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, text) {
			foundLast = true
		}
	}

	if !foundLast {
		t.Errorf("last message not found (output len: %d)", buf.Len())
	}
}

func TestTimeFormatS(t *testing.T) {
	var buf bytes.Buffer

	BasicConfig(BasicConfigOpts{
		Level:  INFO,
		Writer: &buf,
		Format: "{time} {message}",
	})

	log := GetLogger("test")

	log.Info("test message")

	Shutdown()

	scanner := bufio.NewScanner(&buf)
	found := false

	ptn := regexp.MustCompile(`^\d{4}(-\d\d){2} \d\d(:\d\d){2}$`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, " test message") {
			found = true
			line = line[:len(line)-13] // cut off the message text
			if len(ptn.FindString(line)) == 0 {
				t.Errorf("Time format (%s) was not as expected (%s)", line, ptn.String())
			}
		}
	}

	if !found {
		t.Errorf("last message not found (output len: %d)", buf.Len())
		print_last_lines(t, buf, 10)
	}
}

func TestTimeFormatMS(t *testing.T) {
	var buf bytes.Buffer

	BasicConfig(BasicConfigOpts{
		Level:  INFO,
		Writer: &buf,
		Format: "{timems} {message}",
	})

	log := GetLogger("test")

	log.Info("test message")

	Shutdown()

	scanner := bufio.NewScanner(&buf)
	found := false

	ptn := regexp.MustCompile(`^\d{4}(-\d\d){2} \d\d(:\d\d){2}.\d{3}$`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, " test message") {
			found = true
			line = line[:len(line)-13] // cut off the message text
			if len(ptn.FindString(line)) == 0 {
				t.Errorf("Time format (%s) was not as expected (%s)", line, ptn.String())
			}
		}
	}

	if !found {
		t.Errorf("last message not found (output len: %d)", buf.Len())
		print_last_lines(t, buf, 10)
	}
}

func TestTimeFormatUS(t *testing.T) {
	var buf bytes.Buffer

	BasicConfig(BasicConfigOpts{
		Level:  INFO,
		Writer: &buf,
		Format: "{timeus} {message}",
	})

	log := GetLogger("test")

	log.Info("test message")

	Shutdown()

	scanner := bufio.NewScanner(&buf)
	found := false

	ptn := regexp.MustCompile(`^\d{4}(-\d\d){2} \d\d(:\d\d){2}.\d{6}$`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, " test message") {
			found = true
			line = line[:len(line)-13] // cut off the message text
			if len(ptn.FindString(line)) == 0 {
				t.Errorf("Time format (%s) was not as expected (%s)", line, ptn.String())
			}
		}
	}

	if !found {
		t.Errorf("last message not found (output len: %d)", buf.Len())
		print_last_lines(t, buf, 10)
	}
}

func TestStaged(t *testing.T) {
	var buf bytes.Buffer

	BasicConfig(BasicConfigOpts{
		Level:  DEBUG,
		Writer: &buf,
		Format: "{timeus} {message}",
	})

	log := GetLogger("test")

	log.StageDebug("test message debug")
	log.StageInfo("test message info")
	log.Error("test message error")

	Shutdown()

	scanner := bufio.NewScanner(&buf)
	var found uint8

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "test message debug") {
			found |= 0x001
		}
		if strings.HasSuffix(line, "test message info") {
			found |= 0b010
		}
		if strings.HasSuffix(line, "test message error") {
			found |= 0b100
		}
	}

	if found != 0b111 {
		t.Errorf("not all messages were found: 0b%03b, expected 0b111 (output len: %d)", found, buf.Len())
		print_last_lines(t, buf, 10)
	}
}

func TestStagedUnflushed(t *testing.T) {
	var buf bytes.Buffer

	BasicConfig(BasicConfigOpts{
		Level:  DEBUG,
		Writer: &buf,
		Format: "{timeus} {message}",
	})

	log := GetLogger("test")

	log.StageDebug("test message debug")
	log.StageInfo("test message info")

	Shutdown()

	scanner := bufio.NewScanner(&buf)
	var found uint8

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "test message debug") {
			found |= 0x001
		}
		if strings.HasSuffix(line, "test message info") {
			found |= 0b010
		}
	}

	if found != 0 {
		t.Errorf("messages were found: 0b%03b, expected 0 (output len: %d)", found, buf.Len())
		print_last_lines(t, buf, 10)
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
		print_last_lines(t, buf, 10)
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
		print_last_lines(t, buf, 10)
	}
}

func TestMulti(t *testing.T) {
	var buf bytes.Buffer

	BasicConfig(BasicConfigOpts{
		Level:  DEBUG,
		Writer: &buf,
	})

	width := 100

	wg := &sync.WaitGroup{}
	wg.Add(width)

	for idx := 0; idx < width; idx++ {
		log := GetLogger(fmt.Sprintf("test%d", idx))

		go func(log *Logger) {
			for idx := 0; idx < width; idx++ {
				log.Info("test message %d", idx)
			}
			wg.Done()
		}(log)
	}

	wg.Wait()

	Shutdown()

	scanner := bufio.NewScanner(&buf)
	var foundLast int

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

func print_last_lines(t *testing.T, buf bytes.Buffer, count int) {
	contents := buf.String()
	lines := strings.Split(contents, "\n")

	lines = lines[:len(lines)-1] // last item is an empty string

	var last []string
	if len(lines) > count {
		last = lines[len(lines)-count:]
	} else {
		last = lines
	}
	t.Errorf("Last %d lines of content:\n", count)
	for _, line := range last {
		t.Error(line)
	}
	t.Error("END content")
}
