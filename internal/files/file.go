package files

import (
	"context"
	"log/slog"

	"github.com/nobl9/nobl9-language-server/internal/logging"
	"github.com/nobl9/nobl9-language-server/internal/yamlast"
)

// File is a virtual representation of an [os.File].
type File struct {
	URI URI
	// Content is the file content.
	Content string
	// Version is the file version reported by the client.
	Version int
	// Skip is true if the file should be skipped from performing any kind of action.
	Skip      bool
	Objects   []*ObjectNode
	SimpleAST SimpleObjectFile
	// Err is the error that occurred while parsing the file AST (if any).
	Err error
}

// AddToLogContext adds basic file details to the logging context.
func (f *File) AddToLogContext(ctx context.Context) context.Context {
	return logging.ContextAttr(ctx,
		slog.String("fileURI", f.URI),
		slog.Int("fileVersion", f.Version))
}

// FindObject returns the [ObjectNode] which spans over the specified line.
func (f *File) FindObject(line int) *ObjectNode {
	for _, object := range f.Objects {
		if line >= object.Node.StartLine && line <= object.Node.EndLine {
			return object
		}
	}
	return nil
}

func (f *File) Update(ctx context.Context, version int, content string) {
	if !f.shouldUpdate(ctx, version) {
		return
	}
	f.Version = version
	f.Content = content

	f.SimpleAST, f.Err = ParseSimpleObjectFile(content)
	if f.Err != nil {
		return
	}

	fileAST, err := yamlast.Parse(content)
	f.Err = err
	if err != nil {
		return
	}
	f.Objects = make([]*ObjectNode, 0, len(fileAST.Nodes))
	for _, node := range fileAST.Nodes {
		f.Objects = append(f.Objects, parseObjectNode(ctx, node))
	}
}

// UpdateSkipped updates the file version and content without doing any parsing.
// It also sets the [File.Skip] flag to true.
func (f *File) UpdateSkipped(ctx context.Context, version int, content string) {
	if !f.shouldUpdate(ctx, version) {
		return
	}
	f.Version = version
	f.Content = content
	f.Skip = true
}

// shouldUpdate checks if version has changed to a newer one.
// If not there's no need to update the file.
func (f *File) shouldUpdate(ctx context.Context, version int) bool {
	shouldUpdate := version > f.Version
	if !shouldUpdate {
		slog.DebugContext(ctx, "file version has not changed, skipping update")
	}
	return shouldUpdate
}

func (f *File) copy() *File {
	objects := make([]*ObjectNode, 0, len(f.Objects))
	for _, object := range f.Objects {
		objects = append(objects, object.copy())
	}
	return &File{
		URI:       f.URI,
		Content:   f.Content,
		Version:   f.Version,
		Skip:      f.Skip,
		Objects:   objects,
		SimpleAST: f.SimpleAST,
		Err:       f.Err,
	}
}
