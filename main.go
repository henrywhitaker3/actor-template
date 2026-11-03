package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"git.plexmox.com/henry/gater/internal/api"
	"git.plexmox.com/henry/gater/internal/bootstrap"
	"git.plexmox.com/henry/gater/internal/meta"
)

var version = "dev"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	app, cancel, err := bootstrap.Bootstrap(ctx, bootstrap.Opts{
		Version:            version,
		RegisterModelFuncs: []bootstrap.RegisterModel{},
		RegisterActorFuncs: []bootstrap.RegisterActor{},
		RegisterAPIRoutes:  []api.RouteRegisterer{},
	})
	if err != nil {
		slog.Error("bootstrap app", "error", err)
		os.Exit(1)
	}
	defer cancel()

	go func() {
		if err := meta.New().Run(ctx); err != nil {
			slog.Error("run meta server", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		<-app.Host.Ready()
		meta.Ready = true
		if err := app.API.Start(ctx); err != nil {
			slog.Error("run api", "error", err)
			os.Exit(1)
		}
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		_ = app.API.Stop(ctx)
	}()

	if err := app.Host.Run(ctx); err != nil {
		slog.Error("run actor runtime", "error", err)
		os.Exit(1)
	}
}
