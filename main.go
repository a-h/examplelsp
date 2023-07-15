package main

import (
	"encoding/json"
	"os"

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
	log := slog.New(slog.NewTextHandler(lf, nil))

	p := protocol.New(log, os.Stdin, os.Stdout)

	p.SetMethodHandler("initialize", func(params json.RawMessage) (result any, err error) {
		var initializeParams messages.InitializeParams
		if err = json.Unmarshal(params, &initializeParams); err != nil {
			return
		}
		log.Info("initialize", slog.Any("params", initializeParams))

		result = messages.InitializeResult{
			Capabilities: messages.ServerCapabilities{},
			ServerInfo: &messages.ServerInfo{
				Name: "examplelsp",
			},
		}
		return
	})

	if err := p.Process(); err != nil {
		log.Error("processing stopped", slog.Any("error", err))
	}
}
