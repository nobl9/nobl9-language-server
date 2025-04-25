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
		uri      URI
		setup    func(fs *FS)
		expected map[URI]*File
		error    error
	}{
		{
			name: "close existing file",
			uri:  "file1",
			setup: func(fs *FS) {
				fs.files["file1"] = &File{}
				fs.files["file2"] = &File{}
			},
			expected: map[URI]*File{
				"file1": {URI: "file1"},
			},
		},
		{
			name:  "close non-existing file",
			uri:   "file2",
			setup: func(fs *FS) {},
			expected: map[URI]*File{
				"file1": {URI: "file1"},
			},
			error: errors.New("file already closed: file2"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := NewFS(nil)
			tc.setup(fs)
			err := fs.CloseFile(tc.uri)
			if tc.error != nil {
				assert.EqualError(t, err, tc.error.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFS_GetFile(t *testing.T) {
	tests := []struct {
		name     string
		uri      URI
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := NewFS(nil)
			tc.setup(fs)
			file, err := fs.GetFile(tc.uri)
			if tc.error != nil {
				assert.EqualError(t, err, tc.error.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, file)
			}
		})
	}
}

func TestFS_HasFile(t *testing.T) {
	tests := []struct {
		name   string
		uri    URI
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := NewFS(nil)
			tc.setup(fs)
			assert.Equal(t, tc.expect, fs.HasFile(tc.uri))
		})
	}
}

func TestFS_OpenFile(t *testing.T) {
	tests := []struct {
		name         string
		uri          URI
		content      string
		version      int
		filePatterns []string
		skipped      bool
		setup        func(fs *FS)
		error        error
	}{
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
		{
			name:    "skipped with no Nobl9 apiVersion",
			uri:     "file1",
			content: "content",
			version: 1,
			skipped: true,
			setup:   func(fs *FS) {},
		},
		{
			name:         "invalid URI (pattern matching check)",
			uri:          "file1",
			content:      "content",
			version:      1,
			filePatterns: []string{"file0"},
			setup:        func(fs *FS) {},
			error:        errors.New(`failed to parse URI file1: parse "file1": invalid URI for request`),
		},
		{
			name:         "matching file pattern (name)",
			uri:          "file://file1",
			content:      "content",
			version:      1,
			filePatterns: []string{"file0", "file1"},
			setup:        func(fs *FS) {},
		},
		{
			name:         "matching file pattern (pattern)",
			uri:          "file:///home/me/nobl9/file1",
			content:      "content",
			version:      1,
			filePatterns: []string{"**/nobl9/*"},
			setup:        func(fs *FS) {},
		},
		{
			name:    "has Nobl9 apiVersion",
			uri:     "file://file2",
			content: "apiVersion: n9/v1alpha",
			version: 1,
			setup:   func(fs *FS) {},
		},
		{
			name:    "has server comment",
			uri:     "file://file2",
			content: "# nobl9-language-server: activate\ncontent",
			version: 1,
			setup:   func(fs *FS) {},
		},
		{
			name:         "has Nobl9 apiVersion but skipped, does not match pattern",
			uri:          "file://file2",
			content:      "apiVersion: n9/v1alpha",
			version:      1,
			filePatterns: []string{"file0", "file1"},
			skipped:      true,
			setup:        func(fs *FS) {},
		},
		{
			name:         "has server comment but skipped, does not match pattern",
			uri:          "file://file2",
			content:      "# nobl9-language-server: activate\ncontent",
			version:      1,
			filePatterns: []string{"file0", "file1"},
			skipped:      true,
			setup:        func(fs *FS) {},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := NewFS(tc.filePatterns)
			tc.setup(fs)

			err := fs.OpenFile(context.Background(), tc.uri, tc.content, tc.version)
			switch {
			case tc.error != nil:
				assert.EqualError(t, err, tc.error.Error())
			default:
				require.NoError(t, err)
				file, err := fs.GetFile(tc.uri)
				require.NoError(t, err)

				assert.Equal(t, tc.uri, file.URI, "uri")
				assert.Equal(t, tc.content, file.Content, "content")
				assert.Equal(t, tc.version, file.Version, "version")
				assert.Equal(t, tc.skipped, file.Skip, "skipped")
			}
		})
	}
}

func TestFS_UpdateFile(t *testing.T) {
	tests := []struct {
		name            string
		uri             URI
		content         string
		expectedContent string
		version         int
		filePatterns    []string
		skipped         bool
		setup           func(fs *FS)
		error           error
	}{
		{
			name:            "update existing file",
			uri:             "file://file1",
			content:         "apiVersion: n9/v1alpha",
			expectedContent: "apiVersion: n9/v1alpha",
			version:         2,
			setup: func(fs *FS) {
				fs.files["file://file1"] = &File{
					URI:     "file://file1",
					Version: 1,
					Content: "old content",
				}
			},
		},
		{
			name:    "update non-existing file",
			uri:     "file://file2",
			content: "new content",
			version: 2,
			setup:   func(fs *FS) {},
			error:   errors.New("file not found: file://file2"),
		},
		{
			name:            "update with same version",
			uri:             "file://file1",
			content:         "new content",
			expectedContent: "old content",
			version:         1,
			setup: func(fs *FS) {
				fs.files["file://file1"] = &File{
					URI:     "file://file1",
					Version: 1,
					Content: "old content",
				}
			},
		},
		{
			name:            "skipped with no Nobl9 apiVersion",
			uri:             "file://file1",
			content:         "content",
			expectedContent: "content",
			version:         2,
			skipped:         true,
			setup: func(fs *FS) {
				fs.files["file://file1"] = &File{
					URI:     "file://file1",
					Version: 1,
					Content: "old content",
				}
			},
		},
		{
			name:         "invalid URI (pattern matching check)",
			uri:          "file1",
			content:      "content",
			version:      2,
			filePatterns: []string{"file0"},
			setup: func(fs *FS) {
				fs.files["file1"] = &File{
					URI:     "file1",
					Version: 1,
					Content: "old content",
				}
			},
			error: errors.New(`failed to parse URI file1: parse "file1": invalid URI for request`),
		},
		{
			name:            "matching file pattern (name)",
			uri:             "file://file1",
			content:         "content",
			expectedContent: "content",
			version:         2,
			filePatterns:    []string{"file0", "file1"},
			setup: func(fs *FS) {
				fs.files["file://file1"] = &File{
					URI:     "file://file1",
					Version: 1,
					Content: "old content",
				}
			},
		},
		{
			name:            "matching file pattern (pattern)",
			uri:             "file:///home/me/nobl9/file1",
			content:         "content",
			expectedContent: "content",
			version:         2,
			filePatterns:    []string{"**/nobl9/*"},
			setup: func(fs *FS) {
				fs.files["file:///home/me/nobl9/file1"] = &File{
					URI:     "file:///home/me/nobl9/file1",
					Version: 1,
					Content: "old content",
				}
			},
		},
		{
			name:            "has Nobl9 apiVersion",
			uri:             "file://file1",
			content:         "apiVersion: n9/v1alpha",
			expectedContent: "apiVersion: n9/v1alpha",
			version:         2,
			setup: func(fs *FS) {
				fs.files["file://file1"] = &File{
					URI:     "file://file1",
					Version: 1,
					Content: "old content",
				}
			},
		},
		{
			name:            "has server comment",
			uri:             "file://file1",
			content:         "# nobl9-language-server: activate\ncontent",
			expectedContent: "# nobl9-language-server: activate\ncontent",
			version:         2,
			setup: func(fs *FS) {
				fs.files["file://file1"] = &File{
					URI:     "file://file1",
					Version: 1,
					Content: "old content",
				}
			},
		},
		{
			name:            "has Nobl9 apiVersion but skipped, does not match pattern",
			uri:             "file://file2",
			content:         "apiVersion: n9/v1alpha",
			expectedContent: "apiVersion: n9/v1alpha",
			version:         2,
			filePatterns:    []string{"file0", "file1"},
			skipped:         true,
			setup: func(fs *FS) {
				fs.files["file://file2"] = &File{
					URI:     "file://file2",
					Version: 1,
					Content: "old content",
				}
			},
		},
		{
			name:            "has server comment but skipped, does not match pattern",
			uri:             "file://file2",
			content:         "# nobl9-language-server: activate\ncontent",
			expectedContent: "# nobl9-language-server: activate\ncontent",
			version:         2,
			filePatterns:    []string{"file0", "file1"},
			skipped:         true,
			setup: func(fs *FS) {
				fs.files["file://file2"] = &File{
					URI:     "file://file2",
					Version: 1,
					Content: "old content",
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := NewFS(tc.filePatterns)
			tc.setup(fs)

			err := fs.UpdateFile(context.Background(), tc.uri, tc.content, tc.version)
			switch {
			case tc.error != nil:
				assert.EqualError(t, err, tc.error.Error())
			default:
				require.NoError(t, err)
				file, err := fs.GetFile(tc.uri)
				require.NoError(t, err)

				assert.Equal(t, tc.uri, file.URI, "uri")
				assert.Equal(t, tc.expectedContent, file.Content, "content")
				assert.Equal(t, tc.version, file.Version, "version")
				assert.Equal(t, tc.skipped, file.Skip, "skipped")
			}
		})
	}
}
