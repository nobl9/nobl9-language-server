package files

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
)

func NewFS(filePatterns []string) *FS {
	return &FS{
		files:        make(map[URI]*File),
		filePatterns: filePatterns,
		mu:           new(sync.RWMutex),
	}
}

type FS struct {
	files map[URI]*File
	// filePatterns are assumed to be validated and normalized with [filepath.ToSlash].
	filePatterns []string
	mu           *sync.RWMutex
}

// GetFile returns a copy of [File] by its URI.
// Otherwise, if the file gets updated it could result in concurrency issues.
func (fs *FS) GetFile(uri URI) (*File, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	file, ok := fs.files[uri]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", uri)
	}
	return file.copy(), nil
}

func (fs *FS) HasFile(uri URI) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	_, hasFile := fs.files[uri]
	return hasFile
}

func (fs *FS) CloseFile(uri URI) error {
	if !fs.HasFile(uri) {
		return errors.Errorf("file already closed: %s", uri)
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if _, hasFile := fs.files[uri]; !hasFile {
		return errors.Errorf("file already closed: %s", uri)
	}
	delete(fs.files, uri)
	return nil
}

func (fs *FS) OpenFile(ctx context.Context, uri URI, content string, version int) error {
	if fs.HasFile(uri) {
		return fmt.Errorf("file already exists: %s", uri)
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	file := &File{URI: uri}
	if err := fs.updateFile(ctx, file, content, version); err != nil {
		return err
	}
	fs.files[uri] = file
	return nil
}

func (fs *FS) UpdateFile(ctx context.Context, uri URI, content string, version int) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	file, ok := fs.files[uri]
	if !ok {
		return fmt.Errorf("file not found: %s", uri)
	}
	return fs.updateFile(ctx, file, content, version)
}

func (fs *FS) updateFile(ctx context.Context, file *File, content string, version int) error {
	skipFile, err := fs.shouldSkipFile(file.URI, content)
	if err != nil {
		return err
	}
	if skipFile {
		file.UpdateSkipped(version, content)
		return nil
	}
	file.Update(ctx, version, content)
	return nil
}

const (
	serverActivateComment = "# nobl9-language-server: activate"
	nobl9ApiVersionPrefix = "apiVersion: n9/"
)

func (fs *FS) shouldSkipFile(uri URI, content string) (bool, error) {
	if len(fs.filePatterns) == 0 {
		if strings.HasPrefix(content, serverActivateComment) ||
			strings.Contains(content, nobl9ApiVersionPrefix) {
			return false, nil
		}
		return true, nil
	}

	fileName, err := filePathFromURI(uri)
	if err != nil {
		return true, err
	}
	fileName = filepath.ToSlash(fileName)
	for _, pattern := range fs.filePatterns {
		ok, err := doublestar.Match(pattern, fileName)
		if err != nil {
			return true, err
		}
		// If the file matches the pattern, don't skip it.
		if ok {
			return false, nil
		}
	}
	// No matches found, skip the file.
	return true, nil
}
