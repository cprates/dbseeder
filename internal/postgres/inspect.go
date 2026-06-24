package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
)

type Inspector struct {
	db           *sqlx.DB
	cache        sync.Map
	typeMap      typeMap
	enumsFetched bool
}

const (
	TypeEnum = "enum"
)

var (
	DefaultSchema = "public"
)

// New creates a new Postgres Inspector to inspect db.
func NewInspector(db *sql.DB) *Inspector {
	return &Inspector{
		db:      sqlx.NewDb(db, DriverName),
		cache:   sync.Map{},
		typeMap: newTypeMap(),
	}
}

// Table returns a table metadata for a table with name in the default schema "public".
func (i *Inspector) Table(ctx context.Context, name string) (TableMeta, error) {
	return i.TableWithSchema(ctx, DefaultSchema, name)
}

// TableWithSchema is the same as Table but allows to specify the schema.
func (i *Inspector) TableWithSchema(ctx context.Context, schema, name string) (TableMeta, error) {
	if schema == "" {
		schema = DefaultSchema
	}
	tableName := TableName{schema: schema, name: name}
	if td, ok := i.cache.Load(tableName.FQ()); ok {
		return td.(TableMeta), nil
	}

	err := i.getCustomTypes(ctx, schema)
	if err != nil {
		return TableMeta{}, fmt.Errorf("fetching custom types: %w", err)
	}

	cols, err := i.getColumns(ctx, tableName)
	if err != nil {
		return TableMeta{}, fmt.Errorf("getting columns for %s: %w", tableName, err)
	}
	if len(cols) == 0 {
		return TableMeta{}, fmt.Errorf("table %s not found or has no columns", tableName)
	}

	constraints, err := i.getTableConstraints(ctx, tableName)
	if err != nil {
		return TableMeta{}, err
	}

	tableMeta := TableMeta{
		name:    tableName,
		columns: cols,
	}

	tableMeta.foreignKeys, err = i.getFKMeta(ctx, &tableMeta, schema, constraints, tableMeta.columns)
	if err != nil {
		return TableMeta{}, fmt.Errorf("setting FKs on table %s: %w", tableName, err)
	}

	tableMeta.primaryKey, err = getTablePK(cols, constraints)
	if err != nil {
		return TableMeta{}, fmt.Errorf("setting primary key on table %s: %w", tableName, err)
	}

	// TODO: simplify this to be just getUniqueIndexes like getTablePK if ColumnMeta.UniqueIndexes is removed
	uIndexes, err := i.getUniqueIndexes(ctx, tableMeta.columns, tableName)
	if err != nil {
		return TableMeta{}, fmt.Errorf("getting unique indexes for table %s: %w", tableName, err)
	}
	err = tableMeta.setUniqueIndexes(uIndexes)
	if err != nil {
		return TableMeta{}, fmt.Errorf("setting unique indexes on table %s: %w", tableName, err)
	}

	// TODO: similar as for setUniqueIndexes
	err = tableMeta.setCheckConstraints(constraints)
	if err != nil {
		return TableMeta{}, fmt.Errorf("setting check constraints on table %s: %w", tableName, err)
	}

	i.cache.Store(tableName.FQ(), tableMeta)

	return tableMeta, nil
}

func (i *Inspector) tableMetaByOID(ctx context.Context, schema string, oid OID) (TableMeta, error) {
	name, err := i.tableNameByOID(ctx, oid)
	if err != nil {
		return TableMeta{}, err
	}

	tMeta, err := i.TableWithSchema(ctx, schema, name)
	return tMeta, err
}

// TODO: could be replaced with a lookup on 'attrelid' if stored in table metadata.
func (i *Inspector) tableNameByOID(ctx context.Context, oid OID) (string, error) {
	var name string
	err := i.db.GetContext(ctx, &name, fmt.Sprintf("SELECT '%d'::regclass;", oid))

	return name, err
}

func (i *Inspector) registerEnums(cTypes []*CTypeMeta) {
	groupedLabels := enumLabels(cTypes)
	registered := map[OID]struct{}{}
	for _, t := range cTypes {
		if _, ok := registered[t.TypeOID]; ok || t.Type != "e" {
			continue
		}
		i.typeMap.registerType(
			t.TypeOID,
			baseType{t.TypeOID, typeEnum{Labels: groupedLabels[t.TypeName]}},
		)
		registered[t.TypeOID] = struct{}{}
	}
}

func enumLabels(enums []*CTypeMeta) map[string][]string {
	labels := make(map[string][]string)
	for _, t := range enums {
		labels[t.TypeName] = append(labels[t.TypeName], t.EnumLabel.String)
	}

	return labels
}

func (i *Inspector) getColumns(ctx context.Context, tableName TableName) ([]*ColumnMeta, error) {
	q := `
SELECT
	c.column_name,
	c.table_name,
	c.ordinal_position,
	c.column_default,
	c.is_nullable,
	c.data_type,
	c.udt_name,
	c.character_maximum_length,
	a.atttypid AS type_oid,
	c.numeric_precision,
	c.numeric_scale
FROM
	information_schema.columns c,
	pg_catalog.pg_attribute a
WHERE
	c.table_schema = $1
	AND c.table_name = $2
	AND a.attname = c.column_name
	AND a.attrelid = $2::regclass;`

	cols := make([]*ColumnMeta, 0)
	err := i.db.SelectContext(ctx, &cols, q, tableName.Schema(), tableName.Simple())
	if err != nil {
		return nil, err
	}

	for _, col := range cols {
		if col.TypeName == "USER-DEFINED" {
			col.TypeName = TypeEnum
		}

		var ok bool
		col.TypeDef, ok = i.typeMap.get(col.TypeOID)
		if !ok {
			return nil, fmt.Errorf("unknown type %s(%d) in column %s", col.TypeName, col.TypeOID, col.Name)
		}
	}

	return cols, nil
}

// Does not include unique indexes that are not defined as table constraints, e.g.: indexes explicitly created with
// 'CREATE UNIQUE INDEX ...'.
func (i *Inspector) getTableConstraints(ctx context.Context, tableName TableName) ([]*ConstraintMeta, error) {
	q := `
SELECT
	c.oid,
	con.conname,
	pg_catalog.Pg_get_constraintdef(con.oid, true) conexpr,
	con.contype,
	con.conindid AS index_oid,
	con.confrelid AS fk_target_table_oid,
	con.confkey AS fk_ref_columns,
	con.conkey AS con_columns_oid
FROM
	pg_catalog.pg_class c,
	pg_catalog.pg_constraint con,
	pg_catalog.pg_namespace n
WHERE
	conrelid = c.oid
	AND n.oid = c.relnamespace
	AND n.nspname = $1
	AND c.relname = $2`
	constraints := make([]*ConstraintMeta, 0)
	err := i.db.SelectContext(ctx, &constraints, q, tableName.Schema(), tableName.Simple())
	if err != nil {
		return nil, err
	}

	return constraints, nil
}

func (i *Inspector) getUniqueIndexes(
	ctx context.Context, parentCols []*ColumnMeta, tableName TableName,
) ([]UniqueIndexMeta, error) {
	q := `
SELECT
    i.indexrelid::regclass::text AS index_name,
    i.indisprimary AS is_pk,
    i.indisunique as is_unique,
    i.indpred IS NOT NULL AS is_partial,
    idxs.indexdef AS index_def,
	i.indkey AS con_columns
FROM pg_catalog.pg_index i
JOIN pg_catalog.pg_class c ON c.oid = i.indrelid
JOIN pg_catalog.pg_namespace n ON c.relnamespace = n.oid
JOIN pg_catalog.pg_indexes idxs ON idxs.indexname = i.indexrelid::regclass::text AND idxs.schemaname = n.nspname
WHERE
	n.nspname = $1
	AND c.relname = $2
;`
	indexes := make([]UniqueIndexMeta, 0)
	err := i.db.SelectContext(ctx, &indexes, q, tableName.Schema(), tableName.Simple())
	if err != nil {
		return nil, err
	}

	uIndexes := make([]UniqueIndexMeta, 0)
	for _, index := range indexes {
		if !index.Unique {
			continue
		}

		columnsPosition := stringToColumnPositions(index.ConColsPosition)
		if len(index.ConColsPosition) == 0 {
			return nil, fmt.Errorf("unique index %s constrains has no columns", index.Name)
		}

		colMeta, e := columnsForPos(parentCols, columnsPosition)
		if e != nil {
			return nil, fmt.Errorf("getting unique index constrained columns: %w", e)
		}
		index.ConCols = append(index.ConCols, colMeta...)

		uIndexes = append(uIndexes, index)
	}

	return uIndexes, nil
}

func columnsForPos(columns []*ColumnMeta, pos []int) ([]*ColumnMeta, error) {
	cols := make([]*ColumnMeta, len(pos))
	for i, pos := range pos {
		var c *ColumnMeta
		for _, col := range columns {
			if pos == col.Position {
				c = col
				break
			}
		}

		if c == nil {
			return nil, fmt.Errorf("column with position %d not found", pos)
		}

		cols[i] = c
	}

	return cols, nil
}

// the format returned by postgres in "1 3 4".
func stringToColumnPositions(s string) []int {
	parts := strings.Split(s, " ")

	positions := make([]int, len(parts))
	for i, pos := range parts {
		p, _ := strconv.ParseInt(pos, 10, 16)
		positions[i] = int(p)
	}

	return positions
}

// 'columns' is the list of columns in the constrained table.
func (i *Inspector) getFKMeta(
	ctx context.Context, parent *TableMeta, schema string, cons []*ConstraintMeta, columns []*ColumnMeta,
) ([]FKMeta, error) {
	fkMetas := make([]FKMeta, 0)
	for _, con := range cons {
		if con.Type != "f" {
			continue
		}

		var refTable *TableMeta
		var err error
		if con.Type == "f" && con.FKTargetTableOID != con.OID {
			rt, err := i.tableMetaByOID(ctx, schema, con.FKTargetTableOID)
			if err != nil || rt.name.schema == "" {
				return nil, fmt.Errorf("unable to get table metadata for oid %d: %w", con.FKTargetTableOID, err)
			}
			refTable = &rt
		} else {
			// self refs.
			refTable = parent
		}

		conCols, err := columnsForPos(columns, con.ConstrainedColumns)
		if err != nil {
			return nil, fmt.Errorf("getColumnsByPos in constrained table: %w", err)
		}

		refCols, err := columnsForPos(refTable.columns, con.FKReferencedColumns)
		if err != nil {
			return nil, fmt.Errorf("getColumnsByPos in referenced table %s: %w", refTable.name, err)
		}

		fkMetas = append(fkMetas, FKMeta{
			Name:     con.Name,
			ConCols:  conCols,
			RefTable: refTable,
			RefCols:  refCols,
		})
	}

	return fkMetas, nil
}

// For now the only custom type supported is Enum.
func (i *Inspector) getCustomTypes(ctx context.Context, schema string) error {
	if i.enumsFetched {
		return nil
	}

	q := `
SELECT
t.oid AS type_oid,
t.typname,
t.typtype,
e.enumsortorder,
e.enumlabel
FROM pg_catalog.pg_type t
LEFT JOIN pg_catalog.pg_enum e
	ON e.enumtypid = t.oid
LEFT JOIN pg_catalog.pg_namespace n
	ON n.oid = t.typnamespace
WHERE
	n.nspname = $1
	AND t.typtype = 'e';
`
	types := make([]*CTypeMeta, 0)
	err := i.db.SelectContext(ctx, &types, q, schema)
	if err != nil {
		return err
	}

	i.registerEnums(types)
	i.enumsFetched = true

	return nil
}
