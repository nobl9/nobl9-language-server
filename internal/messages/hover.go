package messages

const HoverMethod = "textDocument/hover"

type HoverParams struct {
	TextDocumentPositionParams
}

type HoverResponse struct {
	Contents any `json:"contents"`
}
