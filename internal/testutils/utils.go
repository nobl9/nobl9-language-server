package testutils

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-language-server/internal/files"
)

var (
	moduleRoot string
	once       sync.Once
)

// FindModuleRoot returns the absolute path to the modules root.
func FindModuleRoot() string {
	once.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		dir = filepath.Clean(dir)
		for {
			if fi, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !fi.IsDir() {
				moduleRoot = dir
				break
			}
			d := filepath.Dir(dir)
			if d == dir {
				break
			}
			dir = d
		}
	})
	return moduleRoot
}

// RegisterTestFiles registers all files from the specified directory in the provided [files.FS].
func RegisterTestFiles(t *testing.T, fs *files.FS, testFileDir string) {
	t.Helper()

	entries, err := os.ReadDir(testFileDir)
	require.NoError(t, err)
	for _, entry := range entries {
		require.False(t, entry.IsDir())
		path := filepath.Join(testFileDir, entry.Name())
		data, err := os.ReadFile(path) // #nosec G304
		require.NoError(t, err)
		err = fs.OpenFile(context.Background(), path, string(data), 1)
		require.NoError(t, err)
	}
}
