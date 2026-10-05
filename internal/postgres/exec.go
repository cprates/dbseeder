package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

func ExecScript(ctx context.Context, conn *sql.DB, script string) error {
	_, err := conn.ExecContext(ctx, script)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return fmt.Errorf("executing script: %w. %s", err, pgErr.Detail)
		}
		return fmt.Errorf("executing script: %w", err)
	}

	return nil
}
