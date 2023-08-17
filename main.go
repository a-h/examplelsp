package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/a-h/examplelsp/lsp"
	"github.com/a-h/examplelsp/messages"
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

	m := lsp.NewMux(log, os.Stdin, os.Stdout)

	fileURIToContents := map[string]string{}

	m.HandleMethod("initialize", func(params json.RawMessage) (result any, err error) {
		var initializeParams messages.InitializeParams
		if err = json.Unmarshal(params, &initializeParams); err != nil {
			return
		}
		log.Info("recevied initialize method", slog.Any("params", initializeParams))

		result = messages.InitializeResult{
			Capabilities: messages.ServerCapabilities{
				TextDocumentSync: messages.TextDocumentSyncKindFull,
				CompletionProvider: &messages.CompletionOptions{
					TriggerCharacters: []string{"%"},
				},
			},
			ServerInfo: &messages.ServerInfo{
				Name: "examplelsp",
			},
		}
		return
	})

	m.HandleNotification("initialized", func(params json.RawMessage) (err error) {
		log.Info("received initialized notification", slog.Any("params", params))
		return nil
	})

	m.HandleMethod(messages.CompletionRequestMethod, func(rawParams json.RawMessage) (result any, err error) {
		log.Info("received completion request", slog.Any("params", rawParams))

		var params messages.CompletionParams
		if err = json.Unmarshal(rawParams, &params); err != nil {
			return
		}

		doc, _ := cooklang.ParseString(fileURIToContents[params.TextDocument.URI])
		var r []messages.CompletionItem
		for _, step := range doc.Steps {
			for _, ingredient := range step.Ingredients {
				if positionIsInRange(ingredient.Range, params.Position) {
					r = append(r, ingredientUnitCompletionItems...)
				}
			}
		}
		return r, nil
	})

	// Create a queue to process document updates in the order they're received.
	documentUpdates := make(chan messages.TextDocumentItem, 10)
	go func() {
		for doc := range documentUpdates {
			fileURIToContents[doc.URI] = doc.Text
			diagnostics := []messages.Diagnostic{}
			diagnostics = append(diagnostics, getRecipeParseErrorDiagnostics(doc.Text)...)
			diagnostics = append(diagnostics, getAmericanMeasurementsDiagnostics(doc.Text)...)
			diagnostics = append(diagnostics, getSwearwordDiagnostics(doc.Text)...)
			m.Notify(messages.PublishDiagnosticsMethod, messages.PublishDiagnosticsParams{
				URI:         doc.URI,
				Version:     &doc.Version,
				Diagnostics: diagnostics,
			})
		}
	}()

	m.HandleNotification(messages.DidOpenTextDocumentNotification, func(rawParams json.RawMessage) (err error) {
		log.Info("received didOpenTextDocument notification")

		var params messages.DidOpenTextDocumentParams
		if err = json.Unmarshal(rawParams, &params); err != nil {
			return
		}
		documentUpdates <- params.TextDocument

		return nil
	})

	m.HandleNotification(messages.DidChangeTextDocumentNotification, func(rawParams json.RawMessage) (err error) {
		log.Info("received didChangeTextDocument notification")

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

	if err := m.Process(); err != nil {
		log.Error("processing stopped", slog.Any("error", err))
	}
}

var ingredientUnitCompletionItems = []messages.CompletionItem{
	{
		Label:         "g",
		Kind:          messages.CompletionItemKindUnit,
		Detail:        "grams",
		Documentation: "Grams are a unit of mass.",
	},
	{
		Label:         "kg",
		Kind:          messages.CompletionItemKindUnit,
		Detail:        "kilograms",
		Documentation: "Kilograms are a unit of mass.",
	},
	{
		Label:         "ml",
		Kind:          messages.CompletionItemKindUnit,
		Detail:        "milliliters",
		Documentation: "Milliliters are a unit of volume.",
	},
}

func positionIsInRange(r cooklang.Range, position messages.Position) bool {
	return position.Line >= r.Start.Line &&
		position.Line <= r.End.Line &&
		position.Character >= r.Start.Character &&
		position.Character <= r.End.Character
}

func getSwearwordDiagnostics(text string) (diagnostics []messages.Diagnostic) {
	swearWordRanges := findSwearWords(text)
	for _, r := range swearWordRanges {
		diagnostics = append(diagnostics, messages.Diagnostic{
			Range:    r,
			Severity: ptr(messages.DiagnosticSeverityWarning),
			Source:   ptr("examplelsp"),
			Message:  "Mild swearword",
		})
	}
	return
}

func getAmericanMeasurementsDiagnostics(text string) (diagnostics []messages.Diagnostic) {
	recipe, err := cooklang.ParseString(text)
	if err != nil {
		return
	}
	lines := strings.Split(text, "\n")
	for _, step := range recipe.Steps {
		for _, ingredient := range step.Ingredients {
			if ingredient.Amount.Unit == "cup" {
				im := ingredientMarkup(ingredient)
				// Find the position.
				for lineIndex, line := range lines {
					ingredientIndex := strings.Index(line, im)
					if ingredientIndex < 0 {
						continue
					}
					// Find the step line.
					diagnostics = append(diagnostics, messages.Diagnostic{
						Range: messages.Range{
							Start: messages.NewPosition(lineIndex, ingredientIndex),
							End:   messages.NewPosition(lineIndex, ingredientIndex+len(im)),
						},
						Severity: ptr(messages.DiagnosticSeverityInformation),
						Source:   ptr("examplelsp"),
						Message:  "Cups are a silly measurement, consider grams",
					})
				}
			}
		}
	}
	return
}

func getRecipeParseErrorDiagnostics(text string) (diagnostics []messages.Diagnostic) {
	_, err := cooklang.ParseString(text)
	if err == nil {
		return
	}
	cerr, isCooklangError := err.(*cooklang.Error)
	if !isCooklangError {
		return
	}
	diagnostics = append(diagnostics, messages.Diagnostic{
		Range: messages.Range{
			Start: messages.NewPosition(cerr.Range.Start.Line, cerr.Range.Start.Character),
			End:   messages.NewPosition(cerr.Range.End.Line, cerr.Range.End.Character),
		},
		Severity: ptr(messages.DiagnosticSeverityError),
		Source:   ptr("examplelsp"),
		Message:  cerr.Message,
	})
	return
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

func getLineLength[T int | int64](s string, lineIndex T) (length int) {
	var l T
	var c int
	for _, r := range s {
		c++
		if r == '\n' {
			if lineIndex == l {
				return c
			}
			c = 0
			l++
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
