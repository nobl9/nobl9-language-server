package hover

import (
	"context"
	"log/slog"

	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

type providerInterface interface {
	Hover(
		ctx context.Context,
		params messages.HoverParams,
		node *files.SimpleObjectNode,
		line *yamlastsimple.Line,
	) *messages.HoverResponse
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
	file, err := h.files.GetFile(params.TextDocument.URI)
	if err != nil {
		return nil, err
	}
	if file.Skip {
		slog.DebugContext(ctx, "skipping file")
		return nil, nil
	}

	var (
		node *files.SimpleObjectNode
		line *yamlastsimple.Line
	)
	for _, node = range file.SimpleAST {
		line = node.Doc.FindLine(params.Position.Line - node.Doc.Offset)
		if line != nil {
			break
		}
	}
	if node == nil || line == nil {
		slog.ErrorContext(ctx, "no document found", slog.Any("line", params.Position.Line))
		return nil, nil
	}
	return h.provider.Hover(ctx, params, node, line), nil
}
