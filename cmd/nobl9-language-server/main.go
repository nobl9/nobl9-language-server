package main

import (
	"context"
	"flag"
	"fmt"
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
	"github.com/nobl9/nobl9-language-server/internal/version"

	"github.com/nobl9/nobl9-language-server/internal/mux"
)

func main() {
	os.Exit(run())
}

func run() int {
	config := parseFlags()

	if len(flag.Args()) > 0 {
		fmt.Fprintln(os.Stderr, "Error: positional arguments are not allowed")
		os.Exit(1)
	}
	if config.ShowVersion {
		fmt.Println(version.GetUserAgent())
		return 0
	}

	logCloser := logging.Setup(logging.Config{
		LogFile:  config.LogFile,
		LogLevel: config.LogLevel,
	})
	defer func() { _ = logCloser.Close() }()

	conn, err := bootstrap()
	if err != nil {
		slog.Error("failed to create server", slog.Any("error", err))
		return 1
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
	return 0
}

func bootstrap() (*jsonrpc2.Conn, error) {
	ctx := context.Background()
	span, _ := logging.StartSpan(ctx, "bootstrap")
	defer span.Finish()

	srv, err := server.New(ctx, version.GetVersion())
	if err != nil {
		return nil, err
	}
	stream := stdio.New(os.Stdin, os.Stdout)
	handler := mux.New(srv.GetHandlers()).Handle
	return connection.NewJSONRPC2(ctx, stream, handler), nil
}

type configuration struct {
	LogFile     string
	LogLevel    logging.Level
	ShowVersion bool
}

func parseFlags() configuration {
	var c configuration
	flag.BoolVar(&c.ShowVersion, "version", false, "Show version information")
	flag.StringVar(&c.LogFile, "logFilePath", "", "Log messages to the provided file")
	flag.TextVar(&c.LogLevel, "logLevel", logging.DefaultLevel(), "Log messages at the provided level")
	flag.Parse()
	return c
}
