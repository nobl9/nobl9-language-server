package server

import (
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"

	"github.com/nobl9/nobl9-language-server/internal/codeactions"
	"github.com/nobl9/nobl9-language-server/internal/completion"
	"github.com/nobl9/nobl9-language-server/internal/diagnostics"
	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/hover"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/nobl9repo"
	"github.com/nobl9/nobl9-language-server/internal/sdkdocs"
)

type handlersRegistry struct {
	Diagnostics    paramsOnlyHandlerFunc[messages.TextDocumentItem]
	Completion     paramsOnlyHandlerFunc[messages.CompletionParams]
	Hover          paramsOnlyHandlerFunc[messages.HoverParams]
	CodeAction     paramsOnlyHandlerFunc[messages.CodeActionParams]
	ExecuteCommand paramsOnlyHandlerFunc[messages.ExecuteCommandParams]
}

func newHandlersRegistry(
	filesystem *files.FS,
	sdkClient *sdk.Client,
	notifier *rpcConnectionNotifier,
) (*handlersRegistry, error) {
	// Common dependencies.
	objectsRepo := nobl9repo.NewRepo(sdkClient)
	sdkDocs, err := sdkdocs.New()
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup SDK docs provider")
	}

	// Diagnostics.
	diagnosticsProvider := diagnostics.NewProvider(sdkDocs, objectsRepo)
	diagnosticsHandler := diagnostics.NewHandler(filesystem, diagnosticsProvider)
	// Completion.
	completionHandler := completion.NewHandler(filesystem,
		completion.NewValuesCompletionProvider(sdkDocs),
		completion.NewKeysCompletionProvider(sdkDocs),
		completion.NewReferencesCompletionProvider(objectsRepo),
		completion.NewSnippetsProvider(),
	)
	// Hover.
	hoverProvider := hover.NewProvider(sdkDocs)
	hoverHandler := hover.NewHandler(filesystem, hoverProvider)
	// Code actions.
	codeActionsHandler := codeactions.NewHandler(filesystem, objectsRepo, notifier)

	return &handlersRegistry{
		Diagnostics:    diagnosticsHandler.Handle,
		Completion:     completionHandler.Handle,
		Hover:          hoverHandler.Handle,
		CodeAction:     codeActionsHandler.HandleCodeAction,
		ExecuteCommand: codeActionsHandler.HandleExecuteCommand,
	}, nil
}
