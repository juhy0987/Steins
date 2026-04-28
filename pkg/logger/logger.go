// Package logger provides a thin zerolog wrapper used across Steins.
//
// logger 패키지는 Steins 전반에서 사용하는 zerolog 기반 wrapper를 제공합니다.
package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type ctxKey struct{}

// Logger wraps a zerolog.Logger with the field-builder API used in the rules.
//
// Logger는 룰셋에서 사용하는 field-builder API를 가진 zerolog.Logger wrapper입니다.
type Logger struct {
	zl zerolog.Logger
}

// New returns a Logger configured for the given environment.
// `pretty` enables a human-readable console writer (development).
func New(pretty bool) *Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	var w io.Writer = os.Stdout
	if pretty {
		w = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	zl := zerolog.New(w).With().Timestamp().Logger()
	return &Logger{zl: zl}
}

// WithFields attaches structured fields to a derived logger.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	if len(fields) == 0 {
		return l
	}
	return &Logger{zl: l.zl.With().Fields(fields).Logger()}
}

// WithField attaches a single field.
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{zl: l.zl.With().Interface(key, value).Logger()}
}

// WithError attaches an error field.
func (l *Logger) WithError(err error) *Entry {
	return &Entry{zl: l.zl, err: err}
}

// Debug emits a DEBUG-level log entry.
func (l *Logger) Debug() *Event { return &Event{ev: l.zl.Debug()} }

// Info emits an INFO-level log entry.
func (l *Logger) Info() *Event { return &Event{ev: l.zl.Info()} }

// Warn emits a WARN-level log entry.
func (l *Logger) Warn() *Event { return &Event{ev: l.zl.Warn()} }

// Error emits an ERROR-level log entry.
func (l *Logger) Error() *Event { return &Event{ev: l.zl.Error()} }

// Fatal emits a FATAL-level log entry and exits with code 1.
func (l *Logger) Fatal() *Event { return &Event{ev: l.zl.Fatal()} }

// ToContext attaches the logger to a context.
func (l *Logger) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext retrieves a logger from context, falling back to a default.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		return l
	}
	return &Logger{zl: zerolog.Nop()}
}

// Entry carries an attached error so the caller can chain a level + message.
type Entry struct {
	zl  zerolog.Logger
	err error
}

func (e *Entry) WithField(key string, value interface{}) *Entry {
	return &Entry{zl: e.zl.With().Interface(key, value).Logger(), err: e.err}
}

func (e *Entry) Debug() *Event { return &Event{ev: e.zl.Debug().Err(e.err)} }
func (e *Entry) Info() *Event  { return &Event{ev: e.zl.Info().Err(e.err)} }
func (e *Entry) Warn() *Event  { return &Event{ev: e.zl.Warn().Err(e.err)} }
func (e *Entry) Error() *Event { return &Event{ev: e.zl.Error().Err(e.err)} }
func (e *Entry) Fatal() *Event { return &Event{ev: e.zl.Fatal().Err(e.err)} }

// Event wraps a zerolog Event so callers terminate with Msg(...).
type Event struct {
	ev *zerolog.Event
}

func (e *Event) Str(key, value string) *Event {
	e.ev.Str(key, value)
	return e
}

func (e *Event) Int(key string, value int) *Event {
	e.ev.Int(key, value)
	return e
}

func (e *Event) Int64(key string, value int64) *Event {
	e.ev.Int64(key, value)
	return e
}

func (e *Event) Dur(key string, value time.Duration) *Event {
	e.ev.Dur(key, value)
	return e
}

func (e *Event) Err(err error) *Event {
	e.ev.Err(err)
	return e
}

func (e *Event) Msg(msg string) {
	e.ev.Msg(msg)
}
