package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/nobl9/nobl9-language-server/internal/connection"
	"github.com/nobl9/nobl9-language-server/internal/logging"
	"github.com/nobl9/nobl9-language-server/internal/server"
	"github.com/nobl9/nobl9-language-server/internal/stdio"

	"github.com/nobl9/nobl9-language-server/internal/mux"
)

var LspVersion = "v1.0.0"

func main() {
	conf := parseConfig()
	logging.Setup(conf)

	conn, err := bootstrap()
	if err != nil {
		slog.Error("failed to create server", slog.Any("error", err))
		os.Exit(1)
	}
	<-conn.DisconnectNotify()

	slog.Info("server shutdown")
}

func bootstrap() (*jsonrpc2.Conn, error) {
	ctx := context.Background()
	span, ctx := logging.StartSpan(ctx, "bootstrap")
	defer span.Finish()

	srv, err := server.New(ctx, LspVersion)
	if err != nil {
		return nil, err
	}
	stream := stdio.New(os.Stdin, os.Stdout)
	handler := mux.New(srv.GetHandlers()).Handle
	return connection.NewJSONRPC2(ctx, stream, handler), nil
}

func parseConfig() logging.Config {
	var c logging.Config
	flag.StringVar(&c.LogFile, "logFile", "nobl9-language-server.log", "Log messages to the provided file")
	flag.TextVar(&c.LogLevel, "logLevel", logging.DefaultLevel(), "Log messages at the provided level")
	flag.Parse()
	return c
}
