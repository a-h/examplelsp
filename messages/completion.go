package messages

const CompletionRequestMethod = "textDocument/completion"

type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      *CompletionContext     `json:"context"`
}

type TriggerKind int

const (
	TriggerKindInvoked                         TriggerKind = 1
	TriggerKindTriggerCharacter                TriggerKind = 2
	TriggerKindTriggerForIncompleteCompletions TriggerKind = 3
)

type CompletionContext struct {
	TriggerKind      TriggerKind `json:"triggerKind"`
	TriggerCharacter string      `json:"triggerCharacter"`
}

type CompletionResult struct {
	Items []CompletionItem `json:"items"`
}

type CompletionItem struct {
	Label         string             `json:"label"`
	Kind          CompletionItemKind `json:"kind,omitempty"`
	Detail        string             `json:"detail,omitempty"`
	Documentation string             `json:"documentation,omitempty"`
}

type CompletionItemKind int

const (
	CompletionItemKindUndefined CompletionItemKind = iota
	CompletionItemKindText
	CompletionItemKindMethod
	CompletionItemKindFunction
	CompletionItemKindConstructor
	CompletionItemKindField
	CompletionItemKindVariable
	CompletionItemKindClass
	CompletionItemKindInterface
	CompletionItemKindModule
	CompletionItemKindProperty
	CompletionItemKindUnit
	CompletionItemKindValue
	CompletionItemKindEnum
	CompletionItemKindKeyword
	CompletionItemKindSnippet
	CompletionItemKindColor
	CompletionItemKindFile
	CompletionItemKindReference
	CompletionItemKindFolder
	CompletionItemKindEnumMember
	CompletionItemKindConstant
	CompletionItemKindStruct
	CompletionItemKindEvent
	CompletionItemKindOperator
	CompletionItemKindTypeParameter
)
