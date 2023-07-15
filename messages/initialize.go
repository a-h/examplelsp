package messages

type InitializeParams struct {
	// Information about the client
	ClientInfo *struct {
		Name    string  `json:"name"`
		Version *string `json:"version"`
	} `json:"clientInfo"`

	// The capabilities provided by the client (editor or tool)
	Capabilities ClientCapabilities `json:"capabilities"`
}

type ClientCapabilities struct {
}

type InitializeResult struct {
	// The capabilities the language server provides.
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo"`
}

type ServerCapabilities struct {
}

type ServerInfo struct {
	Name    string  `json:"name"`
	Version *string `json:"version"`
}