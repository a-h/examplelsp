import * as vscode from "vscode";

import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient;

export function activate(context: vscode.ExtensionContext) {
  vscode.window.showInformationMessage(`Starting LSP`);
  let serverOptions: ServerOptions = {
	command: "examplelsp",
	transport: TransportKind.stdio,
  };

  let clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "cook" }],
  };

  client = new LanguageClient(
    "cook",
    "cook",
    serverOptions,
    clientOptions
  );

  client.start();
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}