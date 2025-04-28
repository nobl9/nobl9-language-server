package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v3"

	"github.com/nobl9/nobl9-language-server/internal/config"
	"github.com/nobl9/nobl9-language-server/internal/logging"
	"github.com/nobl9/nobl9-language-server/internal/version"
)

const envPrefix = "NOBL9_LANGUAGE_SERVER_"

type Command struct {
	app    *cli.Command
	config *Config
}

type Config struct {
	LogLevel     logging.Level
	LogFilePath  string
	FilePatterns []string
}

func New(mainFunc func(*Config) error) *Command {
	cmd := &Command{
		config: new(Config),
	}

	cmd.app = &cli.Command{
		Name:  config.ServerName,
		Usage: "Language server implementing LSP (Language Server Protocol) for Nobl9 configuration files",
		Description: `LSP stands for Language Server Protocol.
It defines the protocol used between an editor or an IDE and a language server (this program).
It provides language features like auto complete, diagnose file, display documentation etc.

To learn more about Nobl9 configuration schema, visit: https://docs.nobl9.com/yaml-guide`,
		Action: func(context.Context, *cli.Command) error { return mainFunc(cmd.config) },
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "logLevel",
				Usage:   "Log messages at the provided level",
				Value:   "INFO",
				Sources: cli.EnvVars(envPrefix + "LOG_LEVEL"),
				Action:  func(_ context.Context, _ *cli.Command, s string) error { return cmd.parseLogLevelFlag(s) },
			},
			&cli.StringFlag{
				Name:    "logFilePath",
				Usage:   "Write logged messages to the provided file, by default logs are written to stderr",
				Sources: cli.EnvVars(envPrefix + "LOG_FILE_PATH"),
				Action:  func(_ context.Context, _ *cli.Command, s string) error { return cmd.parseLogFilePath(s) },
			},
			&cli.StringFlag{
				Name:    "filePatterns",
				Usage:   "Comma separated list of file patterns to process",
				Sources: cli.EnvVars(envPrefix + "FILE_PATTERNS"),
				Action:  func(_ context.Context, _ *cli.Command, s string) error { return cmd.parseFilePatternsFlag(s) },
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "Show version information",
				Action: func(context.Context, *cli.Command) error {
					fmt.Println(version.GetUserAgent())
					return nil
				},
			},
		},
	}

	return cmd
}

func (c *Command) Run(ctx context.Context) error {
	return c.app.Run(ctx, os.Args)
}

func (c *Command) parseLogLevelFlag(s string) error {
	level := new(logging.Level)
	if err := level.UnmarshalText([]byte(s)); err != nil {
		return err
	}
	c.config.LogLevel = *level
	return nil
}

func (c *Command) parseFilePatternsFlag(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	split := strings.Split(s, ",")
	filePatterns := make([]string, 0, len(split))
	for _, p := range split {
		p = filepath.ToSlash(p)
		if ok := doublestar.ValidatePattern(p); !ok {
			return fmt.Errorf("invalid file pattern: %s", p)
		}
		filePatterns = append(filePatterns, p)
	}
	c.config.FilePatterns = filePatterns
	return nil
}

func (c *Command) parseLogFilePath(s string) error {
	s = os.ExpandEnv(s)
	if !strings.HasPrefix(s, "~") {
		c.config.LogFilePath = s
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "failed to determine user home directory")
	}
	s = filepath.Clean(filepath.Join(home, strings.TrimPrefix(s, "~")))
	c.config.LogFilePath = s
	return nil
}
