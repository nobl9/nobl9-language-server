package logging

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
)

var logLevel slog.Level

// levelNames maps custom [slog.Leveler] to their string representation.
var levelNames = map[slog.Leveler]string{
	LevelTrace: "TRACE",
}

func Setup(flagConf Config) io.Closer {
	conf := flagConf
	var writer io.WriteCloser
	if conf.LogFile != "" {
		lf, err := os.OpenFile(conf.LogFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o600)
		if err != nil {
			slog.Error("failed to open log output file", slog.Any("error", err))
			os.Exit(1)
		}
		writer = lf
	} else {
		writer = os.Stderr
	}
	logLevel = conf.LogLevel.Level
	jsonHandler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		// We're using our own source handler.
		AddSource: false,
		Level:     conf.LogLevel.Level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				levelLabel, exists := levelNames[level]
				if !exists {
					levelLabel = level.String()
				}
				a.Value = slog.StringValue(levelLabel)
			}
			return a
		},
	})
	// Source handler should always be the first in the chain
	// in order to keep the number of frames it has to skip consistent.
	handler := sourceHandler{Handler: jsonHandler}
	defaultLogger := slog.New(contextHandler{Handler: handler})
	slog.SetDefault(defaultLogger)
	return writer
}

func GetLogLevel() slog.Level {
	return logLevel
}

// Level is a custom [slog.Leveler] that adds custom levels support extending [slog.Level].
type Level struct {
	slog.Level
}

// UnmarshalText implements [encoding.TextUnmarshaler].
// It's case-insensitive.
func (l *Level) UnmarshalText(data []byte) error {
	data = bytes.ToUpper(data)
	if string(data) == "TRACE" {
		l.Level = LevelTrace
		return nil
	}
	return l.Level.UnmarshalText(data)
}

func DefaultLevel() Level {
	return Level{Level: slog.LevelInfo}
}

type Config struct {
	LogFile  string `json:"logFile"`
	LogLevel Level  `json:"logLevel"`
}

type logContextAttrKey struct{}

// contextHandler is a [slog.Handler] that adds contextual attributes
// to the [slog.Record] before calling the underlying handler.
type contextHandler struct{ slog.Handler }

// Handle adds contextual attributes to the Record before calling the underlying handler.
func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(logContextAttrKey{}).([]slog.Attr); ok {
		for _, v := range attrs {
			r.AddAttrs(v)
		}
	}
	return h.Handler.Handle(ctx, r)
}

// ContextAttr appends a [slog.Attr] to the provided [context.Context] so that it will be
// included in any [slog.Record] created with such context.
func ContextAttr(ctx context.Context, attr ...slog.Attr) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if v, ok := ctx.Value(logContextAttrKey{}).([]slog.Attr); ok {
		return context.WithValue(ctx, logContextAttrKey{}, append(v, attr...))
	}
	return context.WithValue(ctx, logContextAttrKey{}, attr)
}

// sourceHandler is a [slog.Handler] that adds [slog.Source] information to the [slog.Record].
type sourceHandler struct{ slog.Handler }

// Handle adds [slog.Source] information to the [slog.Record]
// before calling the underlying handler.
func (h sourceHandler) Handle(ctx context.Context, r slog.Record) error {
	f, ok := runtime.CallersFrames([]uintptr{r.PC}).Next()
	if !ok {
		r.AddAttrs(slog.Attr{
			Key: slog.SourceKey,
			Value: slog.AnyValue(&slog.Source{
				Function: f.Function,
				File:     f.File,
				Line:     f.Line,
			}),
		})
	}
	return h.Handler.Handle(ctx, r)
}
