package main

import (
	"fmt"
	"log/slog"

	application "github.com/Jayleonc/service/internal/app"
)

func main() {
	app, err := application.Bootstrap()
	if err != nil {
		slog.Error("failed to bootstrap application", "error", err)
		return
	}

	addr := fmt.Sprintf("%s:%d", app.Config.Server.Host, app.Config.Server.Port)
	if err := app.Engine.Run(addr); err != nil {
		slog.Error("server exited", "error", err)
	}
}
