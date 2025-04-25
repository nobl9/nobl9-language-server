package tests

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-language-server/internal/testutils"
)

const logFile = "test.log"

func newServerCommand(t *testing.T, ctx context.Context) *serverCommand {
	_ = os.Remove(logFile)

	root := testutils.FindModuleRoot()
	cmd := exec.CommandContext(
		ctx,
		"go",
		"run",
		"-ldflags=-X github.com/nobl9/nobl9-language-server/internal/version.BuildVersion=1.0.0-test",
		filepath.Join(root, "cmd", "nobl9-language-server", "main.go"),
		"-logFilePath="+logFile,
		"-logLevel=TRACE",
	)

	inputPipe, err := cmd.StdinPipe()
	require.NoError(t, err)

	outputPipe, err := cmd.StdoutPipe()
	require.NoError(t, err)

	stderr := new(bytes.Buffer)
	cmd.Stderr = io.MultiWriter(os.Stderr, stderr)

	return &serverCommand{
		IN:     inputPipe,
		OUT:    outputPipe,
		cmd:    cmd,
		stderr: stderr,
	}
}

type serverCommand struct {
	IN  io.WriteCloser
	OUT io.ReadCloser

	cmd    *exec.Cmd
	stderr io.Reader
}

func (s *serverCommand) Start(t *testing.T) {
	err := s.cmd.Start()
	require.NoError(t, err)
}

func (s *serverCommand) Stop(t *testing.T) {
	_, err := s.cmd.Process.Wait()
	assert.NoError(t, err)

	out, err := io.ReadAll(s.stderr)
	assert.NoError(t, err)
	assert.Empty(t, string(out))
}

func newJSONRPCClient(writer io.Writer, reader io.Reader) *jsonRPCClient {
	return &jsonRPCClient{
		codec:  jsonrpc2.VSCodeObjectCodec{},
		writer: writer,
		reader: bufio.NewReader(reader),
	}
}

type jsonRPCClient struct {
	codec  jsonrpc2.ObjectCodec
	writer io.Writer
	reader *bufio.Reader

	responses      []*jsonrpc2.Response
	serverRequests []*jsonrpc2.Request
}

func (c *jsonRPCClient) AssertResult(t *testing.T, id uint64, expected any) {
	t.Helper()
	msg := fmt.Sprintf("response for id %d", id)

	var resp *jsonrpc2.Response
	resp, c.responses = shiftSlice(c.responses)
	require.NotNil(t, resp, msg)

	require.Equal(t, id, resp.ID.Num, msg)
	require.Nil(t, resp.Error, msg)
	require.NotNil(t, resp.Result, msg)

	jsonExpectedResp, err := json.Marshal(expected)
	require.NoError(t, err, msg)
	require.JSONEq(t, string(jsonExpectedResp), string(*resp.Result), msg)
}

func (c *jsonRPCClient) AssertError(t *testing.T, id uint64, expected *jsonrpc2.Error) {
	t.Helper()
	msg := fmt.Sprintf("response for id %d", id)

	var resp *jsonrpc2.Response
	resp, c.responses = shiftSlice(c.responses)
	require.NotNil(t, resp, msg)

	require.Equal(t, id, resp.ID.Num, msg)
	require.Nil(t, resp.Result, msg)
	require.NotNil(t, resp.Error, msg)

	require.Equal(t, *expected, *resp.Error, msg)
}

func (c *jsonRPCClient) Request(t *testing.T, method string, id uint64, params any) {
	t.Helper()
	msg := fmt.Sprintf("request for method %s and id %d", method, id)

	jsonParams, err := json.Marshal(params)
	require.NoError(t, err, msg)
	request := jsonrpc2.Request{
		Method: method,
		Params: ptr(json.RawMessage(jsonParams)),
		ID:     jsonrpc2.ID{Num: id},
	}

	err = c.codec.WriteObject(c.writer, request)
	require.NoError(t, err, msg)
}

func (c *jsonRPCClient) AssertServerRequest(t *testing.T, method string, expected any) {
	t.Helper()
	msg := fmt.Sprintf("server request for method %s", method)

	for i, req := range c.serverRequests {
		if req.Method != method {
			continue
		}

		require.NotNil(t, req.Params, msg)
		jsonExpectedReq, err := json.Marshal(expected)
		require.NoError(t, err, msg)
		require.JSONEq(t, string(jsonExpectedReq), string(*req.Params), msg)

		c.serverRequests = slices.Delete(c.serverRequests, i, i+1)
		return
	}
	t.Fatalf("not found: %s", msg)
}

// ReadMessages reads n JSON RPC messages from the stream.
// Since the stream may contain asynchronous requests to the RPC client made by the server
// we need to read from the stream until we find the first response.
// Any requests encountered are added to the client's storage for later use.
func (c *jsonRPCClient) ReadMessages(t *testing.T, n int) {
	t.Helper()

	for range n {
		var iface map[string]interface{}
		err := c.codec.ReadObject(c.reader, &iface)
		require.NoError(t, err)
		data, err := json.Marshal(iface)
		require.NoError(t, err)

		if iface["method"] != nil {
			var req jsonrpc2.Request
			err = json.Unmarshal(data, &req)
			require.NoError(t, err)
			c.serverRequests = append(c.serverRequests, &req)
		} else {
			var resp jsonrpc2.Response
			err = json.Unmarshal(data, &resp)
			require.NoError(t, err)
			c.responses = append(c.responses, &resp)
		}
	}
}

func getTestFileURI(filename string) string {
	return "file://" + filepath.Join(testutils.FindModuleRoot(), "tests", "files", filename)
}

func readTestFile(t *testing.T, filename string) string {
	t.Helper()

	path := filepath.Join(testutils.FindModuleRoot(), "tests", "files", filename)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func ptr[T any](v T) *T { return &v }

// shiftSlice returns the first element of the slice and the rest of the slice.
func shiftSlice[T any](s []T) (firstElement T, rest []T) {
	if len(s) == 0 {
		return *new(T), nil
	}
	return s[0], s[1:]
}
