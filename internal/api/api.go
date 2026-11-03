// Package api
package api

import (
	"fmt"
	"log/slog"

	"github.com/henrywhitaker3/windowframe/v2/http"
	"github.com/italypaleale/francis/host/local"
	"gorm.io/gorm"
)

type RouteRegisterer func(*http.HTTP, Opts) error

type Opts struct {
	Name    string
	Version string
	URL     string
	Host    *local.Host
	DB      *gorm.DB
	Routes  []RouteRegisterer
}

func New(opts Opts) (*http.HTTP, error) {
	h := http.New(http.HTTPOpts{
		Port: 8080,
		Openapi: http.OpenapiOpts{
			Enabled:        true,
			ServiceName:    opts.Name,
			ServiceVersion: opts.Version,
			PublicURL:      opts.URL,
		},
		Logger: slog.With("component", "api"),
	})

	for _, r := range opts.Routes {
		if err := r(h, opts); err != nil {
			return nil, fmt.Errorf("register routes: %w", err)
		}
	}

	return h, nil
}
