package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
)

func postgresConnectionFlags() cli.MutuallyExclusiveFlags {
	return cli.MutuallyExclusiveFlags{
		Required: true,
		Flags: [][]cli.Flag{
			{
				&cli.StringFlag{
					Name:    "dsn",
					Usage:   "PostgreSQL connection string. Cannot be combined with the --pg-* flags",
					Sources: cli.EnvVars("POSTGRES_DSN"),
				},
			},
			{
				&cli.StringFlag{
					Name:    "pg-db",
					Usage:   "database name. Required when --dsn is not set",
					Sources: cli.EnvVars("POSTGRES_DB"),
				},
				&cli.StringFlag{
					Name:    "pg-user",
					Usage:   "database user. Required when --dsn is not set",
					Sources: cli.EnvVars("POSTGRES_USER"),
				},
				&cli.StringFlag{
					Name:    "pg-pwd",
					Usage:   "database password. Required when --dsn is not set",
					Sources: cli.EnvVars("POSTGRES_PASSWORD"),
				},
				&cli.StringFlag{
					Name:    "pg-addr",
					Value:   "localhost",
					Usage:   "database host",
					Sources: cli.EnvVars("POSTGRES_HOST"),
				},
				&cli.IntFlag{
					Name:    "pg-port",
					Value:   5432,
					Usage:   "database port",
					Sources: cli.EnvVars("POSTGRES_PORT"),
				},
			},
		},
	}
}

func connectionDSN(cmd *cli.Command) (string, error) {
	if cmd.IsSet("dsn") {
		if cmd.String("dsn") == "" {
			return "", errors.New("--dsn cannot be empty")
		}
		return cmd.String("dsn"), nil
	}

	missing := missingValues(cmd, "pg-db", "pg-user", "pg-pwd")
	if len(missing) > 0 {
		verb := "is"
		if len(missing) > 1 {
			verb = "are"
		}
		return "", fmt.Errorf("%s %s required when --dsn is not set", joinWithAnd(missing), verb)
	}

	addr := cmd.String("pg-addr")
	if addr == "" {
		return "", errors.New("--pg-addr cannot be empty")
	}
	port := cmd.Int("pg-port")
	if port < 1 || port > 65535 {
		return "", errors.New("--pg-port must be between 1 and 65535")
	}

	return postgresDSN(cmd.String("pg-user"), cmd.String("pg-pwd"), addr, port, cmd.String("pg-db")), nil
}

func postgresDSN(user, password, addr string, port int, dbName string) string {
	u := &url.URL{
		Scheme:  "postgres",
		User:    url.UserPassword(user, password),
		Host:    net.JoinHostPort(addr, strconv.Itoa(port)),
		Path:    "/" + dbName,
		RawPath: "/" + url.PathEscape(dbName),
	}
	return u.String()
}

func missingValues(cmd *cli.Command, names ...string) []string {
	var missing []string
	for _, name := range names {
		if cmd.String(name) == "" {
			missing = append(missing, "--"+name)
		}
	}
	return missing
}

func joinWithAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}
