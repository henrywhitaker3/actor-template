// Package bootstrap
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/henrywhitaker3/actor-template/internal/api"
	"github.com/henrywhitaker3/actor-template/internal/config"
	"github.com/henrywhitaker3/actor-template/internal/database"
	"github.com/henrywhitaker3/windowframe/v2/http"
	"github.com/henrywhitaker3/windowframe/v2/log"
	"github.com/henrywhitaker3/windowframe/v2/otel"
	"github.com/henrywhitaker3/windowframe/v2/tracing"
	"github.com/italypaleale/francis/components/postgres"
	"github.com/italypaleale/francis/components/sqlite"
	"github.com/italypaleale/francis/host/local"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type App struct {
	Config *config.Config
	Host   *local.Host
	DB     *gorm.DB
	API    *http.HTTP
}

type RegisterModel func(*gorm.DB) error

type RegisterActor func(*App) error

type Opts struct {
	Version            string
	RegisterModelFuncs []RegisterModel
	RegisterActorFuncs []RegisterActor
	RegisterAPIRoutes  []api.RouteRegisterer
}

func Bootstrap(ctx context.Context, opts Opts) (*App, context.CancelFunc, error) {
	conf, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}

	log.Setup(log.SlogLevel(conf.LogLevel))

	shutdown, err := otel.Init(otel.Opts{
		Tracing: tracing.TracingOpts{
			ServiceName: conf.App.Name,
		},
		Version:    opts.Version,
		Registerer: prometheus.DefaultRegisterer,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("init otel: %w", err)
	}

	db, err := database.Open(conf.Database.Driver, string(conf.Database.URL))
	if err != nil {
		return nil, nil, fmt.Errorf("open db connection: %w", err)
	}

	for _, f := range opts.RegisterModelFuncs {
		if err := f(db); err != nil {
			return nil, nil, fmt.Errorf("register gorm model: %w", err)
		}
	}

	host, err := runtime(conf.Database, conf.Runtime)
	if err != nil {
		return nil, nil, fmt.Errorf("init runtime: %w", err)
	}

	app := &App{
		Config: conf,
		Host:   host,
		DB:     db,
	}

	for _, f := range opts.RegisterActorFuncs {
		if err := f(app); err != nil {
			return nil, nil, fmt.Errorf("register actor: %w", err)
		}
	}

	api, err := api.New(api.Opts{
		Routes:  opts.RegisterAPIRoutes,
		Name:    conf.App.Name,
		Version: opts.Version,
		URL:     conf.App.URL,
		Host:    host,
		DB:      db,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("init api: %w", err)
	}
	app.API = api

	return app, shutdown, nil
}

func runtime(db config.Database, conf config.Runtime) (*local.Host, error) {
	hostname, err := hostname(conf)
	if err != nil {
		return nil, err
	}

	var dbOpt local.HostOption
	switch db.Driver {
	case "sqlite":
		dbOpt = local.WithSQLiteProvider(sqlite.SQLiteProviderOptions{
			ConnectionString: string(db.URL),
		})
	case "postgres":
		dbOpt = local.WithPostgresProvider(postgres.PostgresProviderOptions{
			ConnectionString: string(db.URL),
		})
	}

	hostOptions := []local.HostOption{
		local.WithAddress(net.JoinHostPort(hostname, "7157")),
		local.WithRuntimePSKs([]byte(conf.PSK)),
		local.WithLogger(slog.With("component", "runtime")),
		dbOpt,
	}
	if len(conf.ManagementTokens) > 0 {
		mgmtToken := []string{}
		for _, t := range conf.ManagementTokens {
			mgmtToken = append(mgmtToken, string(t))
		}
		hostOptions = append(hostOptions, local.WithManagementAPI(local.ManagementOptions{
			Bind:             ":7401",
			ManagementTokens: mgmtToken,
			AllowedOrigins:   conf.ManagementAllowedOrigins,
		}))
	}

	host, err := local.NewHost(hostOptions...)
	if err != nil {
		return nil, fmt.Errorf("init host: %w", err)
	}
	return host, nil
}

func hostname(conf config.Runtime) (string, error) {
	if conf.Hostname != "" {
		return conf.Hostname, nil
	}
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("get hostname: %w", err)
	}
	return hostname, nil
}
