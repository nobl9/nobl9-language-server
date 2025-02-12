package recovery

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/nobl9/nobl9-language-server/internal/messages"
)

func LogPanic(ctx context.Context, conn *jsonrpc2.Conn, recovered any) {
	if recovered == nil {
		return
	}
	slog.ErrorContext(ctx, "panic recovered", slog.Any("panic", recovered))
	notifyClient(ctx, conn, messages.ShowMessageMethod, messages.ShowMessageParams{
		Type: messages.MessageTypeError,
		Message: "Nobl9 LSP has encountered an internal error.\n" +
			"Please check the logs for more information and contact the developers.",
	})
	notifyClient(ctx, conn, messages.LogMessageMethod, messages.LogMessageParams{
		Type:    messages.MessageTypeError,
		Message: fmt.Sprint(recovered),
	})
}

func notifyClient(ctx context.Context, conn *jsonrpc2.Conn, method string, params any) {
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
