package messages

const InitializeMethod = "initialize"

type InitializeParams struct {
	// Information about the client
	ClientInfo *ClientInfo `json:"clientInfo"`
	RootURI    string      `json:"rootUri,omitempty"`
	ProcessID  int         `json:"processId,omitempty"`
	// The capabilities provided by the client (editor or tool)
	Capabilities          ClientCapabilities `json:"capabilities"`
	InitializationOptions *InitializeOptions `json:"initializationOptions,omitempty"`
	Trace                 string             `json:"trace,omitempty"`
}

type ClientInfo struct {
	Name    string  `json:"name"`
	Version *string `json:"version"`
}

type ClientCapabilities any

type InitializeOptions struct {
	DocumentFormatting bool `json:"documentFormatting"`
	RangeFormatting    bool `json:"documentRangeFormatting"`
	Hover              bool `json:"hover"`
	DocumentSymbol     bool `json:"documentSymbol"`
	CodeAction         bool `json:"codeAction"`
	Completion         bool `json:"completion"`
}
type InitializeResponse struct {
	// The capabilities the language server provides.
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   ServerInfo         `json:"serverInfo"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerCapabilities struct {
	TextDocumentSync           TextDocumentSyncKind         `json:"textDocumentSync,omitempty"`
	DocumentSymbolProvider     bool                         `json:"documentSymbolProvider,omitempty"`
	CompletionProvider         *CompletionProvider          `json:"completionProvider,omitempty"`
	DefinitionProvider         bool                         `json:"definitionProvider,omitempty"`
	DocumentFormattingProvider bool                         `json:"documentFormattingProvider,omitempty"`
	RangeFormattingProvider    bool                         `json:"documentRangeFormattingProvider,omitempty"`
	ExecuteCommandProvider     *ExecuteCommandProvider      `json:"executeCommandProvider"`
	HoverProvider              bool                         `json:"hoverProvider,omitempty"`
	CodeActionProvider         bool                         `json:"codeActionProvider,omitempty"`
	Workspace                  *ServerCapabilitiesWorkspace `json:"workspace,omitempty"`
}

type ExecuteCommandProvider struct {
	Commands []string `json:"commands"`
}

type CompletionProvider struct {
	ResolveProvider   bool     `json:"resolveProvider"`
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type ServerCapabilitiesWorkspace struct {
	WorkspaceFolders WorkspaceFoldersServerCapabilities `json:"workspaceFolders"`
}

type WorkspaceFoldersServerCapabilities struct {
	Supported           bool `json:"supported"`
	ChangeNotifications bool `json:"changeNotifications"`
}

type TextDocumentSyncKind int

const (
	TextDocumentSyncKindNone TextDocumentSyncKind = iota
	TextDocumentSyncKindFull
	TextDocumentSyncKindIncremental
)
