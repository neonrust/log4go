package log4go

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
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

	levelColoring map[int]string
}

// NewTemplateFormatter returns a new TemplateFormatter.
func NewTemplateFormatter(format string) (*TemplateFormatter, error) {
	fmt := new(TemplateFormatter)
	fmt.formatString = format

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
	tfLevel
	tfMessage

	tfFieldWidth      = 0x100 // width: 0 (auto) - 254
	tfFieldWidthMask  = 0xff00
	tfFieldWidthShift = 8

	tfAlignRight = 0x10000
	tfAlignLeft  = 0 // i.e. the default
)

var tokenToValue = map[string]int{
	"time":    tfTime,
	"timems":  tfTimeMilliseconds,
	"name":    tfName,
	"level":   tfLevel,
	"message": tfMessage,
}

var templatePtn *regexp.Regexp
var templateSpecPtn *regexp.Regexp

var defaultlevelColoring map[int]string

var color_bold string
var color_normal string
var color_faint string
var color_red string
var color_fail string
var color_green string
var color_yellow string
var color_blue string
var color_purple string
var color_red_bg string

func init() {
	_esc := func(codes... string) string {
		return strings.Join([]string{
			"\x1b",
			"[",
			strings.Join(codes, ";"),
			"m",
		}, "")
	}
	color_bold = _esc("1")
	color_normal = _esc("0")
	color_faint = _esc("38", "5", "240")
	color_red = _esc("31", "1")
	color_fail = _esc("41", "37", "1")
	color_green = _esc("38", "5", "66")
	color_yellow = _esc("38", "5", "220")
	color_blue = _esc("38", "5", "39")
	color_purple = _esc("38", "5", "96")
	color_red_bg = _esc("41", "1")

	defaultlevelColoring = map[int]string {
		FATAL: color_red_bg + color_bold,
		ERROR: color_red,
		WARNING: color_yellow,
		INFO: color_normal,
		DEBUG: color_faint,
	}
}


// EnableLevelColoring sets default coloring based on level, false to disable.
func (f *TemplateFormatter) EnableLevelColoring(enable bool) {
	if enable {
		f.levelColoring = defaultlevelColoring
	} else {
		f.levelColoring = nil
	}
}

// SetLevelColoring specifies how to color log lines based on level, nil to disable.
func (f *TemplateFormatter) SetLevelColoring(levelToColors map[int]string) {
	f.levelColoring = levelToColors
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
	if f.levelColoring != nil {
		if color, ok := f.levelColoring[r.Level]; ok {
			parts = append(parts, color)
			colorSet = true
		}
	}

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
			case token == tfLevel:
				s = LevelName(r.Level)
			case token == tfMessage:
				s = r.Message
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

func (f *TemplateFormatter) formatTime(t time.Time, resolution... int) string {
	ts := fmt.Sprintf("%4d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())

	if len(resolution) == 1 && resolution[0] == 1000 {
		ts = fmt.Sprintf("%s.%03d", ts, int(t.Nanosecond()/1e6))
	}
	return ts
}
