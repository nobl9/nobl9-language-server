package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/nobl9/nobl9-go/manifest"
	v1alphaParser "github.com/nobl9/nobl9-go/manifest/v1alpha/parser"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/nobl9/nobl9-language-server/internal/logging"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/mux"
	"github.com/nobl9/nobl9-language-server/internal/recovery"
	"github.com/nobl9/nobl9-language-server/internal/sdkclient"
	"github.com/nobl9/nobl9-language-server/internal/sdkdocs"
	"github.com/nobl9/nobl9-language-server/internal/yamlastfast"
)

const (
	lspName    = "nobl9-language-server"
	languageID = "yaml"
)

type pathCompletionProviderI interface {
	Complete(
		kind manifest.Kind,
		line *yamlastfast.Line,
		position messages.Position,
	) []messages.CompletionItem
}

type completionProviderI interface {
	Complete(
		cmpCtx messages.CompletionContext,
		kind manifest.Kind,
		path string,
	) []messages.CompletionItem
}

func New(ctx context.Context, lspVersion string) (*Server, error) {
	span, _ := logging.StartSpan(ctx, "server_start")
	defer span.Finish()

	sdkClient, err := sdk.DefaultClient()
	if err != nil {
		return nil, err
	}
	sdkDocs, err := sdkdocs.New()
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup SDK docs provider")
	}
	// TODO: make sure it sits in the right place.
	v1alphaParser.UseStrictDecodingMode = true
	return &Server{
		lspVersion:              lspVersion,
		files:                   newFilesystem(),
		updates:                 make(chan documentUpdateEvent, 10),
		completionProvider:      newCompletionProvidersRegistry(sdkClient),
		pathsCompletionProvider: newPathsCompletionProvider(sdkDocs),
		snippetsProvider:        newSnippetsProvider(),
		hoverProvider:           newDocsHoverProvider(sdkDocs),
		diagnostics:             newDiagnosticsProvider(sdkDocs, sdkclient.New(sdkClient)),
		sdkClient:               sdkClient,
	}, nil
}

type Server struct {
	lspVersion              string
	initialized             atomic.Bool
	conn                    *jsonrpc2.Conn
	files                   *filesystem
	updates                 chan documentUpdateEvent
	completionProvider      completionProviderI
	pathsCompletionProvider pathCompletionProviderI
	snippetsProvider        *snippetsProvider
	hoverProvider           hoverProvider
	diagnostics             *diagnosticsProvider
	sdkClient               *sdk.Client

	runDiagnosticsLoopOnce sync.Once
}

type documentUpdateEvent struct {
	Context context.Context
	Item    messages.TextDocumentItem
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
		messages.CompletionMethod:     handleParamsOnly(s.handleCompletion),
		messages.HoverMethod:          handleParamsOnly(s.handleHover),
		messages.CodeActionMethod:     handleParamsOnly(s.handleCodeAction),
		messages.ExecuteCommandMethod: handleParamsOnly(s.handleExecuteCommand),
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
				ResolveProvider: false,
			},
			HoverProvider:      true,
			CodeActionProvider: true,
			ExecuteCommandProvider: &messages.ExecuteCommandProvider{
				Commands: codeActionCommandNames,
			},
		},
		ServerInfo: messages.ServerInfo{
			Name:    lspName,
			Version: s.lspVersion,
		},
	}
	if s.initialized.CompareAndSwap(false, true) {
		s.conn = conn
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
	s.updates <- documentUpdateEvent{
		Context: ctx,
		Item:    params.TextDocument,
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
	s.updates <- documentUpdateEvent{
		Context: ctx,
		Item: messages.TextDocumentItem{
			URI:     params.TextDocument.URI,
			Version: params.TextDocument.Version,
			Text:    params.ContentChanges[0].Text,
		},
	}
	return nil, nil
}

func (s *Server) handleCompletion(ctx context.Context, params messages.CompletionParams) (interface{}, error) {
	file, err := s.files.GetFile(params.TextDocument.URI)
	if err != nil {
		return nil, err
	}
	var (
		doc  *fastObjectNode
		line *yamlastfast.Line
	)
	for _, doc = range file.FastAST {
		line = doc.Doc.FindLine(params.Position.Line - doc.Doc.Offset)
		if line != nil {
			break
		}
	}
	if doc == nil || line == nil {
		slog.ErrorContext(ctx, "no document found", slog.Any("line", params.Position.Line))
		return nil, nil
	}

	items := s.pathsCompletionProvider.Complete(doc.Kind, line, params.Position)
	items = append(items, s.completionProvider.Complete(params.CompletionContext, doc.Kind, line.Path)...)
	items = append(items, s.snippetsProvider.Complete(file.FastAST, params.Position.Line, line)...)
	return items, nil
}

func (s *Server) handleHover(ctx context.Context, params messages.HoverParams) (interface{}, error) {
	// Change from 0-based to 1-based line number.
	params.Position.Line++

	object, err := s.findObject(ctx, params.TextDocument.URI, params.Position.Line)
	if err != nil || object == nil {
		return nil, err
	}
	node, err := object.Node.Find(params.Position.Line)
	if err != nil {
		return nil, err
	}
	return s.hoverProvider.Hover(object.Object.GetKind(), node.GetPath()), nil
}

const (
	commandApply       = "APPLY"
	commandDelete      = "DELETE"
	commandApplyDryRun = "APPLY_DRY_RUN"
)

var codeActionCommandNames = []string{
	commandApply,
	commandApplyDryRun,
	commandDelete,
}

var codeActionCommands = map[string]struct {
	Title          string
	FailedMessage  string
	SuccessMessage string
}{
	commandApply: {
		Title:          "Apply objects defined in this file",
		FailedMessage:  "Failed to apply objects",
		SuccessMessage: "Objects applied successfully",
	},
	commandApplyDryRun: {
		Title:          "Apply objects defined in this file (dry-run)",
		FailedMessage:  "Failed to apply objects (dry-run)",
		SuccessMessage: "Objects applied successfully (dry-run)",
	},
	commandDelete: {
		Title:          "Delete objects defined in this file",
		FailedMessage:  "Failed to delete objects",
		SuccessMessage: "Objects deleted successfully",
	},
}

func (s *Server) handleCodeAction(_ context.Context, params messages.CodeActionParams) (interface{}, error) {
	actions := make([]messages.Command, 0, len(codeActionCommands))
	for _, cmdName := range codeActionCommandNames {
		cmd := codeActionCommands[cmdName]
		actions = append(actions, messages.Command{
			Title:     cmd.Title,
			Command:   cmdName,
			Arguments: []any{params.TextDocument.URI},
		})
	}
	return actions, nil
}

func (s *Server) handleExecuteCommand(ctx context.Context, params messages.ExecuteCommandParams) (interface{}, error) {
	uri, ok := params.Arguments[0].(string)
	if !ok {
		return nil, errors.Errorf(
			"invalid arguments: expected URI as the first argument, was: %v",
			params.Arguments)
	}
	file, err := s.files.GetFile(uri)
	if err != nil {
		return nil, err
	}
	objects := make([]manifest.Object, 0, len(file.Objects))
	for _, obj := range file.Objects {
		objects = append(objects, obj.Object)
	}
	switch params.Command {
	case commandApplyDryRun:
	case commandApply:
		err = s.sdkClient.Objects().V1().Apply(ctx, objects)
	case commandDelete:
		err = s.sdkClient.Objects().V1().Delete(ctx, objects)
	default:
		return nil, errors.New("unknown command: " + params.Command)
	}

	var message messages.ShowMessageParams
	if err != nil {
		message = messages.ShowMessageParams{
			Type:    messages.MessageTypeError,
			Message: codeActionCommands[params.Command].FailedMessage,
		}
	} else {
		message = messages.ShowMessageParams{
			Type:    messages.MessageTypeInfo,
			Message: codeActionCommands[params.Command].SuccessMessage,
		}
	}
	return nil, s.conn.Notify(ctx, messages.ShowMessageMethod, message)
}

func (s *Server) runDiagnosticsLoop() {
	for update := range s.updates {
		s.handleSingleUpdateDiagnostics(update)
	}
}

func (s *Server) handleSingleUpdateDiagnostics(update documentUpdateEvent) {
	defer func() {
		recovery.LogPanic(update.Context, s.conn, recover())
	}()

	params, err := s.handleDiagnostics(update.Context, update.Item)
	if err != nil {
		slog.ErrorContext(update.Context, "failed to diagnose file", slog.Any("error", err))
	}
	if params == nil {
		return
	}
	if err = s.conn.Notify(update.Context, messages.PublishDiagnosticsMethod, params); err != nil {
		slog.ErrorContext(update.Context, "failed to send diagnostics", slog.Any("error", err))
	}
}

func (s *Server) handleDiagnostics(
	ctx context.Context,
	item messages.TextDocumentItem,
) (*messages.PublishDiagnosticsParams, error) {
	if item.Text == "" {
		return nil, nil
	}
	file, err := s.files.GetFile(item.URI)
	if err != nil {
		return nil, err
	}
	params := messages.PublishDiagnosticsParams{
		URI:     item.URI,
		Version: item.Version,
	}
	diags := s.diagnostics.DiagnoseFile(ctx, file)
	// We need to send empty diagnostics to clear the previous ones.
	// Otherwise, once an error occurs it will never be cleared even if the user fixes the issue.
	if len(diags) == 0 {
		diags = make([]messages.Diagnostic, 0)
	}
	// If the file has changed we don't want to send diagnostics for the old version.
	if file, err = s.files.GetFile(item.URI); err == nil && file.Version != item.Version {
		return nil, nil
	}
	params.Diagnostics = diags
	return &params, nil
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

func (s *Server) findObject(ctx context.Context, uri string, line int) (*objectNode, error) {
	file, err := s.files.GetFile(uri)
	if err != nil {
		return nil, err
	}
	object := file.FindObject(line)
	if object == nil {
		slog.WarnContext(ctx, "no object found", slog.Any("line", line))
		return nil, nil
	}
	// If there was an error parsing the object, we can't provide completions.
	if object.Err != nil {
		return nil, nil // nolint: nilerr
	}
	return object, nil
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

func ptr[T any](v T) *T { return &v }
