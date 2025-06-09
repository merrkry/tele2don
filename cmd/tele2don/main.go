package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/merrkry/tele2don/internal/service"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b, err := service.LoadBridgeService(ctx)
	if err != nil {
		slog.Error("Failed to load bridge service", "err", err)
		os.Exit(1)
	}

	slog.Info("Starting bridge service.")
	go b.Start(ctx)

	<-stop
	slog.Info("Received shutdown signal, stopping the service.")
}
