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
	Start Position `json:"start"`
	End   Position `json:"end"`
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

type CompletionItemKind int

const (
	TextCompletion          CompletionItemKind = 1
	MethodCompletion        CompletionItemKind = 2
	FunctionCompletion      CompletionItemKind = 3
	ConstructorCompletion   CompletionItemKind = 4
	FieldCompletion         CompletionItemKind = 5
	VariableCompletion      CompletionItemKind = 6
	ClassCompletion         CompletionItemKind = 7
	InterfaceCompletion     CompletionItemKind = 8
	ModuleCompletion        CompletionItemKind = 9
	PropertyCompletion      CompletionItemKind = 10
	UnitCompletion          CompletionItemKind = 11
	ValueCompletion         CompletionItemKind = 12
	EnumCompletion          CompletionItemKind = 13
	KeywordCompletion       CompletionItemKind = 14
	SnippetCompletion       CompletionItemKind = 15
	ColorCompletion         CompletionItemKind = 16
	FileCompletion          CompletionItemKind = 17
	ReferenceCompletion     CompletionItemKind = 18
	FolderCompletion        CompletionItemKind = 19
	EnumMemberCompletion    CompletionItemKind = 20
	ConstantCompletion      CompletionItemKind = 21
	StructCompletion        CompletionItemKind = 22
	EventCompletion         CompletionItemKind = 23
	OperatorCompletion      CompletionItemKind = 24
	TypeParameterCompletion CompletionItemKind = 25
)

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
