package files

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

func NewFS() *FS {
	return &FS{
		files: make(map[fileURI]*File),
		mu:    new(sync.RWMutex),
	}
}

type fileURI = string

type FS struct {
	files map[fileURI]*File
	mu    *sync.RWMutex
}

// GetFile returns a copy of [File] by its URI.
// Otherwise, if the file gets updated it could result in concurrency issues.
func (f *FS) GetFile(uri fileURI) (*File, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	file, ok := f.files[uri]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", uri)
	}
	return file.copy(), nil
}

func (f *FS) HasFile(uri fileURI) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, hasFile := f.files[uri]
	return hasFile
}

func (f *FS) CloseFile(uri fileURI) error {
	if !f.HasFile(uri) {
		return errors.New(uri + " file already closed")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, hasFile := f.files[uri]; !hasFile {
		return errors.New(uri + " file already closed")
	}
	delete(f.files, uri)
	return nil
}

func (f *FS) OpenFile(ctx context.Context, uri fileURI, content string, version int) error {
	if f.HasFile(uri) {
		return fmt.Errorf("file already exists: %s", uri)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	file := NewFile(ctx, uri, version, content)
	f.files[uri] = file
	return nil
}

func (f *FS) UpdateFile(ctx context.Context, uri fileURI, content string, version int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	file, ok := f.files[uri]
	if !ok {
		return fmt.Errorf("file not found: %s", uri)
	}
	file.Update(ctx, version, content)
	return nil
}
