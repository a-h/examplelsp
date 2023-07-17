package messages

const PublishDiagnosticsMethod = "textDocument/publishDiagnostics"

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     *int         `json:"version"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type Diagnostic struct {
	Range    Range               `json:"range"`
	Severity *DiagnosticSeverity `json:"severity"`
	// The diagnostic's code, which might appear in the user interface.
	Code            *string          `json:"code"`
	CodeDescription *CodeDescription `json:"codeDescription"`
	// A human-readable string describing the source of this
	// diagnostic, e.g. 'typescript' or 'super lint'.
	Source             *string                        `json:"source"`
	Message            string                         `json:"message"`
	Tags               []DiagnosticTag                `json:"tags"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation"`
	Data               any                            `json:"any"`
}

type CodeDescription struct {
	HREF string `json:"href"`
}

type DiagnosticSeverity int

const (
	DiagnosticSeverityError       DiagnosticSeverity = 1
	DiagnosticSeverityWarning     DiagnosticSeverity = 2
	DiagnosticSeverityInformation DiagnosticSeverity = 3
	DiagnosticSeverityHint        DiagnosticSeverity = 4
)

type DiagnosticTag int

const (
	DiagnosticTagUnnecessary DiagnosticTag = 1
	DiagnosticTagDeprecated  DiagnosticTag = 2
)

type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}
