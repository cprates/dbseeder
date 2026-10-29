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
		Name:  "seed",
		Usage: "seed a PostgreSQL database",
		Description: "Connect with --dsn, or with --pg-db, --pg-user, and --pg-pwd. " +
			"The two forms cannot be combined.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "cfg",
				Value:     "./seed_cfg.json",
				Usage:     "path to the seed config file",
				TakesFile: true,
				Validator: func(path string) error {
					if path == "" {
						return errors.New("cfg path cannot be empty")
					}
					return nil
				},
			},
		},
		MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
			postgresConnectionFlags(),
		},
		Action: seed,
	}
}

func seed(ctx context.Context, cmd *cli.Command) error {
	dsn, err := connectionDSN(cmd)
	if err != nil {
		return cli.Exit(err, 1)
	}

	cfg, err := config.Load(cmd.String("cfg"))
	if err != nil {
		return cli.Exit(err, 1)
	}

	db, err := sqlx.Connect(postgres.DriverName, dsn)
	if err != nil {
		return cli.Exit(fmt.Errorf("unable to connect to DB: %w", err), 1)
	}

	fmt.Printf("[%s] Starting...\n", time.Now().Format(time.DateTime))
	start := time.Now()
	if err := seeder.New(*cfg, seeder.NewPostgres(db.DB)).Seed(ctx); err != nil {
		return cli.Exit(fmt.Errorf("failed to seed DB: %w", err), 1)
	}

	fmt.Printf("[%s] %d tables seeded in %s\n", time.Now().Format(time.DateTime), len(cfg.Tables), time.Since(start))
	return nil
}
