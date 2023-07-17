package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

	p.SetNotificationHandler(messages.DidOpenTextDocumentNotification, func(rawParams json.RawMessage) (err error) {
		log.Info("received didOpenTextDocument method", slog.Any("params", rawParams))

		var params messages.DidOpenTextDocumentParams
		if err = json.Unmarshal(rawParams, &params); err != nil {
			return
		}
		// Store the contents.
		uriToContents[params.TextDocument.URI] = params.TextDocument.Text

		go func(doc messages.TextDocumentItem) {
			swearWordRanges := findSwearWords(doc.Text)
			diagnostics := make([]messages.Diagnostic, len(swearWordRanges))
			for i, r := range swearWordRanges {
				diagnostics[i] = messages.Diagnostic{
					Range:    r,
					Severity: ptr(messages.DiagnosticSeverityWarning),
					Source:   ptr("examplelsp"),
					Message:  "Mild swearword",
				}
			}
			p.Notify(messages.PublishDiagnosticsMethod, messages.PublishDiagnosticsParams{
				URI:         doc.URI,
				Version:     &doc.Version,
				Diagnostics: diagnostics,
			})
		}(params.TextDocument)

		return nil
	})

	if err := p.Process(); err != nil {
		log.Error("processing stopped", slog.Any("error", err))
	}
}

func ptr[T any](v T) *T {
	return &v
}

// https://www.digitalspy.com/tv/a809925/ofcom-swear-words-ranking-in-order-of-offensiveness/
var swearWords = []string{
	"arse",
	"bloody",
	"cow",
	"damn",
	"git",
	"jesus christ",
	"minger",
	"sod off",
}

func findSwearWords(text string) (ranges []messages.Range) {
	for lineIndex, line := range strings.Split(text, "\n") {
		line := strings.ToLower(line)
		for _, sw := range swearWords {
			if swIndex := strings.Index(line, sw); swIndex >= 0 {
				ranges = append(ranges, messages.Range{
					Start: messages.Position{
						Line:      lineIndex,
						Character: swIndex,
					},
					End: messages.Position{
						Line:      lineIndex,
						Character: swIndex + len(sw),
					},
				})
			}
		}
	}
	return ranges
}
