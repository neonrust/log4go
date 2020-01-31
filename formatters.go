package log4go

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tatsujin1/log4go/color"
)

// Formatter interface for formatters.
type Formatter interface {
	// Format formats a Record into a byte array
	Format(rec *Record) ([]byte, error)
}

// TemplateFormatter is formatting based on a string template.
type TemplateFormatter struct {
	formatString string
	formatTokens []interface{}

	levelColoring map[Level]string

	patternColoringPatterns []PatternColor
	patternColoring         map[string]string

	processMessage func(m, c string) string
}

// PatternColor pairs a color and a match pattern.
type PatternColor struct {
	color   string
	pattern *regexp.Regexp
}

func defaultProcessMessage(m, c string) string {
	return m
}

// NewTemplateFormatter returns a new TemplateFormatter.
func NewTemplateFormatter(format string) (*TemplateFormatter, error) {
	fmt := new(TemplateFormatter)
	fmt.formatString = format
	fmt.processMessage = defaultProcessMessage

	err := fmt.SetFormat(format)
	if err != nil {
		return nil, err
	}
	return fmt, nil
}

const (
	tfTime = iota
	tfTimeMilliseconds
	tfName
	tfBaseName
	tfLevel
	tfMessage

	tfFieldWidth      = 0x100 // width: 0 (auto) - 254
	tfFieldWidthMask  = 0xff00
	tfFieldWidthShift = 8

	tfAlignRight = 0x10000
	tfAlignLeft  = 0 // i.e. the default
)

// TODO: or string->func(Record) string
var tokenToValue = map[string]int{
	"time":     tfTime,
	"timems":   tfTimeMilliseconds,
	"name":     tfName,
	"basename": tfBaseName,
	"level":    tfLevel,
	"message":  tfMessage,
}

var templatePtn *regexp.Regexp
var templateSpecPtn *regexp.Regexp

var defaultLevelColoring map[Level]string
var defaultPatternColoringPatterns []PatternColor
var defaultPatternColoring map[string]string

func init() {
	defaultLevelColoring = map[Level]string{
		FATAL:   color.RedBg + color.Bold,
		ERROR:   color.Red,
		WARNING: color.Yellow,
		INFO:    color.Normal,
		DEBUG:   color.Faint,
	}

	defaultPatternColoringPatterns = []PatternColor{
		{"brackets", regexp.MustCompile(`([<>\]\(\)\{\}]|\[)`)}, // all kinds of brackets
		{"punct", regexp.MustCompile(`([-/\*\+\.,:])`)},
		{"quoted", regexp.MustCompile(`('[^']+'|"[^"]+")`)}, // quoted strings
	}
	defaultPatternColoring = map[string]string{
		"brackets": color.Purple,
		"punct":    color.Blue,
		"quoted":   color.Green,
	}
}

// EnableLevelColoring sets default coloring based on level, false to disable.
func (f *TemplateFormatter) EnableLevelColoring(enable bool) {
	if enable {
		f.levelColoring = defaultLevelColoring
	} else {
		f.levelColoring = nil
	}
}

// SetLevelColoring specifies how to color log lines based on level, nil to disable.
func (f *TemplateFormatter) SetLevelColoring(levelToColors map[Level]string) {
	f.levelColoring = levelToColors
}

// EnablePatternColoring sets default colors & patterns, false to disable.
func (f *TemplateFormatter) EnablePatternColoring(enable bool) {
	if enable {
		f.patternColoringPatterns = defaultPatternColoringPatterns
		f.patternColoring = defaultPatternColoring

		f.processMessage = makeProcessor(f.patternColoring, f.patternColoringPatterns)
	} else {
		f.patternColoringPatterns = nil
		f.patternColoring = nil
		f.processMessage = defaultProcessMessage
	}
}

// SetPatternColoring sets the color map and the patterns using them (any pattern matching '[' must be first).
func (f *TemplateFormatter) SetPatternColoring(colors map[string]string, patterns []PatternColor) {
	f.patternColoringPatterns = patterns
	f.patternColoring = colors
	f.processMessage = makeProcessor(f.patternColoring, f.patternColoringPatterns)
}

func makeProcessor(colors map[string]string, patterns []PatternColor) func(m, c string) string {
	return func(m string, baseColor string) string {
		repl := "$1" + baseColor
		for _, colPtn := range patterns {
			if color, exists := colors[colPtn.color]; exists {
				m = colPtn.pattern.ReplaceAllString(m, color+repl)
			}
		}
		return m
	}
}

// SetFormat setts the formatters template string format.
func (f *TemplateFormatter) SetFormat(template string) error {
	var err error
	if templatePtn == nil {
		templatePtn, err = regexp.Compile(`\{[^}]+\}`)
		if err != nil {
			return err
		}
	}
	if templateSpecPtn == nil {
		templateSpecPtn, err = regexp.Compile(`^\{([^}]+?)(([<>])(\d+))?\}$`) // e.g. "{name<20}" - left align, max width 20
		if err != nil {
			return err
		}
	}

	m := templatePtn.FindAllStringIndex(template, -1)
	if m == nil {
		return fmt.Errorf("invalid format template string: '%s'", template)
	}

	// compile the template into a token list
	tokens := []interface{}{}
	last := 0
	for _, tag := range m {
		start, end := tag[0], tag[1]
		if start > last {
			// part before the token
			tokens = append(tokens, template[last:start])
		}
		last = end

		item := template[start:end]

		spec := templateSpecPtn.FindStringSubmatch(item)
		token := spec[1]
		alignment := spec[3]
		width := spec[4]
		if len(alignment) > 0 && len(width) > 0 {
			w, _ := strconv.Atoi(width)
			if w > 0 {
				if w > 254 {
					w = 254
				}
				tokens = append(tokens, tfFieldWidth+(w-1)<<tfFieldWidthShift)
				if alignment == ">" {
					tokens = append(tokens, tfAlignRight)
				}
			}
		}

		value, ok := tokenToValue[token]
		if !ok {
			return fmt.Errorf("unknown format template token: '%s'", token)
		}

		tokens = append(tokens, value)
	}

	f.formatTokens = tokens

	return nil
}

// GetFormat returns the formatters template string.
func (f *TemplateFormatter) GetFormat() string {
	return f.formatString
}

const colorReset = "\x1b[0m"

// Format returns the record as a string.
func (f *TemplateFormatter) Format(r *Record) ([]byte, error) {
	parts := make([]string, 0, 10)

	alignFmt := ""
	width := 0

	colorSet := false
	var lineColor string
	if f.levelColoring != nil {
		var exists bool
		if lineColor, exists = f.levelColoring[r.Level]; exists {
			parts = append(parts, lineColor)
			colorSet = true
		} else {
			lineColor = "\x1b[0m"
		}
	}

	var processedMessage string

	for _, token := range f.formatTokens {
		switch token := token.(type) {
		case string:
			parts = append(parts, token)
		case int:
			s := ""
			switch {
			case token == tfTime:
				s = f.formatTime(r.Time, 1000)
			case token == tfTime:
				s = f.formatTime(r.Time)
			case token == tfName:
				if len(r.Name) == 0 {
					s = "root"
				} else {
					s = r.Name
				}
			case token == tfBaseName:
				if len(r.Name) == 0 {
					s = "root"
				} else {
					parts := strings.Split(r.Name, "/")
					s = parts[len(parts)-1]
				}
			case token == tfLevel:
				s = LevelName(r.Level)
			case token == tfMessage:
				if len(processedMessage) > 0 {
					s = processedMessage
				} else if len(r.Message) > 0 {
					processedMessage = f.processMessage(r.Message, lineColor)
					s = processedMessage
				}
			case token&tfFieldWidthMask > 0:
				width = ((token & tfFieldWidthMask) >> tfFieldWidthShift)
				if (token & tfAlignRight) > 0 {
					alignFmt = fmt.Sprintf("%%%ds", width)
				} else {
					alignFmt = fmt.Sprintf("%%-%ds", width)
				}
			}

			if len(s) > 0 {
				if len(alignFmt) > 0 {
					s = fmt.Sprintf(alignFmt, s)
					if len(s) > width {
						s = s[:width]
					}

					alignFmt = "" // field width used, reset it for next token
					width = 0
				}

				parts = append(parts, s)
			}
		}
	}

	if colorSet {
		parts = append(parts, colorReset)
	}

	return []byte(strings.Join(parts, "")), nil
}

func (f *TemplateFormatter) formatTime(t time.Time, resolution ...int) string {
	ts := fmt.Sprintf("%4d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())

	if len(resolution) == 1 && resolution[0] == 1000 {
		ts = fmt.Sprintf("%s.%03d", ts, int(t.Nanosecond()/1e6))
	}
	return ts
}
