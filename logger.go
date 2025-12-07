package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

var _ slog.Handler = (*slogHandler)(nil)

type slogHandler struct {
	out   io.Writer
	level slog.Level
}

func newSlogHandler(out io.Writer, level slog.Level) *slogHandler {
	return &slogHandler{
		level: level,
		out:   out,
	}
}

func (h *slogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *slogHandler) Handle(ctx context.Context, record slog.Record) error {
	if connID, ok := ctx.Value(connIDKey).(string); ok {
		record.Add(slog.String(string(connIDKey), connID))
	}

	level := strings.ToUpper(record.Level.String())
	level += strings.Repeat(" ", 5-len(level))

	timestamp := record.Time.Format("[01-02|15:04:05.000]")

	fields := make([]string, 0, record.NumAttrs())

	record.Attrs(func(a slog.Attr) bool {
		fields = append(fields, fmt.Sprintf("%s=%v", a.Key, a.Value))
		return true
	})

	var buf bytes.Buffer

	_, _ = fmt.Fprintf(&buf, "%s %s %s %s\n", level, timestamp, record.Message, strings.Join(fields, " "))

	_, err := h.out.Write(buf.Bytes())
	return err
}

func (h *slogHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *slogHandler) WithGroup(_ string) slog.Handler {
	return h
}

type contextKey string

const connIDKey contextKey = "conn_id"

func setupLogger(levelStr string) {
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := newSlogHandler(os.Stdout, level)
	slog.SetDefault(slog.New(handler))
}
