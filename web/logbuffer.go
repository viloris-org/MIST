package web

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// LogEntry is a single log record sent to the UI.
type LogEntry struct {
	Time    string         `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

const maxLogEntries = 500

// LogBuffer is a thread-safe ring buffer of log entries.
type LogBuffer struct {
	mu      sync.Mutex
	entries []LogEntry
	pos     int
	filled  bool
	hub     *WSHub
}

// NewLogBuffer creates a new log ring buffer.
func NewLogBuffer(hub *WSHub) *LogBuffer {
	return &LogBuffer{
		entries: make([]LogEntry, maxLogEntries),
		hub:     hub,
	}
}

// append adds an entry to the buffer.
func (lb *LogBuffer) append(entry LogEntry) {
	lb.mu.Lock()
	lb.entries[lb.pos] = entry
	lb.pos++
	if lb.pos >= maxLogEntries {
		lb.pos = 0
		lb.filled = true
	}
	lb.mu.Unlock()

	if lb.hub != nil {
		lb.hub.BroadcastLog(entry)
	}
}

// Snapshot returns a copy of all entries in chronological order.
func (lb *LogBuffer) Snapshot() []LogEntry {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if !lb.filled {
		out := make([]LogEntry, lb.pos)
		copy(out, lb.entries[:lb.pos])
		return out
	}

	out := make([]LogEntry, maxLogEntries)
	copy(out, lb.entries[lb.pos:])
	copy(out[maxLogEntries-lb.pos:], lb.entries[:lb.pos])
	return out
}

// LogrusHook captures logrus entries into the buffer.
type LogrusHook struct {
	buffer  *LogBuffer
	levels  []logrus.Level
	formats map[logrus.Level]string
	stopped atomic.Bool
}

// NewLogrusHook creates a hook that feeds the log buffer.
func NewLogrusHook(buffer *LogBuffer) *LogrusHook {
	return &LogrusHook{
		buffer: buffer,
		levels: logrus.AllLevels,
		formats: map[logrus.Level]string{
			logrus.PanicLevel: "panic",
			logrus.FatalLevel: "fatal",
			logrus.ErrorLevel: "error",
			logrus.WarnLevel:  "warn",
			logrus.InfoLevel:  "info",
			logrus.DebugLevel: "debug",
			logrus.TraceLevel: "trace",
		},
	}
}

func (h *LogrusHook) Levels() []logrus.Level { return h.levels }

func (h *LogrusHook) Fire(entry *logrus.Entry) error {
	if h.stopped.Load() {
		return nil
	}
	level := h.formats[entry.Level]
	if level == "" {
		level = "unknown"
	}
	le := LogEntry{
		Time:    entry.Time.Format(time.RFC3339Nano),
		Level:   level,
		Message: entry.Message,
	}
	if len(entry.Data) > 0 {
		le.Fields = make(map[string]any, len(entry.Data))
		for k, v := range entry.Data {
			le.Fields[k] = v
		}
	}
	h.buffer.append(le)
	return nil
}
