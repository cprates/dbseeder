package seeder

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/cprates/dbseeder/internal/ir"
	"github.com/cprates/dbseeder/internal/postgres"
)

type Postgres struct {
	cache     sync.Map
	db        *sql.DB
	inspector *postgres.Inspector
}

func NewPostgres(db *sql.DB) *Postgres {
	return &Postgres{
		cache:     sync.Map{},
		db:        db,
		inspector: postgres.NewInspector(db),
	}
}

func (p *Postgres) GetTable(ctx context.Context, schema, name string) (Table, error) {
	if td, ok := p.cache.Load(schema + "." + name); ok {
		return td.(Table), nil
	}

	tMeta, err := p.inspector.TableWithSchema(ctx, schema, name)
	if err != nil {
		return Table{}, err
	}

	table, err := p.tableMetaToTable(ctx, tMeta, schema, name)
	if err != nil {
		return Table{}, err
	}

	p.cache.Store(table.FQName(), table)

	return table, nil
}

func (p *Postgres) Parse(ctx context.Context, query string) (ir.Expression, error) {
	return postgres.Parse(ctx, p.inspector.TableWithSchema, query)
}

// // GenSQLScript generates insert statements grouped by table wrapped in a transaction.
func (p *Postgres) GenSQLScript(ctx context.Context, tables []*Table) (string, error) {
	script := strings.Builder{}
	script.WriteString("BEGIN;\n")
	done := make(map[string]struct{})
	for _, table := range tables {
		if len(table.Rows()) == 0 {
			continue
		}

		pgTable, err := p.inspector.TableWithSchema(ctx, table.Schema(), table.Name())
		if err != nil {
			return "", err
		}
		err = p.orderedInserts(&script, pgTable, tables, done)
		if err != nil {
			return "", fmt.Errorf("generating insert statement for %s: %w", table.Name(), err)
		}
	}
	script.WriteString("\nCOMMIT;\n")

	return script.String(), nil
}

// orderedInserts makes sure inserts are in correct order based on dependencies between tables.
func (p *Postgres) orderedInserts(
	script *strings.Builder, tableMeta postgres.TableMeta, data []*Table, done map[string]struct{},
) error {
	fqName := tableMeta.Name().FQ()
	if _, ok := done[fqName]; ok {
		return nil
	}
	done[fqName] = struct{}{}
	table := getTable(fqName, data)
	if table == nil {
		return nil
	}

	for _, fk := range tableMeta.ForeignKeys() {
		err := p.orderedInserts(script, *fk.RefTable, data, done)
		if err != nil {
			return err
		}
	}

	return genInserts(tableMeta, script, table.Rows())
}

func getTable(fqName string, tables []*Table) *Table {
	for _, table := range tables {
		if fqName == table.FQName() {
			return table
		}
	}

	return nil
}

func genInserts(table postgres.TableMeta, script *strings.Builder, rows []Row) error {
	if len(rows) == 0 {
		return nil
	}

	const stmt = "\nINSERT INTO \"%s\".\"%s\" (%s) VALUES\n"
	mustWrite := func(_ int, err error) {
		if err != nil {
			panic(err)
		}
	}
	columns := rows[0].ColumnNames()
	header := strings.Join(columns, ",")
	mustWrite(fmt.Fprintf(script, stmt, table.Name().Schema(), table.Name().Simple(), header))

	for i, row := range rows {
		mustWrite(1, script.WriteByte('('))
		for iCol, colName := range columns {
			v := row[colName]
			col := table.ColumnByName(colName)

			// cover self-ref tables use-case - columns that are self-ref have a value set to nil
			if v == nil {
				col.WrapValue(nil)
			}

			b, err := col.TypeDef.Encode(v.Val())
			if err != nil {
				return fmt.Errorf("unable to encode value for column %s: %w", colName, err)
			}
			mustWrite(script.Write(b))

			if iCol < len(columns)-1 {
				mustWrite(1, script.WriteByte(','))
			}
		}

		if i < len(rows)-1 {
			mustWrite(script.WriteString("),\n"))
		} else {
			mustWrite(script.WriteString(");\n"))
		}
	}

	return nil
}

func (p *Postgres) tableMetaToTable(ctx context.Context, meta postgres.TableMeta, schema, name string) (Table, error) {
	pgCols := meta.Columns()
	columns := make([]Column, len(pgCols))
	for i, cd := range pgCols {
		columns[i] = columnFromMetadata(cd)
	}

	table := Table{
		schema:        schema,
		name:          name,
		columns:       columns,
		columnsGetter: p.getTableColumnsAdapter(ctx),
	}

	rels := make(map[string]*ForeignKey)
	var err error
	for _, fk := range meta.ForeignKeys() {
		var refTable Table
		if fk.RefTable.Name().Simple() == name {
			// for self-refs just reference itself
			refTable = table
		} else {
			refTableName := fk.RefTable.Name()
			refTable, err = p.tableMetaToTable(ctx, *fk.RefTable, refTableName.Schema(), refTableName.Simple())
			if err != nil {
				return Table{}, err
			}
		}
		refTableFQName := refTable.FQName()
		rel, ok := rels[refTableFQName]
		if !ok {
			rel = &ForeignKey{table: refTable}
			rels[refTableFQName] = rel
			table.fks = append(table.fks, rel)
		}
		// expects ConCols and RefCols to be in the correct order
		for i, col := range fk.ConCols {
			rel.refs = append(rel.refs, ForeignKeyRef{
				Col:           columnFromMetadata(col),
				ReferencedCol: columnFromMetadata(fk.RefCols[i]),
			})
		}
	}

	table.constraints = append(table.constraints, NewConNotNull(nullableColumns(columns)))

	for _, con := range meta.UniqueIndexes() {
		cols := make([]Column, len(con.ConCols))
		for i, cd := range con.ConCols {
			cols[i] = columnFromMetadata(cd)
		}
		if con.PK {
			table.constraints = append(
				table.constraints,
				NewConPK(con.Name, columnsFromMetadata(con.ConCols), schema, name),
			)
		} else {
			var cond ir.Expression
			if con.Partial {
				cond, err = p.buildPartialIndexCond(ctx, name, con.Definition)
				if err != nil {
					return Table{}, fmt.Errorf("building expression for partial index %s: %w", con.Name, err)
				}
			}
			table.constraints = append(
				table.constraints,
				NewConUniqueIdx(con.Name, columnsFromMetadata(con.ConCols), schema, name, cond),
			)
		}
	}

	for _, con := range meta.CheckConstraints() {
		checkExpr, err := p.buildCheckCond(ctx, name, con.Definition)
		if err != nil {
			return Table{}, fmt.Errorf("building expression for check condition %s: %w", con.Name, err)
		}
		table.ExtendExpr(checkExpr)
	}

	if fksExpr := table.buildFKsExpr(); fksExpr != nil {
		table.ExtendExpr(fksExpr)
	}

	return table, nil
}

func columnsFromMetadata(cds []*postgres.ColumnMeta) []Column {
	cols := make([]Column, len(cds))
	for i, cd := range cds {
		cols[i] = columnFromMetadata(cd)
	}

	return cols
}

func columnFromMetadata(cd *postgres.ColumnMeta) Column {
	return Column{
		name:     cd.Name,
		nullable: bool(cd.Nullable),
		// TODO: as performance goes, these indirections are far from ideal
		generateVal: func() ColumnVal {
			return cd.GenerateVal()
		},
		wrapValue: func(val any) ColumnVal {
			return cd.WrapValue(val)
		},
	}
}

func (p *Postgres) getTableColumnsAdapter(ctx context.Context) func(schema, name string) []Column {
	return func(schema, name string) []Column {
		tMeta, err := p.inspector.TableWithSchema(ctx, schema, name)
		if err != nil {
			panic(err)
		}

		pgCols := tMeta.Columns()
		columns := make([]Column, len(pgCols))
		for i, cd := range pgCols {
			columns[i] = columnFromMetadata(cd)
		}

		return columns
	}
}

func nullableColumns(columns []Column) []Column {
	nullable := make([]Column, 0)
	for _, col := range columns {
		if col.nullable {
			continue
		}

		nullable = append(nullable, col)
	}

	return nullable
}

func (p *Postgres) buildPartialIndexCond(ctx context.Context, tableName, definition string) (ir.Expression, error) {
	where := "WHERE"
	i := strings.Index(definition, where)
	if i == -1 {
		return nil, fmt.Errorf("no where clause? Definition: %s", definition)
	}
	cond := "SELECT * FROM " + tableName + " WHERE" + definition[i+len(where):]
	expr, err := postgres.Parse(ctx, p.inspector.TableWithSchema, cond)

	return expr, err
}

func (p *Postgres) buildCheckCond(ctx context.Context, tableName, definition string) (ir.Expression, error) {
	where := "CHECK"
	i := strings.Index(definition, where)
	if i == -1 {
		return nil, fmt.Errorf("no check clause? Definition: %s", definition)
	}
	cond := "SELECT * FROM " + tableName + " WHERE" + definition[i+len(where):]
	expr, err := postgres.Parse(ctx, p.inspector.TableWithSchema, cond)

	return expr, err
}

func (p *Postgres) ExecScript(ctx context.Context, script string) error {
	return postgres.ExecScript(ctx, p.db, script)
}
