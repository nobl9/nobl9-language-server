package files

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFS_CloseFile(t *testing.T) {
	tests := []struct {
		name     string
		uri      fileURI
		setup    func(fs *FS)
		expected map[fileURI]*File
		error    error
	}{
		{
			name: "close existing file",
			uri:  "file1",
			setup: func(fs *FS) {
				fs.files["file1"] = &File{}
				fs.files["file2"] = &File{}
			},
			expected: map[fileURI]*File{
				"file1": {URI: "file1"},
			},
		},
		{
			name:  "close non-existing file",
			uri:   "file2",
			setup: func(fs *FS) {},
			expected: map[fileURI]*File{
				"file1": {URI: "file1"},
			},
			error: errors.New("file2 file already closed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFS()
			tt.setup(fs)
			err := fs.CloseFile(tt.uri)
			if tt.error != nil {
				assert.EqualError(t, err, tt.error.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFS_GetFile(t *testing.T) {
	tests := []struct {
		name     string
		uri      fileURI
		setup    func(fs *FS)
		error    error
		expected *File
	}{
		{
			name: "get existing file",
			uri:  "file1",
			setup: func(fs *FS) {
				fs.files["file1"] = &File{URI: "file1"}
			},
			expected: &File{URI: "file1", Objects: make([]*ObjectNode, 0)},
		},
		{
			name:  "get non-existing file",
			uri:   "file2",
			setup: func(fs *FS) {},
			error: errors.New("file not found: file2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFS()
			tt.setup(fs)
			file, err := fs.GetFile(tt.uri)
			if tt.error != nil {
				assert.EqualError(t, err, tt.error.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, file)
			}
		})
	}
}

func TestFS_HasFile(t *testing.T) {
	tests := []struct {
		name   string
		uri    fileURI
		setup  func(fs *FS)
		expect bool
	}{
		{
			name: "file exists",
			uri:  "file1",
			setup: func(fs *FS) {
				fs.files["file1"] = &File{}
				fs.files["file2"] = &File{}
			},
			expect: true,
		},
		{
			name:   "file does not exist",
			uri:    "file2",
			setup:  func(fs *FS) { fs.files["file1"] = &File{} },
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFS()
			tt.setup(fs)
			assert.Equal(t, tt.expect, fs.HasFile(tt.uri))
		})
	}
}

func TestFS_OpenFile(t *testing.T) {
	tests := []struct {
		name    string
		uri     fileURI
		content string
		version int
		setup   func(fs *FS)
		error   error
	}{
		{
			name:    "open new file",
			uri:     "file1",
			content: "content",
			version: 1,
			setup:   func(fs *FS) {},
		},
		{
			name:    "open existing file",
			uri:     "file1",
			content: "content",
			version: 1,
			setup: func(fs *FS) {
				fs.files["file1"] = &File{}
			},
			error: errors.New("file already exists: file1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFS()
			tt.setup(fs)
			err := fs.OpenFile(context.Background(), tt.uri, tt.content, tt.version)
			switch {
			case tt.error != nil:
				assert.EqualError(t, err, tt.error.Error())
			default:
				require.NoError(t, err)
				file, err := fs.GetFile(tt.uri)
				require.NoError(t, err)
				assert.Equal(t, tt.uri, file.URI)
				assert.Equal(t, tt.content, file.Content)
				assert.Equal(t, tt.version, file.Version)
			}
		})
	}
}

func TestFS_UpdateFile(t *testing.T) {
	tests := []struct {
		name            string
		uri             fileURI
		content         string
		expectedContent string
		version         int
		setup           func(fs *FS)
		error           error
	}{
		{
			name:            "update existing file",
			uri:             "file1",
			content:         "new content",
			expectedContent: "new content",
			version:         2,
			setup: func(fs *FS) {
				fs.files["file1"] = &File{
					URI:     "file1",
					Version: 1,
					Content: "old content",
				}
			},
		},
		{
			name:    "update non-existing file",
			uri:     "file2",
			content: "new content",
			version: 2,
			setup:   func(fs *FS) {},
			error:   errors.New("file not found: file2"),
		},
		{
			name:            "update with same version",
			uri:             "file1",
			content:         "new content",
			expectedContent: "old content",
			version:         1,
			setup: func(fs *FS) {
				fs.files["file1"] = &File{
					URI:     "file1",
					Version: 1,
					Content: "old content",
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFS()
			tt.setup(fs)
			err := fs.UpdateFile(context.Background(), tt.uri, tt.content, tt.version)
			switch {
			case tt.error != nil:
				assert.EqualError(t, err, tt.error.Error())
			default:
				require.NoError(t, err)
				file, err := fs.GetFile(tt.uri)
				require.NoError(t, err)
				assert.Equal(t, tt.uri, file.URI)
				assert.Equal(t, tt.expectedContent, file.Content)
				assert.Equal(t, tt.version, file.Version)
			}
		})
	}
}
