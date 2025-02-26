package diagnostics

import (
	"context"

	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
)

type providerInterface interface {
	DiagnoseFile(ctx context.Context, file *files.File) []messages.Diagnostic
}

func NewHandler(fs *files.FS, provider providerInterface) *Handler {
	return &Handler{
		fs:          fs,
		diagnostics: provider,
	}
}

type Handler struct {
	fs          *files.FS
	diagnostics providerInterface
}

func (h *Handler) Handle(ctx context.Context, item messages.TextDocumentItem) (any, error) {
	if item.Text == "" {
		return nil, nil
	}
	file, err := h.fs.GetFile(item.URI)
	if err != nil {
		return nil, err
	}
	params := messages.PublishDiagnosticsParams{
		URI:     item.URI,
		Version: item.Version,
	}
	diags := h.diagnostics.DiagnoseFile(ctx, file)
	// We need to send empty diagnostics to clear the previous ones.
	// Otherwise, once an error occurs it will never be cleared even if the user fixes the issue.
	if len(diags) == 0 {
		diags = make([]messages.Diagnostic, 0)
	}
	// If the file has changed we don't want to send diagnostics for the old version.
	if file, err = h.fs.GetFile(item.URI); err == nil && file.Version != item.Version {
		return nil, nil
	}
	params.Diagnostics = diags
	return &params, nil
}
