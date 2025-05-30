package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/merrkry/tele2don/internal/config"
	"github.com/merrkry/tele2don/internal/server"
)

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "err", err)
		os.Exit(1)
	}

	err = server.StartBridge(ctx, cfg)
	if err != nil {
		slog.Error("An error occured", "err", err)
		os.Exit(1)
	}
	
	<-stop
	slog.Info("Received shutdown signal, stopping the server.")
}
