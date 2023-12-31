package messages

type InitializeParams struct {
	// Information about the client
	ClientInfo *ClientInfo `json:"clientInfo"`

	// The capabilities provided by the client (editor or tool)
	Capabilities ClientCapabilities `json:"capabilities"`
}

type ClientInfo struct {
	Name    string  `json:"name"`
	Version *string `json:"version"`
}

type ClientCapabilities struct {
}

type InitializeResult struct {
	// The capabilities the language server provides.
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo"`
}

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#serverCapabilities
type ServerCapabilities struct {
	TextDocumentSync   TextDocumentSyncKind `json:"textDocumentSync"`
	CompletionProvider *CompletionOptions   `json:"completionProvider,omitempty"`
}

type TextDocumentSyncKind int

const (
	TextDocumentSyncKindNone TextDocumentSyncKind = iota
	TextDocumentSyncKindFull
	TextDocumentSyncKindIncremental
)

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type ServerInfo struct {
	Name    string  `json:"name"`
	Version *string `json:"version"`
}
