package action

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/promhippie/prometheus-hcloud-sd/pkg/adapter"
	"github.com/promhippie/prometheus-hcloud-sd/pkg/config"
	"github.com/promhippie/prometheus-hcloud-sd/pkg/middleware"
	"github.com/promhippie/prometheus-hcloud-sd/pkg/version"
)

// Server handles the server sub-command.
func Server(cfg *config.Config, logger log.Logger) error {
	level.Info(logger).Log(
		"msg", "Launching Prometheus HetznerCloud SD",
		"version", version.Version,
		"revision", version.Revision,
		"date", version.BuildDate,
		"go", version.GoVersion,
	)

	var gr run.Group

	{
		ctx := context.Background()
		clients := make(map[string]*hcloud.Client, len(cfg.Target.Credentials))

		for _, credential := range cfg.Target.Credentials {
			clients[credential.Project] = hcloud.NewClient(
				hcloud.WithToken(
					credential.Token,
				),
			)
		}

		disc := Discoverer{
			clients:   clients,
			logger:    logger,
			refresh:   cfg.Target.Refresh,
			separator: ",",
			lasts:     make(map[string]struct{}),
		}

		a := adapter.NewAdapter(ctx, cfg.Target.File, "hcloud-sd", disc, logger)
		a.Run()
	}

	{
		server := &http.Server{
			Addr:         cfg.Server.Addr,
			Handler:      handler(cfg, logger),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		gr.Add(func() error {
			level.Info(logger).Log(
				"msg", "Starting metrics server",
				"addr", cfg.Server.Addr,
			)

			return server.ListenAndServe()
		}, func(reason error) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(ctx); err != nil {
				level.Error(logger).Log(
					"msg", "Failed to shutdown metrics gracefully",
					"err", err,
				)

				return
			}

			level.Info(logger).Log(
				"msg", "Metrics shutdown gracefully",
				"reason", reason,
			)
		})
	}

	{
		stop := make(chan os.Signal, 1)

		gr.Add(func() error {
			signal.Notify(stop, os.Interrupt)

			<-stop

			return nil
		}, func(err error) {
			close(stop)
		})
	}

	return gr.Run()
}

func handler(cfg *config.Config, logger log.Logger) *chi.Mux {
	mux := chi.NewRouter()
	mux.Use(middleware.Recoverer(logger))
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Timeout)
	mux.Use(middleware.Cache)

	reg := promhttp.HandlerFor(
		registry,
		promhttp.HandlerOpts{
			ErrorLog: promLogger{logger},
		},
	)

	mux.Route("/", func(root chi.Router) {
		root.Get(cfg.Server.Path, func(w http.ResponseWriter, r *http.Request) {
			reg.ServeHTTP(w, r)
		})

		root.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)

			io.WriteString(w, http.StatusText(http.StatusOK))
		})

		root.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)

			io.WriteString(w, http.StatusText(http.StatusOK))
		})
	})

	return mux
}
