package logger

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type LogEntry struct {
	Time    time.Time
	Level   slog.Level
	Message string
	Attrs   map[string]interface{}
}

// MemoryHandler is a custom slog.Handler that stores the last N logs in memory
// for the TUI to read, while optionally falling back to another handler.
type MemoryHandler struct {
	opts     slog.HandlerOptions
	fallback slog.Handler
	
	mu       sync.Mutex
	logs     []LogEntry
	capacity int
}

func NewMemoryHandler(capacity int, fallback slog.Handler, opts slog.HandlerOptions) *MemoryHandler {
	return &MemoryHandler{
		opts:     opts,
		fallback: fallback,
		logs:     make([]LogEntry, 0, capacity),
		capacity: capacity,
	}
}

func (m *MemoryHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= m.opts.Level.Level()
}

func (m *MemoryHandler) Handle(ctx context.Context, r slog.Record) error {
	entry := LogEntry{
		Time:    r.Time,
		Level:   r.Level,
		Message: r.Message,
		Attrs:   make(map[string]interface{}),
	}
	
	r.Attrs(func(a slog.Attr) bool {
		entry.Attrs[a.Key] = a.Value.Any()
		return true
	})

	m.mu.Lock()
	if len(m.logs) >= m.capacity {
		// Shift array to make room (simple approach for small capacities)
		m.logs = append(m.logs[1:], entry)
	} else {
		m.logs = append(m.logs, entry)
	}
	m.mu.Unlock()

	if m.fallback != nil {
		return m.fallback.Handle(ctx, r)
	}
	return nil
}

func (m *MemoryHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return m
}

func (m *MemoryHandler) WithGroup(name string) slog.Handler {
	return m
}

// GetLogs returns a copy of the recent logs
func (m *MemoryHandler) GetLogs() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]LogEntry, len(m.logs))
	copy(res, m.logs)
	return res
}

// FormatLogs is a utility to format logs as strings for the TUI
func (m *MemoryHandler) FormatLogs() []string {
	logs := m.GetLogs()
	var formatted []string
	for _, l := range logs {
		var attrStr string
		if len(l.Attrs) > 0 {
			var buf bytes.Buffer
			for k, v := range l.Attrs {
				buf.WriteString(fmt.Sprintf("%s=%v ", k, v))
			}
			attrStr = " " + buf.String()
		}
		formatted = append(formatted, fmt.Sprintf("[%s] %s%s", l.Level.String(), l.Message, attrStr))
	}
	return formatted
}
