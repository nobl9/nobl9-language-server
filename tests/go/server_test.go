package tests

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-language-server/internal/messages"
	"github.com/nobl9/nobl9-language-server/internal/testutils"
)

// TestCase is a test case that sends a single JSON RPC request
// and expects a single response.
type TestCase struct {
	// Scenario describes the test case.
	Scenario string
	// Request is the JSON RPC request to send.
	Request TestCaseRequest
	// Response is the expected response from the JSON RPC request.
	Response TestCaseResponse
	// ServerRequests are the expected requests from the server, e.g. diagnostics.
	ServerRequests []TestCaseRequest
}

type TestCaseRequest struct {
	// ID is the JSON RPC request id.
	ID uint64
	// Method is the JSON RPC method to call.
	Method string
	// Params are the parameters to pass to the JSON RPC call.
	Params any
}

type TestCaseResponse struct {
	// ID is the expected JSON RPC response id.
	ID uint64
	// Result is the expected JSON RPC response result.
	Result any
	// Error is the expected error from the JSON RPC request.
	Error *jsonrpc2.Error
}

func TestLSP(t *testing.T) {
	// Currently, access to the Nobl9 API is mandatory.
	// This test will fail if the following environment variables are not set.
	// Since we're only operating on Project (which has no object references) we won't be calling the API anyway.
	t.Setenv("NOBL9_LANGUAGE_SERVER_NO_CONFIG_FILE", "true")
	t.Setenv("NOBL9_LANGUAGE_SERVER_CLIENT_ID", "fake-id")
	t.Setenv("NOBL9_LANGUAGE_SERVER_CLIENT_SECRET", "fake-secret")

	ctx, cancel := context.WithCancel(context.Background())

	server := newServerCommand(t, ctx)
	client := newJSONRPCClient(server.IN, server.OUT)

	server.Start(t)

	t.Cleanup(func() {
		cancel()
		server.Stop(t)

		if t.Failed() {
			logFileData, err := os.ReadFile(logFile)
			require.NoError(t, err)
			t.Logf("log file contents:\n%s", logFileData)
		}
	})

	tests := []TestCase{
		{
			Scenario: "initialize connection",
			Request: TestCaseRequest{
				ID:     1,
				Method: messages.InitializeMethod,
				Params: messages.InitializeParams{
					ClientInfo: &messages.ClientInfo{Name: "test"},
				},
			},
			Response: TestCaseResponse{
				ID: 1,
				Result: messages.InitializeResponse{
					Capabilities: messages.ServerCapabilities{
						TextDocumentSync: messages.TextDocumentSyncKindFull,
						CompletionProvider: &messages.CompletionProvider{
							ResolveProvider:   false,
							TriggerCharacters: []string{":"},
						},
						HoverProvider:      true,
						CodeActionProvider: true,
						ExecuteCommandProvider: &messages.ExecuteCommandProvider{
							Commands: []string{"APPLY", "APPLY_DRY_RUN", "DELETE"},
						},
					},
					ServerInfo: messages.ServerInfo{
						Name:    "nobl9-language-server",
						Version: "1.0.0-test",
					},
				},
			},
		},
		{
			Scenario: "initialized",
			Request: TestCaseRequest{
				ID:     2,
				Method: messages.InitializedMethod,
			},
			Response: TestCaseResponse{
				ID: 2,
			},
		},
		{
			Scenario: "unsupported method",
			Request: TestCaseRequest{
				ID:     3,
				Method: "textDocument/unsupported",
			},
			Response: TestCaseResponse{
				ID: 3,
				Error: &jsonrpc2.Error{
					Code:    jsonrpc2.CodeMethodNotFound,
					Message: "method not supported: textDocument/unsupported",
				},
			},
		},
		{
			Scenario: "open file - wrong language",
			Request: TestCaseRequest{
				ID:     4,
				Method: messages.DidOpenMethod,
				Params: messages.DidOpenParams{
					TextDocument: messages.TextDocumentItem{
						URI:        "file:///tmp/test.txt",
						LanguageID: "plaintext",
						Text:       "This is just some text.",
						Version:    1,
					},
				},
			},
			Response: TestCaseResponse{
				ID: 4,
				Error: &jsonrpc2.Error{
					Code:    jsonrpc2.CodeInvalidParams,
					Message: "unsupported language id: plaintext",
				},
			},
		},
		{
			Scenario: "open valid file",
			Request: TestCaseRequest{
				ID:     5,
				Method: messages.DidOpenMethod,
				Params: messages.DidOpenParams{
					TextDocument: messages.TextDocumentItem{
						URI:        getTestFileURI("valid-project.yaml"),
						LanguageID: "yaml",
						Text:       readTestFile(t, "valid-project.yaml"),
						Version:    1,
					},
				},
			},
			Response: TestCaseResponse{
				ID: 5,
			},
			// Empty diagnostics are sent whenever a file changes and there are no issues,
			// this includes opening a new file.
			ServerRequests: []TestCaseRequest{
				{
					Method: messages.PublishDiagnosticsMethod,
					Params: messages.PublishDiagnosticsParams{
						URI:         getTestFileURI("valid-project.yaml"),
						Version:     1,
						Diagnostics: []messages.Diagnostic{},
					},
				},
			},
		},
		{
			Scenario: "hover",
			Request: TestCaseRequest{
				ID:     5,
				Method: messages.HoverMethod,
				Params: messages.HoverParams{
					TextDocumentPositionParams: messages.TextDocumentPositionParams{
						TextDocument: messages.TextDocumentIdentifier{
							URI: getTestFileURI("valid-project.yaml"),
						},
						Position: messages.Position{
							Line:      1,
							Character: 1,
						},
					},
				},
			},
			Response: TestCaseResponse{
				ID: 5,
				Result: messages.HoverResponse{
					Contents: messages.MarkupContent{
						Kind: messages.Markdown,
						Value: "`kind:string`\n\n" +
							"Kind represents all the [Object] kinds available in the API to perform operations on.\n\n" +
							"**Validation rules:**\n\n- should be equal to 'Project'",
					},
				},
			},
		},
		{
			Scenario: "close file",
			Request: TestCaseRequest{
				ID:     7,
				Method: messages.DidCloseMethod,
				Params: messages.DidCloseParams{
					TextDocument: messages.TextDocumentIdentifier{
						URI: getTestFileURI("valid-project.yaml"),
					},
				},
			},
			Response: TestCaseResponse{
				ID: 7,
			},
		},
		{
			Scenario: "hover on a closed file",
			Request: TestCaseRequest{
				ID:     8,
				Method: messages.HoverMethod,
				Params: messages.HoverParams{
					TextDocumentPositionParams: messages.TextDocumentPositionParams{
						TextDocument: messages.TextDocumentIdentifier{
							URI: getTestFileURI("valid-project.yaml"),
						},
						Position: messages.Position{
							Line:      1,
							Character: 1,
						},
					},
				},
			},
			Response: TestCaseResponse{
				ID: 8,
				Error: &jsonrpc2.Error{
					Message: fmt.Sprintf(
						"file not found: file://%s/tests/go/inputs/valid-project.yaml",
						testutils.FindModuleRoot()),
				},
			},
		},
		{
			Scenario: "open invalid file - diagnostics",
			Request: TestCaseRequest{
				ID:     9,
				Method: messages.DidOpenMethod,
				Params: messages.DidOpenParams{
					TextDocument: messages.TextDocumentItem{
						URI:        getTestFileURI("invalid-service.yaml"),
						LanguageID: "yaml",
						Text:       readTestFile(t, "invalid-service.yaml"),
						Version:    1,
					},
				},
			},
			Response: TestCaseResponse{
				ID: 9, // didOpen response which is empty.
			},
			ServerRequests: []TestCaseRequest{
				{
					Method: messages.PublishDiagnosticsMethod,
					Params: messages.PublishDiagnosticsParams{
						URI:     getTestFileURI("invalid-service.yaml"),
						Version: 1,
						Diagnostics: []messages.Diagnostic{
							{
								Message:  "metadata.project: property is required but was empty",
								Severity: messages.DiagnosticSeverityError,
								Source:   ptr("nobl9-language-server"),
								Range: messages.Range{
									Start: messages.Position{
										Line:      2,
										Character: 0,
									},
									End: messages.Position{
										Line:      2,
										Character: 8,
									},
								},
							},
							{
								Message: "string must match regular expression: " +
									"'^[a-z0-9]([-a-z0-9]*[a-z0-9])?$' (e.g. 'my-name', '123-abc');" +
									" an RFC-1123 compliant label name must consist of lower case" +
									" alphanumeric characters or '-', and must start and end with" +
									" an alphanumeric character",
								Severity: messages.DiagnosticSeverityError,
								Source:   ptr("nobl9-language-server"),
								Range: messages.Range{
									Start: messages.Position{
										Line:      3,
										Character: 8,
									},
									End: messages.Position{
										Line:      3,
										Character: 23,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Scenario: "open file still in creation",
			Request: TestCaseRequest{
				ID:     10,
				Method: messages.DidOpenMethod,
				Params: messages.DidOpenParams{
					TextDocument: messages.TextDocumentItem{
						URI:        getTestFileURI("completion.yaml"),
						LanguageID: "yaml",
						Text:       readTestFile(t, "completion.yaml"),
						Version:    1,
					},
				},
			},
			Response: TestCaseResponse{
				ID: 10, // didOpen response which is empty.
			},
			ServerRequests: []TestCaseRequest{
				{
					Method: messages.PublishDiagnosticsMethod,
					Params: messages.PublishDiagnosticsParams{
						URI:     getTestFileURI("completion.yaml"),
						Version: 1,
						Diagnostics: []messages.Diagnostic{
							{
								Message:  "property is required but was empty",
								Severity: messages.DiagnosticSeverityError,
								Source:   ptr("nobl9-language-server"),
								Range: messages.Range{
									Start: messages.Position{
										Line:      7,
										Character: 2,
									},
									End: messages.Position{
										Line:      7,
										Character: 17,
									},
								},
							},
							{
								Message:  fmt.Sprintf("S is not a valid Kind, try [%s]", strings.Join(manifest.KindNames(), ", ")),
								Severity: messages.DiagnosticSeverityError,
								Source:   ptr("nobl9-language-server"),
								Range: messages.Range{
									Start: messages.Position{
										Line:      32,
										Character: 0,
									},
									End: messages.Position{
										Line:      32,
										Character: 0,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Scenario: "complete kind",
			Request: TestCaseRequest{
				ID:     11,
				Method: messages.CompletionMethod,
				Params: messages.CompletionParams{
					TextDocumentPositionParams: messages.TextDocumentPositionParams{
						TextDocument: messages.TextDocumentIdentifier{
							URI: getTestFileURI("completion.yaml"),
						},
						Position: messages.Position{
							Line:      33,
							Character: 6,
						},
					},
					CompletionContext: messages.CompletionContext{
						TriggerKind: messages.TriggerKindInvoked,
					},
				},
			},
			Response: TestCaseResponse{
				ID: 11,
				Result: func() []messages.CompletionItem {
					var items []messages.CompletionItem
					for _, kind := range manifest.KindNames() {
						items = append(items, messages.CompletionItem{
							Label: kind,
							Kind:  messages.ValueCompletion,
						})
					}
					return items
				}(),
			},
		},
		{
			Scenario: "complete SLO budgeting method",
			Request: TestCaseRequest{
				ID:     12,
				Method: messages.CompletionMethod,
				Params: messages.CompletionParams{
					TextDocumentPositionParams: messages.TextDocumentPositionParams{
						TextDocument: messages.TextDocumentIdentifier{
							URI: getTestFileURI("completion.yaml"),
						},
						Position: messages.Position{
							Line:      7,
							Character: 19,
						},
					},
					CompletionContext: messages.CompletionContext{
						TriggerKind: messages.TriggerKindInvoked,
					},
				},
			},
			Response: TestCaseResponse{
				ID: 12,
				Result: []messages.CompletionItem{
					{
						Label: "Occurrences",
						Kind:  messages.ValueCompletion,
					},
					{
						Label: "Timeslices",
						Kind:  messages.ValueCompletion,
					},
				},
			},
		},
		{
			Scenario: "open barebones object",
			Request: TestCaseRequest{
				ID:     5,
				Method: messages.DidOpenMethod,
				Params: messages.DidOpenParams{
					TextDocument: messages.TextDocumentItem{
						URI:        getTestFileURI("barebones-object.yaml"),
						LanguageID: "yaml",
						Text:       readTestFile(t, "barebones-object.yaml"),
						Version:    1,
					},
				},
			},
			Response: TestCaseResponse{
				ID: 5,
			},
			ServerRequests: []TestCaseRequest{
				{
					Method: messages.PublishDiagnosticsMethod,
					Params: messages.PublishDiagnosticsParams{
						URI:     getTestFileURI("barebones-object.yaml"),
						Version: 1,
						Diagnostics: []messages.Diagnostic{
							{
								Message:  "non-map value is specified",
								Severity: messages.DiagnosticSeverityError,
								Source:   ptr("go-yaml"),
								Range: messages.Range{
									Start: messages.Position{Line: 1, Character: 0},
									End:   messages.Position{Line: 1, Character: 1},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Scenario, func(t *testing.T) {
			client.Request(t, test.Request.Method, test.Request.ID, test.Request.Params)

			client.ReadMessages(t, 1+len(test.ServerRequests))

			if test.Response.Error != nil {
				client.AssertError(t, test.Response.ID, test.Response.Error)
			} else {
				client.AssertResult(t, test.Response.ID, test.Response.Result)
			}
			for _, req := range test.ServerRequests {
				client.AssertServerRequest(t, req.Method, req.Params)
			}
		})
	}

	t.Run("check log file for panics", func(t *testing.T) {
		logFileData, err := os.ReadFile(logFile)
		require.NoError(t, err)

		scanner := bufio.NewScanner(bytes.NewReader(logFileData))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "panic") {
				var v any
				_ = json.Unmarshal([]byte(line), &v)
				data, _ := json.MarshalIndent(v, "", "  ")
				if len(data) > 0 {
					line = string(data)
				}
				t.Fatalf("found recovered panic in logs:\n%s", line)
			}
		}
	})
}
