package recovery

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/nobl9/nobl9-language-server/internal/messages"
)

var conn *jsonrpc2.Conn

// Setup sets the RPC connection which will be used to notify client about panics.
func Setup(c *jsonrpc2.Conn) { conn = c }

// SafeGo runs the given function in a goroutine and recovers from any panic
// handling it with [LogPanic].
func SafeGo(fn func()) {
	go func() {
		defer func() { LogPanic(context.Background(), recover()) }()
		fn()
	}()
}

// LogPanic logs the panic and notifies the client both by sending
// window/showMessage and window/logMessage request.
func LogPanic(ctx context.Context, recovered any) {
	if recovered == nil {
		return
	}
	stackTrace := string(debug.Stack())
	slog.ErrorContext(ctx, "panic recovered",
		slog.Any("panic", recovered),
		slog.String("stack", stackTrace))

	notifyClient(ctx, conn, messages.ShowMessageMethod, messages.ShowMessageParams{
		Type: messages.MessageTypeError,
		Message: "Nobl9 LSP has encountered an internal error.\n" +
			"Please check the logs for more information and contact the developers.",
	})
	notifyClient(ctx, conn, messages.LogMessageMethod, messages.LogMessageParams{
		Type:    messages.MessageTypeError,
		Message: fmt.Sprintf("%v\nStack trace:\n%s", recovered, stackTrace),
	})
}

func notifyClient(ctx context.Context, conn *jsonrpc2.Conn, method string, params any) {
	if conn == nil {
		return
	}
	err := conn.Notify(ctx, method, params)
	if err == nil {
		return
	}
	slog.ErrorContext(
		ctx,
		"failed to notify client",
		slog.Any("error", err),
		slog.Any("method", method),
		slog.Any("params", params),
	)
}
