package messages

const CodeActionMethod = "textDocument/codeAction"

type CodeActionParams struct {
	WorkDoneProgressParams
	PartialResultParams

	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
}

type CodeActionResponse struct {
	Title       string         `json:"title"`
	Kind        CodeActionKind `json:"kind"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	IsPreferred *bool          `json:"isPreferred,omitempty"`
	Edit        *WorkspaceEdit `json:"edit,omitempty"`
	Command     *Command       `json:"command"`
}

type CodeActionKind string

const (
	CodeActionEmpty                 CodeActionKind = ""
	CodeActionQuickFix              CodeActionKind = "quickfix"
	CodeActionRefactor              CodeActionKind = "refactor"
	CodeActionRefactorExtract       CodeActionKind = "refactor.extract"
	CodeActionRefactorInline        CodeActionKind = "refactor.inline"
	CodeActionRefactorRewrite       CodeActionKind = "refactor.rewrite"
	CodeActionSource                CodeActionKind = "source"
	CodeActionSourceOrganizeImports CodeActionKind = "source.organizeImports"
)
