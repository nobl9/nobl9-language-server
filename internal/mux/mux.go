package mux

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/nobl9/nobl9-language-server/internal/logging"
	"github.com/nobl9/nobl9-language-server/internal/recovery"
)

// rpcMethod is the name of a JSON RPC method.
// Example: "textDocument/didOpen".
type rpcMethod = string

// New construct a new [Mux] instance which will route JSON RPC messages to the
// designated [MethodHandler], routed via RPC method name.
// Example: New(map[rpcMethod]MethodHandler{"textDocument/didOpen":...})
func New(handlers map[rpcMethod]HandlerFunc) *Mux {
	return &Mux{handlers: handlers}
}

// Mux is multiplexer used to handle all incoming JSON RPC messages through [Mux.Handle].
// It delegates JSON RPC messages to a [MethodHandler] based on its method name.
type Mux struct {
	handlers map[rpcMethod]HandlerFunc
}

// HandlerFunc defines a JSON RPC method handler function shape.
// It returns an arbitrary result and an error.
// Result can be of any JSON-serializable type.
type HandlerFunc func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error)

// Handle is invoked for each incoming request.
// If the request is a call, it must return a value or an error for the reply.
// It is used in with [jsonrpc2.HandlerWithError] which in turn implements [jsonrpc2.Handler].
func (m *Mux) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
	ctx = logging.ContextAttr(ctx,
		slog.String("id", req.ID.String()),
		slog.String("method", req.Method))
	span, ctx := logging.StartSpan(ctx, "mux_handle")
	defer span.Finish()

	if logging.GetLogLevel() <= slog.LevelDebug {
		var params any
		if req.Params != nil {
			err := json.Unmarshal(*req.Params, &params)
			if err != nil {
				params = string(*req.Params)
			}
		}
		slog.DebugContext(ctx, "received request", slog.Any("params", params))
	}

	handle, ok := m.handlers[req.Method]
	if !ok {
		slog.ErrorContext(ctx, "method handler not found")
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeMethodNotFound,
			Message: fmt.Sprintf("method not supported: %s", req.Method)}
	}

	defer func() {
		recovery.LogPanic(ctx, conn, recover())
	}()
	result, err := handle(ctx, conn, req)
	if err != nil {
		slog.ErrorContext(ctx, "failed to handle method", slog.Any("error", err))
		return nil, err
	}
	slog.DebugContext(ctx, "served method", slog.Any("result", result))
	return result, nil
}
