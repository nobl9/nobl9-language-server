package messages

const PublishDiagnosticsMethod = "textDocument/publishDiagnostics"

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Version     int          `json:"version"`
}

type Diagnostic struct {
	Message            string                         `json:"message"`
	Severity           int                            `json:"severity,omitempty"`
	Source             *string                        `json:"source,omitempty"`
	Range              Range                          `json:"range"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
}

type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

const (
	DiagnosticSeverityError       = 1
	DiagnosticSeverityWarning     = 2
	DiagnosticSeverityInformation = 3
	DiagnosticSeverityHint        = 4
)
