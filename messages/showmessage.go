package messages

const ShowMessageMethod = "window/showMessage"

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#window_showMessage
type ShowMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}

type MessageType int

const (
	MessageTypeError   MessageType = 1
	MessageTypeWarning MessageType = 2
	MessageTypeInfo    MessageType = 3
	MessageTypeLog     MessageType = 4
)
