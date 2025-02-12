package messages

const DidSaveMethod = "textDocument/didSave"

type DidSaveParams struct {
	Text         *string                `json:"text"`
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}
