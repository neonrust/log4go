package logging

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Formatter interface for formatters.
type Formatter interface {
	Format(rec *Record) (string, error)
}

// TemplateFormatter is formatting based on a string template.
type TemplateFormatter struct {
	formatString string
	formatTokens []interface{}
}

// NewTemplateFormatter returns a new TemplateFormatter.
func NewTemplateFormatter(format string) (Formatter, error) {
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
	"name":    tfName,
	"level":   tfLevel,
	"message": tfMessage,
}

var templatePtn *regexp.Regexp
var templateSpecPtn *regexp.Regexp

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

// Format returns the record as a string.
func (f *TemplateFormatter) Format(r *Record) (string, error) {
	parts := make([]string, 0, 100)

	alignFmt := ""
	width := 0

	for _, token := range f.formatTokens {
		switch token := token.(type) {
		case string:
			parts = append(parts, token)
		case int:
			s := ""
			switch {
			case token == tfTime:
				s = f.formatTime(r.Time)
			case token == tfName:
				if len(r.Name) == 0 {
					s = "root"
				} else {
					s = r.Name
				}
			case token == tfLevel:
				s = f.formatLevel(r.Level)
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

	return strings.Join(parts, ""), nil
}

func (f *TemplateFormatter) formatTime(t time.Time) string {
	return fmt.Sprintf("%4d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func (f *TemplateFormatter) formatLevel(level int) string {
	name, ok := levelToName[level]
	if !ok {
		name = "UNSET"
	}
	return name
}
