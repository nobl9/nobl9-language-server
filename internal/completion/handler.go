package completion

import (
	"context"
	"log/slog"

	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/yamlastsimple"
)

type completionProviderType int

const (
	keysCompletionType completionProviderType = iota + 1
	valuesCompletionType
)

type providerInterface interface {
	Complete(
		ctx context.Context,
		params messages.CompletionParams,
		file files.SimpleObjectFile,
		node *files.SimpleObjectNode,
		line *yamlastsimple.Line,
	) []messages.CompletionItem
	getType() completionProviderType
}

func NewHandler(filesystem *files.FS, providers ...providerInterface) *Handler {
	return &Handler{
		files:     filesystem,
		providers: providers,
	}
}

type Handler struct {
	files     *files.FS
	providers []providerInterface
}

func (h *Handler) Handle(ctx context.Context, params messages.CompletionParams) (any, error) {
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

	var items []messages.CompletionItem
	for _, provider := range h.providers {
		switch provider.getType() {
		case keysCompletionType:
			start, end := line.GetKeyPos()
			if !line.IsType(yamlastsimple.LineTypeEmpty) &&
				(params.Position.Character < start || params.Position.Character > end) {
				continue
			}
		case valuesCompletionType:
			if _, end := line.GetKeyPos(); params.Position.Character <= end {
				continue
			}
		}
		items = append(items, provider.Complete(ctx, params, file.SimpleAST, node, line)...)
	}
	return items, nil
}
