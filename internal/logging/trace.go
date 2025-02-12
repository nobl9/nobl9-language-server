package logging

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

// LevelTrace is a custom [slog.Level] used to distinguish trace logging.
const LevelTrace = slog.Level(-8)

// Span is an interface for single trace spans.
type Span interface {
	Finish()
}

type spanHandler struct {
	ctx   context.Context
	name  string
	pc    uintptr
	start time.Time
	end   time.Time
}

type tracerContextKey struct{}

// StartSpan begins a new [Span] with the given name.
// If an existing [Span] is present in the [context.Context],
// the new [Span] name will be joined with the parent [Span] name.
func StartSpan(ctx context.Context, name string) (Span, context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	if parent, _ := ctx.Value(tracerContextKey{}).(*spanHandler); parent != nil && parent.name != "" {
		name = parent.name + "." + name
	}
	var pcs [1]uintptr
	// skip [runtime.Callers, this function]
	runtime.Callers(2, pcs[:])
	span := &spanHandler{
		name:  name,
		start: time.Now(),
		pc:    pcs[0],
	}
	ctx = context.WithValue(ctx, tracerContextKey{}, span)
	return span, ctx
}

const spanFinishedMsg = "SPAN FINISHED"

// Finish ends the [Span] and logs it through [slog.Logger].
func (s *spanHandler) Finish() {
	s.end = time.Now()
	if s.ctx == nil {
		s.ctx = context.Background()
	}
	record := slog.NewRecord(s.end, LevelTrace, spanFinishedMsg, s.pc)
	record.Add(
		slog.Group("span",
			slog.String("name", s.name),
			slog.String("duration", s.end.Sub(s.start).String()),
			slog.Time("start_time", s.start),
			slog.Time("end_time", s.end),
		),
	)
	_ = slog.Default().Handler().Handle(s.ctx, record)
}
