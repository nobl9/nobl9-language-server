package files

import (
	"context"
	"fmt"
	"sync"
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
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.files, uri)
	return nil
}

func (f *FS) OpenFile(ctx context.Context, uri fileURI, content string, version int) error {
	if f.HasFile(uri) {
		return fmt.Errorf("file already exists: %s", uri)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	file, err := NewFile(ctx, uri, content)
	if err != nil {
		return err
	}
	file.UpdateVersion(version)
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
	// Version did not change, no need to update.
	if version == file.Version {
		return nil
	}
	if version >= 0 {
		file.UpdateVersion(version)
	}
	if err := file.Update(ctx, content); err != nil {
		return err
	}
	return nil
}
