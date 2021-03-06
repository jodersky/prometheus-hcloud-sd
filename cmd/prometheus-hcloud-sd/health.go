package main

import (
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log/level"
	"github.com/promhippie/prometheus-hcloud-sd/pkg/config"
	"gopkg.in/urfave/cli.v2"
)

// Health provides the sub-command to perform a health check.
func Health(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "health",
		Usage: "Perform health checks",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "web.address",
				Value:       "0.0.0.0:9000",
				Usage:       "Address to bind the metrics server",
				EnvVars:     []string{"PROMETHEUS_HCLOUD_WEB_ADDRESS"},
				Destination: &cfg.Server.Addr,
			},
			&cli.StringFlag{
				Name:    "hcloud.config",
				Value:   "",
				Usage:   "Path to HetznerCloud configuration file",
				EnvVars: []string{"PROMETHEUS_HCLOUD_CONFIG"},
			},
		},
		Action: func(c *cli.Context) error {
			logger := setupLogger(cfg)

			if c.IsSet("hcloud.config") {
				if err := readConfig(c.String("hcloud.config"), cfg); err != nil {
					level.Error(logger).Log(
						"msg", "Failed to read config",
						"err", err,
					)

					return err
				}
			}

			resp, err := http.Get(
				fmt.Sprintf(
					"http://%s/healthz",
					cfg.Server.Addr,
				),
			)

			if err != nil {
				level.Error(logger).Log(
					"msg", "Failed to request health check",
					"err", err,
				)

				return err
			}

			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				level.Error(logger).Log(
					"msg", "Health check seems to be in bad state",
					"err", err,
					"code", resp.StatusCode,
				)

				return err
			}

			return nil
		},
	}
}
