package hover

import (
	"context"
	"log/slog"

	"github.com/nobl9/nobl9-go/manifest"

	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
)

type providerInterface interface {
	Hover(kind manifest.Kind, path string) *messages.HoverResponse
}

func NewHandler(files *files.FS, provider providerInterface) *Handler {
	return &Handler{
		files:    files,
		provider: provider,
	}
}

type Handler struct {
	files    *files.FS
	provider providerInterface
}

func (h *Handler) Handle(ctx context.Context, params messages.HoverParams) (any, error) {
	// Change from 0-based to 1-based line number.
	params.Position.Line++

	object, err := h.findObject(ctx, params.TextDocument.URI, params.Position.Line)
	if err != nil || object == nil {
		return nil, err
	}
	node, err := object.Node.Find(params.Position.Line)
	if err != nil {
		return nil, err
	}
	return h.provider.Hover(object.Object.GetKind(), node.GetPath()), nil
}

func (h *Handler) findObject(ctx context.Context, uri string, line int) (*files.ObjectNode, error) {
	file, err := h.files.GetFile(uri)
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
