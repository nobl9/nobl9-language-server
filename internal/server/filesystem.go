package server

import (
	"context"
	"fmt"
	"sync"
)

func newFilesystem() *filesystem {
	return &filesystem{
		files: make(map[fileURI]*virtualFile),
		mu:    new(sync.RWMutex),
	}
}

type fileURI = string

type filesystem struct {
	files map[fileURI]*virtualFile
	mu    *sync.RWMutex
}

// GetFile returns a copy of [virtualFile] by its URI.
// Otherwise, if the file gets updated it could result in concurrency issues.
func (f *filesystem) GetFile(uri fileURI) (*virtualFile, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	file, ok := f.files[uri]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", uri)
	}
	return file.copy(), nil
}

func (f *filesystem) HasFile(uri fileURI) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, hasFile := f.files[uri]
	return hasFile
}

func (f *filesystem) CloseFile(uri fileURI) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.files, uri)
	return nil
}

func (f *filesystem) OpenFile(ctx context.Context, uri fileURI, content string, version int) error {
	if f.HasFile(uri) {
		return fmt.Errorf("file already exists: %s", uri)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	file, err := newVirtualFile(ctx, uri, content)
	if err != nil {
		return err
	}
	file.UpdateVersion(version)
	f.files[uri] = file
	return nil
}

func (f *filesystem) UpdateFile(ctx context.Context, uri fileURI, content string, version int) error {
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
