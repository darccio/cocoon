package rt

import (
	"context"
	"crypto/rand"
	"io"
	"log/slog"
	"time"
)

// Host provides declared capabilities to one serialized guest instance.
// Imports must not reenter their guest or retain guest memory beyond a call.
type Host struct {
	module    Module
	Logger    *slog.Logger
	Random    io.Reader
	Now       func() time.Time
	lastPanic string
}

// NewHost selects standard-library defaults for logging, randomness, and time.
func NewHost(logger *slog.Logger) *Host {
	if logger == nil {
		logger = slog.Default()
	}
	return &Host{Logger: logger, Random: rand.Reader, Now: time.Now}
}

func shield() {
	if value := recover(); value != nil {
		panic(hostPanic{value: value})
	}
}

// Bind attaches memory after the translated module has been constructed.
func (h *Host) Bind(module Module) { h.module = module }

// Copy returns Go-owned bytes, bounded by current memory length.
func (h *Host) Copy(pointer, size uint32) ([]byte, error) {
	view, err := Range(h.module.Memory(), pointer, size)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), view...), nil
}

// Log copies UTF-8 bytes; level four also records the next guest fault's message.
func (h *Host) Log(level int32, pointer, size uint32) {
	defer shield()
	message, err := h.Copy(pointer, size)
	if err != nil {
		return
	}
	if level == 4 {
		h.lastPanic = string(message)
	}
	logLevel := slog.LevelError
	switch level {
	case 0, 1:
		logLevel = slog.LevelDebug
	case 2:
		logLevel = slog.LevelInfo
	case 3:
		logLevel = slog.LevelWarn
	}
	h.Logger.Log(context.Background(), logLevel, string(message), "source", "cocoon guest")
}

// RandomGet fills guest memory and returns zero on success or one on failure.
func (h *Host) RandomGet(pointer, size uint32) int32 {
	defer shield()
	view, err := Range(h.module.Memory(), pointer, size)
	if err != nil {
		return 1
	}
	if _, err := io.ReadFull(h.Random, view); err != nil {
		return 1
	}
	return 0
}

// ClockNanos returns signed Unix nanoseconds from the configured clock.
func (h *Host) ClockNanos() int64 {
	defer shield()
	return h.Now().UnixNano()
}

func (h *Host) takePanic() string {
	message := h.lastPanic
	h.lastPanic = ""
	return message
}
