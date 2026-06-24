package postgres

import (
	"database/sql"
)

type TableName struct {
	schema string
	name   string
}

// Simple returns the table name without schema.
func (t TableName) Simple() string {
	return t.name
}

// Schema returns the schema the table belongs to.
func (t TableName) Schema() string {
	return t.schema
}

// FQ returns the fully-qualified name of the table.
func (t TableName) FQ() string {
	return t.schema + "." + t.name
}

func (t TableName) String() string {
	return t.FQ()
}

type ColumnMeta struct {
	Name      string `db:"column_name"`
	TableName string `db:"table_name"`
	// Position of the column in the table
	Position int `db:"ordinal_position"`
	// Check PostgreSQL and Golang type affinity to know what type to expect here
	DefaultVal any  `db:"column_default"`
	Nullable   Bool `db:"is_nullable"`
	// See UserDefinedTypeName.
	TypeName string `db:"data_type"`
	TypeOID  OID    `db:"type_oid"`
	// When Type is a custom (user-defined) type holds the type name
	UserDefinedTypeName string `db:"udt_name"`
	// Size of VARCHAR and similar types. No populated otherwise
	MaxLen sql.NullInt32 `db:"character_maximum_length"`
	// If data_type identifies a numeric type, this column contains the (declared or implicit) precision of the type
	// for this column. The precision indicates the number of significant digits. It can be expressed in decimal
	// (base 10) or binary (base 2) terms, as specified in the column numeric_precision_radix. For all other data
	// types, this column is null.
	NumericPrecision sql.NullInt32 `db:"numeric_precision"`
	// If data_type identifies an exact numeric type, this column contains the (declared or implicit) scale of the
	// type for this column. The scale indicates the number of significant digits to the right of the decimal point.
	// It can be expressed in decimal (base 10) or binary (base 2) terms, as specified in the column
	// numeric_precision_radix. For all other data types, this column is null.
	NumericScale sql.NullInt32 `db:"numeric_scale"`
	// Set when processing primary keys. Nil if the table has no PK
	// TODO: Not in use at the moment, maybe remove if it turns out to not be useful
	PKConstraint *ConstraintMeta
	// Set when processing TABLE unique indexes. This column and table defined unique constraints as well as
	// explicitly created as well as unique indexes to support PKs.
	// A column is unique (simple or composed constraint) if this list is not empty.
	// TODO: Not in use at the moment, maybe remove if it turns out to not be useful
	UniqueIndexes []UniqueIndexMeta
	// 'CHECK' constraints on this column if any
	// TODO: Not in use that the moment, maybe remove if it turns out to not be useful
	CheckConstraints []*ConstraintMeta
	TypeDef          baseType
}

type ColumnMetaList []*ColumnMeta

func (c ColumnMetaList) Names() []string {
	names := make([]string, len(c))
	for i, col := range c {
		names[i] = col.Name
	}

	return names
}

type ConstraintMeta struct {
	// Constrained table's oid
	OID OID `db:"oid"`
	// Useful for debugging
	Name string `db:"conname"`
	// SQL expression of the constraint, e.g.:
	//   - Check const.: CHECK (c5 > 0::numeric)
	//   - PK const.: PRIMARY KEY (c1)
	//   - Unique const.: UNIQUE (c3, c4)
	//   - FK const.: FOREIGN KEY (c8) REFERENCES child_t1(id)
	ConstraintExpr string `db:"conexpr"`
	// c = check constraint, f = foreign key constraint, n = not-null constraint (domains only), p = primary key
	// constraint, u = unique constraint, t = constraint trigger, x = exclusion constraint
	Type string `db:"contype"`
	// oid of the index supporting this constraint, if it's a unique, primary key, foreign key, or exclusion
	// constraint; else zero. (references pg_class.oid)
	// For FKs this is the PK index in the referenced table.
	IndexOID OID `db:"index_oid"`
	// If a foreign key, the referenced table; else zero. (references pg_class.oid)
	FKTargetTableOID OID `db:"fk_target_table_oid"`
	// If a foreign key, list of the referenced columns. (references pg_attribute.attnum or
	// information_schema.columns.ordinal_position)
	FKReferencedColumns SliceInt `db:"fk_ref_columns"`
	// If a table constraint (including foreign keys, but not constraint triggers), list of the constrained columns.
	// (references pg_attribute.attnum or information_schema.columns.ordinal_position)
	ConstrainedColumns SliceInt `db:"con_columns_oid"`
}

type FKMeta struct {
	Name string
	// Constrained columns
	ConCols []*ColumnMeta
	// Referenced table
	RefTable *TableMeta
	// Referenced columns
	RefCols []*ColumnMeta
}

type UniqueIndexMeta struct {
	Name string `db:"index_name"`
	// Definition is the SQL statement to create this index
	Definition string `db:"index_def"`
	Partial    bool   `db:"is_partial"`
	Unique     bool   `db:"is_unique"`
	// PK is tru eif this unique index is the Primary Key
	PK bool `db:"is_pk"`
	// ConColsPosition is a list of the constrained columns position in the table as returned by Postgres, e.g.: "1 2"
	ConColsPosition string `db:"con_columns"`
	// ConCols is the list of constrained columns
	ConCols []*ColumnMeta
}

type CheckConstraintMeta struct {
	Name string `db:"index_name"`
	// Definition is the SQL statement to create this index
	Definition string `db:"index_def"`
	// ConCols is the list of constrained columns
	ConCols []*ColumnMeta
}

// CTypeMeta is a definition for custom types. For now the only custom type supported is ENUM.
type CTypeMeta struct {
	// OID of the type, not of the enum in 'pg_enum'
	TypeOID OID `db:"type_oid"`
	// Type is b for a base type, c for a composite type (e.g., a table's row type), d for a domain, e for an enum
	// type, p for a pseudo-type, r for a range type, or m for a multi-range type
	Type     string `db:"typtype"`
	TypeName string `db:"typname"`
	// EnumSortOrder is the index of the enum label. Nil for other types
	EnumSortOrder sql.NullFloat64 `db:"enumsortorder"`
	// EnumLabel is null for other types
	EnumLabel sql.NullString `db:"enumlabel"`
}

type TableMeta struct {
	name             TableName
	columns          []*ColumnMeta
	primaryKey       []*ColumnMeta
	foreignKeys      []FKMeta
	uniqueIndexes    []UniqueIndexMeta
	checkConstraints []CheckConstraintMeta
}

// getTablePK searches for a primary key in cons and returns the list of columns that make the primary key if any.
func getTablePK(cols []*ColumnMeta, cons []*ConstraintMeta) ([]*ColumnMeta, error) {
	var con *ConstraintMeta
	for _, c := range cons {
		if c.Type != "p" {
			continue
		}
		con = c
		break
	}
	if con == nil {
		return nil, nil
	}

	pkCols, err := columnsForPos(cols, con.ConstrainedColumns)
	if err != nil {
		return nil, err
	}

	for _, col := range pkCols {
		col.PKConstraint = con
	}

	return pkCols, nil
}

func (t *TableMeta) Name() TableName {
	return t.name
}

func (t *TableMeta) Columns() []*ColumnMeta {
	return t.columns
}

func (t *TableMeta) PrimaryKey() []*ColumnMeta {
	return t.primaryKey
}

func (t *TableMeta) ForeignKeys() []FKMeta {
	return t.foreignKeys
}

func (t *TableMeta) UniqueIndexes() []UniqueIndexMeta {
	return t.uniqueIndexes
}

func (t *TableMeta) CheckConstraints() []CheckConstraintMeta {
	return t.checkConstraints
}

func (t *TableMeta) ColumnByName(name string) *ColumnMeta {
	for _, col := range t.columns {
		if col.Name == name {
			return col
		}
	}

	return nil
}

// setUniqueIndexes sets table's unique indexes and updates it's columns.
// Unique indexes that support the PK are also included.
func (t *TableMeta) setUniqueIndexes(uIndexes []UniqueIndexMeta) error {
	for _, ui := range uIndexes {
		for _, c := range ui.ConCols {
			c.UniqueIndexes = append(c.UniqueIndexes, ui)
		}

		t.uniqueIndexes = append(t.uniqueIndexes, ui)
	}

	return nil
}

// setCheckConstraints updates p tables with their 'CHECK' constraints.
// Table 'information_schema.check_constraints' has the striped check expression if needed later.
func (t *TableMeta) setCheckConstraints(cons []*ConstraintMeta) error {
	for _, con := range cons {
		if con.Type != "c" {
			continue
		}

		conCols, err := columnsForPos(t.columns, con.ConstrainedColumns)
		if err != nil {
			return err
		}

		for _, col := range conCols {
			col.CheckConstraints = append(col.CheckConstraints, con)
		}

		t.checkConstraints = append(t.checkConstraints, CheckConstraintMeta{
			Name:       con.Name,
			Definition: con.ConstraintExpr,
			ConCols:    conCols,
		})
	}

	return nil
}

func (c *ColumnMeta) GenerateVal() *Value {
	if v, ok := nullOrDefVal(c); ok {
		return v
	}

	return c.TypeDef.SubType.GenVal(c)
}

func (c *ColumnMeta) WrapValue(val any) *Value {
	return &Value{value: val, _type: c.TypeDef}
}
