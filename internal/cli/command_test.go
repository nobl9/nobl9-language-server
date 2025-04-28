package cli

import (
	"log/slog"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-language-server/internal/logging"
)

func Test_logLevelFileFlag(t *testing.T) {
	tests := []struct {
		in  string
		out slog.Level
		err error
	}{
		{"INFO", slog.LevelInfo, nil},
		{"DEBUG", slog.LevelDebug, nil},
		{"WARN", slog.LevelWarn, nil},
		{"ERROR", slog.LevelError, nil},
		{"TRACE", logging.LevelTrace, nil},
		{"", 0, errors.New(`slog: level string "": unknown name`)},
		{"invalid", 0, errors.New(`slog: level string "INVALID": unknown name`)},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			cmd := &Command{config: new(Config)}
			err := cmd.parseLogLevelFlag(tc.in)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.out.String(), cmd.config.LogLevel.String())
		})
	}
}

func Test_parseFilePatternsFlag(t *testing.T) {
	tests := []struct {
		in  string
		out []string
		err error
	}{
		{"", nil, nil},
		{"   ", nil, nil},
		{"a,b,c", []string{"a", "b", "c"}, nil},
		{"a,b,\\", nil, errors.New("invalid file pattern: \\")},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			cmd := &Command{config: new(Config)}
			err := cmd.parseFilePatternsFlag(tc.in)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.out, cmd.config.FilePatterns)
		})
	}
}

func Test_parseLogFilePath(t *testing.T) {
	t.Setenv("FOO", "foo")
	t.Setenv("HOME", "bar")

	tests := []struct {
		in  string
		out string
	}{
		{"/path/to/this", "/path/to/this"},
		{"file", "file"},
		{"/path/to/$FOO", "/path/to/foo"},
		{"$FOO/file", "foo/file"},
		{"~/file", "bar/file"},
		{"~/$FOO/to/$FOO", "bar/foo/to/foo"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			cmd := &Command{config: new(Config)}
			err := cmd.parseLogFilePath(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.out, cmd.config.LogFilePath)
		})
	}
}
