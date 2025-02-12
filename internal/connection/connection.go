package connection

import (
	"context"
	"io"

	"github.com/sourcegraph/jsonrpc2"
)

type JSONRPC2Handler func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (result interface{}, err error)

// NewJSONRPC2 creates a new [JSON RPC 2.0] connection.
//
// [JSON RPC 2.0]: https://www.jsonrpc.org/specification
func NewJSONRPC2(ctx context.Context, stream io.ReadWriteCloser, handler JSONRPC2Handler) *jsonrpc2.Conn {
	return jsonrpc2.NewConn(
		ctx,
		jsonrpc2.NewBufferedStream(stream, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.HandlerWithError(handler))
}
