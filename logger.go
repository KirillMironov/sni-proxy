package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

var _ slog.Handler = (*handler)(nil)

type handler struct {
	out   io.Writer
	level slog.Level
}

func newHandler(out io.Writer, level slog.Level) *handler {
	return &handler{
		level: level,
		out:   out,
	}
}

func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *handler) Handle(_ context.Context, record slog.Record) error {
	level := strings.ToUpper(record.Level.String())
	level += strings.Repeat(" ", 5-len(level))

	timestamp := record.Time.Format("[01-02|15:04:05.000]")

	fields := make([]string, 0, record.NumAttrs())

	record.Attrs(func(a slog.Attr) bool {
		fields = append(fields, fmt.Sprintf("%s=%v", a.Key, a.Value))
		return true
	})

	line := fmt.Sprintf("%s %s %s", level, timestamp, record.Message)
	if len(fields) > 0 {
		line += "\t" + strings.Join(fields, " ")
	}
	line += "\n"

	_, err := h.out.Write([]byte(line))
	return err
}

func (h *handler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *handler) WithGroup(_ string) slog.Handler {
	return h
}
