package codeactions

import (
	"context"
	"log/slog"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/pkg/errors"

	"github.com/nobl9/nobl9-language-server/internal/files"
	"github.com/nobl9/nobl9-language-server/internal/messages"
)

type clientNotifier interface {
	Notify(ctx context.Context, method string, params any) error
}

type objectsRepo interface {
	Apply(ctx context.Context, objects []manifest.Object) error
	Delete(ctx context.Context, objects []manifest.Object) error
}

func NewHandler(files *files.FS, repo objectsRepo, notifier clientNotifier) *Handler {
	return &Handler{
		files:       files,
		objectsRepo: repo,
		notifier:    notifier,
	}
}

type Handler struct {
	files       *files.FS
	objectsRepo objectsRepo
	notifier    clientNotifier
}

func (h *Handler) HandleCodeAction(_ context.Context, params messages.CodeActionParams) (any, error) {
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

func (h *Handler) HandleExecuteCommand(ctx context.Context, params messages.ExecuteCommandParams) (any, error) {
	uri, ok := params.Arguments[0].(string)
	if !ok {
		return nil, errors.Errorf(
			"invalid arguments: expected URI as the first argument, was: %v",
			params.Arguments)
	}
	file, err := h.files.GetFile(uri)
	if err != nil {
		return nil, err
	}
	ctx = file.AddToLogContext(ctx)
	if file.Skip {
		slog.DebugContext(ctx, "skipping file")
		return nil, nil
	}

	objects := make([]manifest.Object, 0, len(file.Objects))
	for _, obj := range file.Objects {
		objects = append(objects, obj.Object)
	}
	switch params.Command {
	case commandApplyDryRun:
	case commandApply:
		err = h.objectsRepo.Apply(ctx, objects)
	case commandDelete:
		err = h.objectsRepo.Delete(ctx, objects)
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
	return nil, h.notifier.Notify(ctx, messages.ShowMessageMethod, message)
}
