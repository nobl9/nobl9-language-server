package messages

const (
	InitializedMethod = "initialized"
	ShutdownMethod    = "shutdown"
)

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type Range struct {
	// Zero-based inclusive start position.
	Start Position `json:"start"`
	// Zero-based exclusive end.
	End Position `json:"end"`
}

func (r Range) IsZero() bool {
	return r.Start.IsZero() && r.End.IsZero()
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

func (p Position) IsZero() bool { return p.Line == 0 && p.Character == 0 }

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type SymbolInformation struct {
	Name          string   `json:"name"`
	Kind          int64    `json:"kind"`
	Deprecated    bool     `json:"deprecated"`
	Location      Location `json:"location"`
	ContainerName *string  `json:"containerName"`
}

type Command struct {
	Title     string `json:"title" yaml:"title"`
	Command   string `json:"command" yaml:"command"`
	Arguments []any  `json:"arguments,omitempty" yaml:"arguments,omitempty"`
	OS        string `json:"-" yaml:"os,omitempty"`
}

type WorkspaceEdit struct {
	Changes         any `json:"changes"`
	DocumentChanges any `json:"documentChanges"`
}

type MarkedString struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

type MarkupKind string

const (
	PlainText MarkupKind = "plaintext"
	Markdown  MarkupKind = "markdown"
)

type MarkupContent struct {
	Kind  MarkupKind `json:"kind"`
	Value string     `json:"value"`
}

type WorkDoneProgressParams struct {
	WorkDoneToken any `json:"workDoneToken"`
}

type PartialResultParams struct {
	PartialResultToken any `json:"partialResultToken"`
}

type NotificationMessage struct {
	Method string `json:"message"`
	Params any    `json:"params"`
}

type DocumentDefinitionParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	PartialResultParams
}

type DidChangeWorkspaceFoldersParams struct {
	Event WorkspaceFoldersChangeEvent `json:"event"`
}

type WorkspaceFoldersChangeEvent struct {
	Added   []WorkspaceFolder `json:"added,omitempty"`
	Removed []WorkspaceFolder `json:"removed,omitempty"`
}

type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}
