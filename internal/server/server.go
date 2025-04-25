package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	v1alphaParser "github.com/nobl9/nobl9-go/manifest/v1alpha/parser"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/nobl9/nobl9-language-server/internal/codeactions"
	"github.com/nobl9/nobl9-language-server/internal/config"
	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/logging"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/mux"
	"github.com/nobl9/nobl9-language-server/internal/recovery"
)

const languageID = "yaml"

func New(ctx context.Context, lspVersion string, filePatterns []string) (*Server, error) {
	span, _ := logging.StartSpan(ctx, "server_start")
	defer span.Finish()

	var conn *jsonrpc2.Conn
	filesystem := files.NewFS(filePatterns)
	notifier := &rpcConnectionNotifier{conn: conn}
	registry, err := newHandlersRegistry(filesystem, notifier)
	if err != nil {
		return nil, err
	}

	// TODO: make sure it sits in the right place.
	v1alphaParser.UseStrictDecodingMode = true
	return &Server{
		lspVersion:      lspVersion,
		files:           filesystem,
		documentUpdates: make(chan documentUpdateEvent, 10),
		handlers:        registry,
		conn:            conn,
		notifier:        notifier,
	}, nil
}

type Server struct {
	lspVersion      string
	initialized     atomic.Bool
	conn            *jsonrpc2.Conn
	files           *files.FS
	handlers        *handlersRegistry
	documentUpdates chan documentUpdateEvent
	notifier        *rpcConnectionNotifier

	runDiagnosticsLoopOnce sync.Once
}

type documentUpdateEvent struct {
	Item messages.TextDocumentItem
}

func (s *Server) GetHandlers() map[string]mux.HandlerFunc {
	return map[string]mux.HandlerFunc{
		messages.InitializeMethod:     s.handleInitialize,
		messages.InitializedMethod:    s.handleInitialized,
		messages.ShutdownMethod:       s.handleShutdown,
		messages.DidOpenMethod:        handleParamsOnly(s.handleDidOpen),
		messages.DidCloseMethod:       handleParamsOnly(s.handleDidClose),
		messages.DidSaveMethod:        handleParamsOnly(s.handleDidSave),
		messages.DidChangeMethod:      handleParamsOnly(s.handleDidChange),
		messages.CompletionMethod:     handleParamsOnly(s.handlers.Completion),
		messages.HoverMethod:          handleParamsOnly(s.handlers.Hover),
		messages.CodeActionMethod:     handleParamsOnly(s.handlers.CodeAction),
		messages.ExecuteCommandMethod: handleParamsOnly(s.handlers.ExecuteCommand),
		messages.SetTraceMethod:       handleParamsOnly(s.handleSetTrace),
		messages.LogTraceMethod:       handleParamsOnly(s.handleLogTrace),
		messages.CancelRequestMethod:  handleParamsOnly(s.handleCancelRequest),
	}
}

func (s *Server) handleInitialize(
	ctx context.Context,
	conn *jsonrpc2.Conn,
	req *jsonrpc2.Request,
) (interface{}, error) {
	// TODO: Figure out If we need to do anything with the client capabilities here.
	_, err := parseRequestParameters[messages.InitializeParams](req.Params)
	if err != nil {
		return nil, err
	}

	resp := messages.InitializeResponse{
		Capabilities: messages.ServerCapabilities{
			TextDocumentSync: messages.TextDocumentSyncKindFull,
			CompletionProvider: &messages.CompletionProvider{
				ResolveProvider:   false,
				TriggerCharacters: []string{":"},
			},
			HoverProvider:      true,
			CodeActionProvider: true,
			ExecuteCommandProvider: &messages.ExecuteCommandProvider{
				Commands: codeactions.GetCommandNames(),
			},
		},
		ServerInfo: messages.ServerInfo{
			Name:    config.ServerName,
			Version: s.lspVersion,
		},
	}
	if s.initialized.CompareAndSwap(false, true) {
		s.conn = conn
		s.notifier.conn = conn
	} else {
		slog.ErrorContext(ctx, "connection already initialized")
	}
	return resp, nil
}

func (s *Server) handleInitialized(_ context.Context, _ *jsonrpc2.Conn, _ *jsonrpc2.Request) (interface{}, error) {
	s.runDiagnosticsLoopOnce.Do(func() {
		go s.runDiagnosticsLoop()
	})
	return nil, nil
}

func (s *Server) handleShutdown(_ context.Context, _ *jsonrpc2.Conn, _ *jsonrpc2.Request) (interface{}, error) {
	return nil, s.conn.Close()
}

func (s *Server) handleDidOpen(ctx context.Context, params messages.DidOpenParams) (interface{}, error) {
	if params.TextDocument.LanguageID != languageID {
		return nil, newUnsupportedLanguageError(params.TextDocument.LanguageID)
	}
	if err := s.files.OpenFile(
		ctx,
		params.TextDocument.URI,
		params.TextDocument.Text,
		params.TextDocument.Version,
	); err != nil {
		return nil, err
	}
	s.documentUpdates <- documentUpdateEvent{
		Item: params.TextDocument,
	}
	return nil, nil
}

// handleDidSave is currently a no-op, we're receiving all the changes via [DidChange] method.
func (s *Server) handleDidSave(_ context.Context, _ messages.DidSaveParams) (interface{}, error) {
	return nil, nil
}

func (s *Server) handleDidClose(_ context.Context, params messages.DidCloseParams) (interface{}, error) {
	return nil, s.files.CloseFile(params.TextDocument.URI)
}

func (s *Server) handleDidChange(ctx context.Context, params messages.DidChangeParams) (interface{}, error) {
	if len(params.ContentChanges) != 1 {
		return nil, nil
	}
	if err := s.files.UpdateFile(
		ctx,
		params.TextDocument.URI,
		params.ContentChanges[0].Text,
		params.TextDocument.Version,
	); err != nil {
		return nil, err
	}
	s.documentUpdates <- documentUpdateEvent{
		Item: messages.TextDocumentItem{
			URI:        params.TextDocument.URI,
			LanguageID: languageID,
			Version:    params.TextDocument.Version,
			Text:       params.ContentChanges[0].Text,
		},
	}
	return nil, nil
}

func (s *Server) runDiagnosticsLoop() {
	for update := range s.documentUpdates {
		s.handleSingleUpdateDiagnostics(update)
	}
}

func (s *Server) handleSingleUpdateDiagnostics(update documentUpdateEvent) {
	ctx := logging.ContextAttr(context.Background(),
		slog.String("uri", update.Item.URI),
		slog.String("language", update.Item.LanguageID),
		slog.Int("version", update.Item.Version))
	span, ctx := logging.StartSpan(ctx, "handle_diagnostics")
	defer span.Finish()
	defer func() { recovery.LogPanic(ctx, recover()) }()

	slog.DebugContext(ctx, "evaluating diagnostics")

	params, err := s.handlers.Diagnostics(ctx, update.Item)
	if err != nil {
		slog.ErrorContext(ctx, "failed to diagnose file", slog.Any("error", err))
	}
	slog.DebugContext(ctx, "evaluated diagnostics", slog.Any("params", params))
	if params == nil {
		return
	}

	if err = s.notifier.Notify(ctx, messages.PublishDiagnosticsMethod, params); err != nil {
		slog.ErrorContext(ctx, "failed to send diagnostics", slog.Any("error", err))
	}
}

func (s *Server) handleSetTrace(ctx context.Context, params messages.SetTraceParams) (interface{}, error) {
	// TODO: handleSetTrace
	return nil, nil
}

func (s *Server) handleLogTrace(ctx context.Context, params messages.LogTraceParams) (interface{}, error) {
	// TODO: handleLogTrace
	return nil, nil
}

func (s *Server) handleCancelRequest(ctx context.Context, params messages.CancelRequestParams) (interface{}, error) {
	// TODO: handleCancelRequest
	return nil, nil
}

type rpcConnectionNotifier struct{ conn *jsonrpc2.Conn }

func (r rpcConnectionNotifier) Notify(ctx context.Context, method string, params interface{}) error {
	return r.conn.Notify(ctx, method, params)
}

type paramsOnlyHandlerFunc[T any] func(ctx context.Context, params T) (interface{}, error)

// handleParamsOnly wraps generic [paramsOnlyHandlerFunc] function call with [mux.HandlerFunc].
// It safely decodes the [jsonrpc2.Request.Params] into the expected type T.
func handleParamsOnly[T any](f paramsOnlyHandlerFunc[T]) mux.HandlerFunc {
	return func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
		params, err := parseRequestParameters[T](req.Params)
		if err != nil {
			return nil, err
		}
		return f(ctx, params)
	}
}

func parseRequestParameters[T any](rawParams *json.RawMessage) (params T, err error) {
	if rawParams == nil {
		return params, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "missing request parameters",
		}
	}
	if err = json.Unmarshal(*rawParams, &params); err != nil {
		return params, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("failed to parse request parameters: %v", err),
			Data:    rawParams,
		}
	}
	return params, nil
}

func newUnsupportedLanguageError(lang string) error {
	return &jsonrpc2.Error{
		Code:    jsonrpc2.CodeInvalidParams,
		Message: fmt.Sprintf("unsupported language id: %s", lang),
	}
}
