package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/nobl9/nobl9-language-server/internal/connection"
	"github.com/nobl9/nobl9-language-server/internal/logging"
	"github.com/nobl9/nobl9-language-server/internal/recovery"
	"github.com/nobl9/nobl9-language-server/internal/server"
	"github.com/nobl9/nobl9-language-server/internal/stdio"

	"github.com/nobl9/nobl9-language-server/internal/mux"
)

var LspVersion = "v1.0.0"

func main() {
	os.Exit(run())
}

func run() int {
	logConf := parseLoggingConfig()
	logCloser := logging.Setup(logConf)
	defer func() { _ = logCloser.Close() }()

	conn, err := bootstrap()
	if err != nil {
		slog.Error("failed to create server", slog.Any("error", err))
		return 1
	}
	defer func() {
		recovery.LogPanic(context.Background(), conn, recover())
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-signals:
		slog.Info("received signal", slog.Any("signal", sig))
	case <-conn.DisconnectNotify():
		slog.Info("connection lost")
	}

	slog.Info("server shutdown")
	return 0
}

func bootstrap() (*jsonrpc2.Conn, error) {
	ctx := context.Background()
	span, _ := logging.StartSpan(ctx, "bootstrap")
	defer span.Finish()

	srv, err := server.New(ctx, LspVersion)
	if err != nil {
		return nil, err
	}
	stream := stdio.New(os.Stdin, os.Stdout)
	handler := mux.New(srv.GetHandlers()).Handle
	return connection.NewJSONRPC2(ctx, stream, handler), nil
}

func parseLoggingConfig() logging.Config {
	var c logging.Config
	flag.StringVar(&c.LogFile, "logFilePath", "", "Log messages to the provided file")
	flag.TextVar(&c.LogLevel, "logLevel", logging.DefaultLevel(), "Log messages at the provided level")
	flag.Parse()
	return c
}
