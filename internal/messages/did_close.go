package messages

const DidCloseMethod = "textDocument/didClose"

type DidCloseParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}
