package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/examplelsp/messages"
	"github.com/a-h/examplelsp/protocol"
	"github.com/aquilax/cooklang-go"
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
	defer func() {
		if r := recover(); r != nil {
			log.Error("panic", slog.Any("recovered", r))
		}
	}()

	p := protocol.New(log, os.Stdin, os.Stdout)

	p.HandleMethod("initialize", func(params json.RawMessage) (result any, err error) {
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

	p.HandleNotification("initialized", func(params json.RawMessage) (err error) {
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

	// Create a queue to process document updates in the order they're received.
	documentUpdates := make(chan messages.TextDocumentItem, 10)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("recovered document update panic", slog.Any("error", r))
			}
		}()
		lineRegexp := regexp.MustCompile(`^line (\d+):`)
		for doc := range documentUpdates {
			diagnostics := []messages.Diagnostic{}

			lineLengths := getLineLengths(doc.Text)
			recipe, err := cooklang.ParseString(doc.Text)
			if err != nil {
				if lineRegexp.MatchString(err.Error()) {
					line, lineNumberErr := strconv.ParseInt(lineRegexp.FindStringSubmatch(err.Error())[1], 10, 64)
					if lineNumberErr != nil {
						log.Error("failed to parse line number from error message", slog.Any("error", lineNumberErr))
						line = 1
					}
					line-- // LSP positions are zero based.
					diagnostics = append(diagnostics, messages.Diagnostic{
						Range: messages.Range{
							Start: messages.Position{
								Line:      int(line),
								Character: 0,
							},
							End: messages.Position{
								Line:      int(line),
								Character: lineLengths[line],
							},
						},
						Severity: ptr(messages.DiagnosticSeverityError),
						Source:   ptr("examplelsp"),
						Message:  strings.SplitN(err.Error(), ":", 2)[1],
					})
				}
			}
			if recipe != nil {
				// Look for silly American measurements.
				lines := strings.Split(doc.Text, "\n")
				for _, step := range recipe.Steps {
					for _, ingredient := range step.Ingredients {
						im := ingredientMarkup(ingredient)

						if ingredient.Amount.Unit == "cup" {
							// Find the position.
							for lineIndex, line := range lines {
								ingredientIndex := strings.Index(line, im)
								if ingredientIndex < 0 {
									continue
								}
								// Find the step line.
								diagnostics = append(diagnostics, messages.Diagnostic{
									Range: messages.Range{
										Start: messages.Position{
											Line:      lineIndex,
											Character: ingredientIndex,
										},
										End: messages.Position{
											Line:      lineIndex,
											Character: ingredientIndex + len(im),
										},
									},
									Severity: ptr(messages.DiagnosticSeverityInformation),
									Source:   ptr("examplelsp"),
									Message:  "Cups are a silly measurement, consider grams",
								})
							}
						}
					}
				}
			}
			swearWordRanges := findSwearWords(doc.Text)
			for _, r := range swearWordRanges {
				diagnostics = append(diagnostics, messages.Diagnostic{
					Range:    r,
					Severity: ptr(messages.DiagnosticSeverityWarning),
					Source:   ptr("examplelsp"),
					Message:  "Mild swearword",
				})
			}
			p.Notify(messages.PublishDiagnosticsMethod, messages.PublishDiagnosticsParams{
				URI:         doc.URI,
				Version:     &doc.Version,
				Diagnostics: diagnostics,
			})
		}
	}()

	p.HandleNotification(messages.DidOpenTextDocumentNotification, func(rawParams json.RawMessage) (err error) {
		log.Info("received didOpenTextDocument notification", slog.Any("params", rawParams))

		var params messages.DidOpenTextDocumentParams
		if err = json.Unmarshal(rawParams, &params); err != nil {
			return
		}
		documentUpdates <- params.TextDocument

		return nil
	})

	p.HandleNotification(messages.DidChangeTextDocumentNotification, func(rawParams json.RawMessage) (err error) {
		log.Info("received didChangeTextDocument notification", slog.Any("params", rawParams))

		var params messages.DidChangeTextDocumentParams
		if err = json.Unmarshal(rawParams, &params); err != nil {
			return
		}

		// In our response to Initializes, we told the client that we need the
		// full content of every document every time - we can't handle partial
		// updates, so there's got to only be one event.
		documentUpdates <- messages.TextDocumentItem{
			URI:     params.TextDocument.URI,
			Version: params.TextDocument.Version,
			Text:    params.ContentChanges[0].Text,
		}

		return nil
	})

	if err := p.Process(); err != nil {
		log.Error("processing stopped", slog.Any("error", err))
	}
}

func ingredientMarkup(ingredient cooklang.Ingredient) string {
	if !strings.Contains(ingredient.Name, " ") && ingredient.Amount.QuantityRaw == "" {
		return fmt.Sprintf("@%s", ingredient.Name)
	}
	unit := ingredient.Amount.Unit
	if unit != "" {
		unit = "%" + unit
	}
	return fmt.Sprintf("@%s{%s%s}", ingredient.Name, ingredient.Amount.QuantityRaw, unit)
}

func getLineLengths(s string) (lengths []int) {
	var c int
	for _, r := range s {
		c++
		if r == '\n' {
			lengths = append(lengths, c)
			c = 0
		}
	}
	return
}

func ptr[T any](v T) *T {
	return &v
}

// https://www.digitalspy.com/tv/a809925/ofcom-swear-words-ranking-in-order-of-offensiveness/
var swearWords = map[string]struct{}{
	"arse":         {},
	"bloody":       {},
	"cow":          {},
	"damn":         {},
	"git":          {},
	"jesus christ": {},
	"minger":       {},
	"sod off":      {},
}

var wordRegexp = regexp.MustCompile(`\w+`)

func findSwearWords(text string) (ranges []messages.Range) {
	for lineIndex, line := range strings.Split(text, "\n") {
		for _, wordPosition := range wordRegexp.FindAllStringIndex(line, -1) {
			word := strings.ToLower(line[wordPosition[0]:wordPosition[1]])
			if _, isSwearword := swearWords[word]; isSwearword {
				ranges = append(ranges, messages.Range{
					Start: messages.Position{
						Line:      lineIndex,
						Character: wordPosition[0],
					},
					End: messages.Position{
						Line:      lineIndex,
						Character: wordPosition[0] + wordPosition[1],
					},
				})
			}
		}
	}
	return ranges
}
