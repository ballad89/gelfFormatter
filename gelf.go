package gelfFormatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Formatter struct {
	hostname string
	Facility string
}

type Message struct {
	Version  string                 `json:"version"`
	Host     string                 `json:"host"`
	Short    string                 `json:"short_message"`
	Full     string                 `json:"full_message,omitempty"`
	TimeUnix float64                `json:"timestamp"`
	Level    int32                  `json:"level,omitempty"`
	Facility string                 `json:"_facility,omitempty"`
	Extra    map[string]interface{} `json:"-"`
	RawExtra json.RawMessage        `json:"-"`
}

// Syslog severity levels
const (
	LOG_EMERG   = int32(0)
	LOG_ALERT   = int32(1)
	LOG_CRIT    = int32(2)
	LOG_ERR     = int32(3)
	LOG_WARNING = int32(4)
	LOG_NOTICE  = int32(5)
	LOG_INFO    = int32(6)
	LOG_DEBUG   = int32(7)
)

func getLevel(l log.Level) int32 {

	var level int32
	switch l {
	case log.PanicLevel:
		level = LOG_EMERG
	case log.FatalLevel:
		level = LOG_CRIT
	case log.ErrorLevel:
		level = LOG_ERR
	case log.WarnLevel:
		level = LOG_WARNING
	case log.InfoLevel:
		level = LOG_INFO
	case log.DebugLevel:
		level = LOG_DEBUG
	}

	return level

}

func NewGelfFormatter(facility string) (*Formatter, error) {
	var err error
	w := Formatter{}

	if w.hostname, err = os.Hostname(); err != nil {
		return nil, err
	}

	if facility != "" {
		w.Facility = facility
	} else {
		w.Facility = path.Base(os.Args[0])
	}

	return &w, nil
}

func (f *Formatter) Format(entry *log.Entry) ([]byte, error) {

	// remove trailing and leading whitespace
	msg := strings.TrimSpace(entry.Message)

	// If there are newlines in the message, use the first line
	// for the short message and set the full message to the
	// original input.  If the input has no newlines, stick the
	// whole thing in Short.
	short := msg
	full := ""

	if i := strings.IndexRune(msg, '\n'); i > 0 {
		short = msg[:i]
		full = msg
	}

	level := getLevel(entry.Level)

	m := Message{
		Version:  "1.1",
		Host:     f.hostname,
		Short:    short,
		Full:     full,
		TimeUnix: float64(entry.Time.Unix()),
		Level:    level,
		Facility: f.Facility,
		Extra:    map[string]interface{}{},
	}

	if level < LOG_WARNING {
		file, line := getCallerIgnoringLogMulti(1)

		entry.Data["file"] = file
		entry.Data["line"] = line
	}

	entry.Data["severity"] = entry.Level.String()

	additionalFields := formatAdditionalfields(entry.Data)

	m.Extra = additionalFields

	mBuf := newBuffer()
	defer bufPool.Put(mBuf)
	if err := m.MarshalJSONBuf(mBuf); err != nil {
		return nil, err
	}
	return mBuf.Bytes(), nil
}

// 1k bytes buffer by default
var bufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 1024))
	},
}

func newBuffer() *bytes.Buffer {
	b := bufPool.Get().(*bytes.Buffer)
	if b != nil {
		b.Reset()
		return b
	}
	return bytes.NewBuffer(nil)
}

func (m *Message) MarshalJSONBuf(buf *bytes.Buffer) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	// write up until the final }
	if _, err = buf.Write(b[:len(b)-1]); err != nil {
		return err
	}
	if len(m.Extra) > 0 {
		eb, err := json.Marshal(m.Extra)
		if err != nil {
			return err
		}
		// merge serialized message + serialized extra map
		if err = buf.WriteByte(','); err != nil {
			return err
		}
		// write serialized extra bytes, without enclosing quotes
		if _, err = buf.Write(eb[1 : len(eb)-1]); err != nil {
			return err
		}
	}

	if len(m.RawExtra) > 0 {
		if err := buf.WriteByte(','); err != nil {
			return err
		}

		// write serialized extra bytes, without enclosing quotes
		if _, err = buf.Write(m.RawExtra[1 : len(m.RawExtra)-1]); err != nil {
			return err
		}
	}

	// write final closing quotes
	_, err = buf.WriteString("}\n")

	return err
}

func formatAdditionalfields(f log.Fields) log.Fields {
	n := make(map[string]interface{})

	for k, v := range f {

		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/sirupsen/logrus/issues/137
			n[fmt.Sprintf("_%s", k)] = v.Error()
		default:
			n[fmt.Sprintf("_%s", k)] = v
		}
	}

	return n
}

func getCallerIgnoringLogMulti(callDepth int) (string, int) {
	// the +1 is to ignore this (getCallerIgnoringLogMulti) frame
	return getCaller(callDepth+1, "/pkg/log/log.go", "/pkg/io/multi.go", "/sirupsen/logrus/")
}

// getCaller returns the filename and the line info of a function
// further down in the call stack.  Passing 0 in as callDepth would
// return info on the function calling getCallerIgnoringLog, 1 the
// parent function, and so on.  Any suffixes passed to getCaller are
// path fragments like "/pkg/log/log.go", and functions in the call
// stack from that file are ignored.
func getCaller(callDepth int, suffixesToIgnore ...string) (file string, line int) {
	// bump by 1 to ignore the getCaller (this) stackframe
	callDepth++
outer:
	for {
		var ok bool
		_, file, line, ok = runtime.Caller(callDepth)
		if !ok {
			file = "???"
			line = 0
			break
		}

		for _, s := range suffixesToIgnore {
			if strings.Contains(file, s) {
				callDepth++
				continue outer
			}
		}
		break
	}
	return
}
