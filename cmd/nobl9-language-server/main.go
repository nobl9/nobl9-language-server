package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/nobl9/nobl9-language-server/internal/cli"
	"github.com/nobl9/nobl9-language-server/internal/connection"
	"github.com/nobl9/nobl9-language-server/internal/logging"
	"github.com/nobl9/nobl9-language-server/internal/recovery"
	"github.com/nobl9/nobl9-language-server/internal/server"
	"github.com/nobl9/nobl9-language-server/internal/stdio"
	"github.com/nobl9/nobl9-language-server/internal/version"

	"github.com/nobl9/nobl9-language-server/internal/mux"
)

func main() {
	cmd := cli.New(run)
	if err := cmd.Run(); err != nil {
		slog.Error("server command returned error", slog.Any("error", err))
		log.Fatal(err)
	}
}

func run(config *cli.Config) error {
	logCloser := logging.Setup(logging.Config{
		LogFile:  config.LogFilePath,
		LogLevel: config.LogLevel,
	})
	defer func() { _ = logCloser.Close() }()

	conn, err := bootstrap(config)
	if err != nil {
		slog.Error("failed to create server", slog.Any("error", err))
		return errors.Wrap(err, "failed to create server")
	}
	recovery.Setup(conn)
	defer func() { recovery.LogPanic(context.Background(), recover()) }()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-signals:
		slog.Info("received signal", slog.Any("signal", sig))
	case <-conn.DisconnectNotify():
		slog.Info("connection lost")
	}

	slog.Info("server shutdown")
	return nil
}

func bootstrap(config *cli.Config) (*jsonrpc2.Conn, error) {
	ctx := context.Background()
	span, _ := logging.StartSpan(ctx, "bootstrap")
	defer span.Finish()

	srv, err := server.New(ctx, version.GetVersion(), config.FilePatterns)
	if err != nil {
		return nil, err
	}
	stream := stdio.New(os.Stdin, os.Stdout)
	handler := mux.New(srv.GetHandlers()).Handle
	return connection.NewJSONRPC2(ctx, stream, handler), nil
}
