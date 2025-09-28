package main

import (
	"fmt"
	"log/slog"

	"github.com/Jayleonc/service/internal/server"
)

func main() {
	app, err := server.Bootstrap()
	if err != nil {
		slog.Error("failed to bootstrap application", "error", err)
		return
	}

	addr := fmt.Sprintf("%s:%d", app.Config.Server.Host, app.Config.Server.Port)
	if err := app.Engine.Run(addr); err != nil {
		slog.Error("server exited", "error", err)
	}
}
