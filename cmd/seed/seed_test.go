package main

import (
	"context"
	"io"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestConnStr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		args    []string
		want    string
		wantErr string
	}{
		{
			name: "returns the connection string",
			args: []string{"seed", "--conn-str", "postgres://user:secret@db.internal:5432/app?sslmode=require"},
			want: "postgres://user:secret@db.internal:5432/app?sslmode=require",
		},
		{
			name:    "missing conn-str",
			args:    []string{"seed"},
			wantErr: `Required flag "conn-str" not set`,
		},
		{
			name:    "empty conn-str",
			args:    []string{"seed", "--conn-str", ""},
			wantErr: `invalid value "" for flag -conn-str: --conn-str cannot be empty`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := runConnStr(t, tc.args)
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

func runConnStr(t *testing.T, args []string) (string, error) {
	t.Helper()

	var connStr string
	command := seedCommand()
	command.Writer = io.Discard
	command.ErrWriter = io.Discard
	command.Action = func(_ context.Context, cmd *cli.Command) error {
		connStr = cmd.String("conn-str")
		return nil
	}
	if err := command.Run(context.Background(), args); err != nil {
		return "", err
	}
	return connStr, nil
}
