package messages

const DidOpenTextDocumentNotification = "textDocument/didOpen"

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}
