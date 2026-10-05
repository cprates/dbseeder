package seeder

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/cprates/dbseeder/internal/ir"
	"github.com/cprates/dbseeder/internal/logger"
)

type Database interface {
	GetTable(ctx context.Context, schema, name string) (Table, error)
	GenSQLScript(ctx context.Context, tables []*Table) (string, error)
	Parse(ctx context.Context, query string) (ir.Expression, error)
	ExecScript(ctx context.Context, script string) error
}

// Seeder seeds a given list of tables in a given DB. All tables must be in the same schema.
type Seeder struct {
	logger logger.Logger
	tables []*Table
	cfg    Config
	db     Database
}

func New(cfg Config, db Database) *Seeder {
	var lgr logger.Logger = logger.NullLogger{}
	if cfg.LogLevel > 0 {
		lgr = logger.TextLogger{}
	}
	return &Seeder{
		logger: lgr,
		cfg:    cfg,
		db:     db,
	}
}

func (s *Seeder) Seed(ctx context.Context) error {
	// TODO: build the dependency graphs and seed each one in parallel.
	for _, tableCfg := range s.cfg.Tables {
		if tableCfg.MaxRows == 0 {
			continue
		}
		err := s.seedTable(ctx, tableCfg)
		if err != nil {
			return fmt.Errorf("unable to seed table %s: %w", tableCfg.Name, err)
		}
	}

	sqlScript, err := s.db.GenSQLScript(ctx, s.tables)
	if err != nil {
		return err
	}

	if s.cfg.OutputPath != "" {
		err = outputSQLScript(s.cfg.OutputPath, sqlScript)
		if err != nil {
			return fmt.Errorf("outputting sql script: %w", err)
		}
	}

	if s.cfg.DryRun {
		return nil
	}

	return s.db.ExecScript(ctx, sqlScript)
}

func (s *Seeder) seedTable(ctx context.Context, tableCfg *TableConfig) error {
	fqName := tableCfg.Name
	table, err := s.getTable(ctx, fqName)
	if err != nil {
		return fmt.Errorf("unable to get table %s: %w", fqName, err)
	}

	seedExpr, err := s.buildSeedExpr(ctx, table, nil, map[string]struct{}{})
	if err != nil {
		return fmt.Errorf("unable to build seed expression for table %s: %w", fqName, err)
	}

	for range tableCfg.MaxRows {
		success := false
		for n := range tableCfg.MaxRetries {
			newRows, err := s.populateConstrainedColumns(table, seedExpr)
			if err != nil {
				return fmt.Errorf("seeding table %s: %w", fqName, err)
			}

			retry, conName, err := s.finaliseRows(ctx, newRows)
			if err != nil {
				return fmt.Errorf("fulfilling rows: %w", err)
			}
			if retry {
				s.logger.Log(
					"Retrying (%d) row generation for table %s due to failed constraint %s\n",
					n, table.FQName(), conName,
				)
				continue
			}

			for tableName, row := range newRows {
				table, err = s.getTable(ctx, tableName)
				if err != nil {
					return fmt.Errorf("unable to get table %s: %w", tableName, err)
				}
				table.rows = append(table.rows, row)
			}

			success = true
			break
		}

		if !success {
			s.logger.Log("Row regeneration exceeded for table %s\n", table.FQName())
		}
	}

	return nil
}

func (s *Seeder) getTable(ctx context.Context, fqName string) (*Table, error) {
	for _, table := range s.tables {
		if table.FQName() == fqName {
			return table, nil
		}
	}

	tableCfg := s.cfg.GetTableCfg(fqName)
	if tableCfg == nil {
		return nil, errors.New("not configured")
	}

	table, err := s.db.GetTable(ctx, s.cfg.Schema, tableCfg.SimpleName())
	if err != nil {
		return nil, err
	}
	table.cfg = tableCfg
	s.tables = append(s.tables, &table)

	if tableCfg.SQLFilter != "" {
		expr, err := s.db.Parse(ctx, tableCfg.SQLFilter)
		if err != nil {
			return nil, fmt.Errorf("parsing sql filter: %w", err)
		}
		table.ExtendExpr(expr)
	}

	return &table, nil
}

func (s *Seeder) buildSeedExpr(
	ctx context.Context, table *Table, expr ir.Expression, merged map[string]struct{},
) (ir.Expression, error) {
	if table.cfg.MaxRows == 0 {
		return expr, nil
	}
	if expr == nil && table.Expression() != nil {
		expr = table.Expression()
	} else if table.Expression() != nil {
		expr = ir.And(table.Expression(), expr)
	}
	merged[table.FQName()] = struct{}{}

	var err error
	for _, rel := range table.fks {
		expr, err = s.buildSeedExprForDependency(ctx, rel.table.FQName(), expr, merged)
		if err != nil {
			return nil, fmt.Errorf("building expression for foreign key table %s: %w", rel.table.FQName(), err)
		}
	}

	return expr, nil
}

func (s *Seeder) buildSeedExprForDependency(
	ctx context.Context, relTableName string, expr ir.Expression, merged map[string]struct{},
) (ir.Expression, error) {
	if _, ok := merged[relTableName]; ok {
		return expr, nil
	}

	relTable, err := s.getTable(ctx, relTableName)
	if err != nil {
		return nil, fmt.Errorf("getting table: %w", err)
	}

	expr, err = s.buildSeedExpr(ctx, relTable, expr, merged)
	if err != nil {
		return nil, fmt.Errorf("building seed expression: %w", err)
	}

	return expr, nil
}

func (s *Seeder) populateConstrainedColumns(table *Table, seedExpr ir.Expression) (RowSet, error) {
	rowSet := RowSet{}
	if seedExpr == nil {
		return rowSet, nil
	}

	generator := NewRowGenerator(seedExpr, table.columnsGetter)
	return generator.Gen(), nil
}

// finaliseRows populate unconstrained columns that are not part of the seed expression and checks if the finalised
// rows satisfy the tables constraints like unique indexes.
// Returns true and the name of the failed constraint if requires to retry the generation of rows, false otherwise.
func (s *Seeder) finaliseRows(ctx context.Context, rows RowSet) (bool, string, error) {
	for fqName, row := range rows {
		table, err := s.getTable(ctx, fqName)
		if err != nil {
			return false, "", fmt.Errorf("getting table %q", fqName)
		}
		table.fullfilRow(row)

		retry, conName := table.applyConstraints(row)
		if retry {
			return true, conName, nil
		}
	}

	return false, "", nil
}

func outputSQLScript(output, script string) error {
	if output == "stdout" {
		n, err := os.Stdout.WriteString(script)
		if err != nil {
			return err
		}
		if n != len(script) {
			return fmt.Errorf("expected to write %d bytes but wrote %d", len(script), n)
		}

		return nil
	}

	return os.WriteFile(output, []byte(script), 0666)
}
