package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/urfave/cli/v3"

	"github.com/cprates/dbseeder/config"
	"github.com/cprates/dbseeder/internal/postgres"
	"github.com/cprates/dbseeder/seeder"
)

func seedCommand() *cli.Command {
	return &cli.Command{
		Name:        "seed",
		Usage:       "seed a PostgreSQL database",
		Description: "Connect with --conn-str.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "cfg",
				Usage:     "path to the seed config file. See the entire DB with default settings is left empty",
				TakesFile: true,
				Required:  false,
			},
			&cli.StringFlag{
				Name:     "conn-str",
				Usage:    "PostgreSQL connection string",
				Sources:  cli.EnvVars("POSTGRES_CONN_STR"),
				Required: true,
				Validator: func(connStr string) error {
					if connStr == "" {
						return errors.New("--conn-str cannot be empty")
					}
					return nil
				},
			},
		},
		Action: seed,
	}
}

func seed(ctx context.Context, cmd *cli.Command) error {
	connStr := cmd.String("conn-str")
	cfg, err := loadSeedCfg(cmd)
	if err != nil {
		return cli.Exit(err, 1)
	}

	db, err := sqlx.Connect(postgres.DriverName, connStr)
	if err != nil {
		return cli.Exit(fmt.Errorf("unable to connect to DB: %w", err), 1)
	}

	fmt.Printf("[%s] Starting...\n", time.Now().Format(time.DateTime))
	start := time.Now()
	if err := seeder.New(cfg, seeder.NewPostgres(db.DB)).Seed(ctx); err != nil {
		return cli.Exit(fmt.Errorf("failed to seed DB: %w", err), 1)
	}

	fmt.Printf("[%s] database seeded in %s\n", time.Now().Format(time.DateTime), time.Since(start))
	return nil
}

func loadSeedCfg(cmd *cli.Command) (config.Seeder, error) {
	if cmd.String("cfg") != "" {
		cfg, err := config.Load(cmd.String("cfg"))
		if err != nil {
			return config.Seeder{}, err
		}

		return cfg, nil
	}

	cfg := config.Seeder{}
	_ = cfg.Validate()

	return cfg, nil
}
