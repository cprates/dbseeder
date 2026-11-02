package main

import (
	"context"
	"io"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestConnectionString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		args    []string
		want    string
		wantErr string
	}{
		{
			name: "conn-str alone",
			args: []string{"seed", "--conn-str", "postgres://user:secret@db.internal:5432/app?sslmode=require"},
			want: "postgres://user:secret@db.internal:5432/app?sslmode=require",
		},
		{
			name: "pg flags use host and port defaults",
			args: []string{"seed", "--pg-db", "app", "--pg-user", "seed", "--pg-pwd", "secret"},
			want: "postgres://seed:secret@localhost:5432/app",
		},
		{
			name: "explicit host and port",
			args: []string{"seed", "--pg-db", "app", "--pg-user", "seed", "--pg-pwd", "secret", "--pg-addr", "db.internal", "--pg-port", "5433"},
			want: "postgres://seed:secret@db.internal:5433/app",
		},
		{
			name: "password and database name are escaped",
			args: []string{"seed", "--pg-db", "app/db", "--pg-user", "seed", "--pg-pwd", "p@ss word"},
			want: "postgres://seed:p%40ss%20word@localhost:5432/app%2Fdb",
		},
		{
			name:    "conn-str combined with a pg flag",
			args:    []string{"seed", "--conn-str", "postgres://db", "--pg-db", "app"},
			wantErr: "option conn-str cannot be set along with option pg-db",
		},
		{
			name:    "conn-str combined with several pg flags",
			args:    []string{"seed", "--conn-str", "postgres://db", "--pg-user", "seed", "--pg-port", "5432"},
			wantErr: "option conn-str cannot be set along with option pg-user",
		},
		{
			name:    "neither form is set",
			args:    []string{"seed"},
			wantErr: "one of these flags needs to be provided: conn-str, pg-db pg-user pg-pwd pg-addr pg-port",
		},
		{
			name:    "password missing",
			args:    []string{"seed", "--pg-db", "app", "--pg-user", "seed"},
			wantErr: "--pg-pwd is required when --conn-str is not set",
		},
		{
			name:    "empty conn-str",
			args:    []string{"seed", "--conn-str", ""},
			wantErr: "--conn-str cannot be empty",
		},
		{
			name:    "empty host",
			args:    []string{"seed", "--pg-db", "app", "--pg-user", "seed", "--pg-pwd", "secret", "--pg-addr", ""},
			wantErr: "--pg-addr cannot be empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := runConnectionString(t, tc.args)
			if tc.wantErr != "" {
				if gotErr == nil || gotErr.Error() != tc.wantErr {
					t.Fatalf("err=%v, want %q", gotErr, tc.wantErr)
				}
				return
			}
			if gotErr != nil {
				t.Fatal(gotErr)
			}
			if got != tc.want {
				t.Fatalf("conn-str=%q, want %q", got, tc.want)
			}
		})
	}
}

func runConnectionString(t *testing.T, args []string) (string, error) {
	t.Helper()

	var connStr string
	var connStrErr error
	command := &cli.Command{
		Name: "seed",
		MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
			postgresConnectionFlags(),
		},
		Writer:    io.Discard,
		ErrWriter: io.Discard,
		Action: func(_ context.Context, cmd *cli.Command) error {
			connStr, connStrErr = connectionString(cmd)
			return nil
		},
	}
	if err := command.Run(context.Background(), args); err != nil {
		return "", err
	}
	return connStr, connStrErr
}
