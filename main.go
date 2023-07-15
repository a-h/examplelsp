package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/a-h/examplelsp/messages"
	"github.com/a-h/examplelsp/protocol"
	"golang.org/x/exp/slog"
)

func main() {
	lf, err := os.Create("examplelsp.log")
	if err != nil {
		slog.Error("failed to create log output file", slog.Any("error", err))
		os.Exit(1)
	}
	defer lf.Close()
	log := slog.New(slog.NewJSONHandler(lf, nil))

	uriToContents := map[string]string{}

	p := protocol.New(log, os.Stdin, os.Stdout)

	p.SetMethodHandler("initialize", func(params json.RawMessage) (result any, err error) {
		var initializeParams messages.InitializeParams
		if err = json.Unmarshal(params, &initializeParams); err != nil {
			return
		}
		log.Info("recevied initialize method", slog.Any("params", initializeParams))

		result = messages.InitializeResult{
			Capabilities: messages.ServerCapabilities{
				TextDocumentSync: messages.TextDocumentSyncKindFull,
			},
			ServerInfo: &messages.ServerInfo{
				Name: "examplelsp",
			},
		}
		return
	})

	p.SetNotificationHandler("initialized", func(params json.RawMessage) (err error) {
		log.Info("received initialized notification", slog.Any("params", params))
		// Start the message pusher.
		go func() {
			count := 1
			for {
				time.Sleep(time.Second * 1)
				p.Notify(messages.ShowMessageMethod, messages.ShowMessageParams{
					Type:    messages.MessageTypeInfo,
					Message: fmt.Sprintf("Shown %d messages", count),
				})
				count++
			}
		}()
		return nil
	})

	p.SetNotificationHandler(messages.DidOpenTextDocumentNotification, func(params json.RawMessage) (err error) {
		log.Info("received didOpenTextDocument method", slog.Any("params", params))

		var p messages.DidOpenTextDocumentParams
		if err = json.Unmarshal(params, &p); err != nil {
			return
		}
		// Store the contents.
		uriToContents[p.TextDocument.URI] = p.TextDocument.Text

		return nil
	})

	if err := p.Process(); err != nil {
		log.Error("processing stopped", slog.Any("error", err))
	}
}
