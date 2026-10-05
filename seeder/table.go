package seeder

import (
	"github.com/cprates/dbseeder/internal/ir"
)

type ColumnVal interface {
	Val() any
	SetVal(any)
	Cmp(l, r any) int
}

// Row represents a row in a table. Values in row must be json serialisable
type Row map[string]ColumnVal

func (r Row) ColumnNames() []string {
	i := 0
	columns := make([]string, len(r))
	for colName := range r {
		columns[i] = colName
		i++
	}

	return columns
}

type Constraint interface {
	Name() string
	Assert(row Row) bool
}

type ForeignKeyRef struct {
	Col           Column
	ReferencedCol Column
}

type ForeignKey struct {
	table Table
	refs  []ForeignKeyRef
}

type Column struct {
	name        string
	nullable    bool
	generateVal func() ColumnVal
	wrapValue   func(val any) ColumnVal
}

// Table is the storage agnostic representation of a table.
type Table struct {
	cfg     *TableConfig
	schema  string
	name    string
	columns []Column
	fks     []*ForeignKey
	// expression that represents the constraints of the schema: check constraints, unique indexes, foreign keys, etc
	expr ir.Expression
	// List of **stateful** constraints to be used while seeding the table
	constraints []Constraint
	// Generated data to seed the table
	rows          []Row
	columnsGetter func(schema, name string) []Column
}

func (t *Table) Schema() string {
	return t.schema
}

// Name of the table without the schema.
func (t *Table) Name() string {
	return t.name
}

func (t *Table) FQName() string {
	return t.schema + "." + t.name
}

func (t *Table) Rows() []Row {
	return t.rows
}

// ExtendExpr adds expr to the existing table expression with an AND operator.
func (t *Table) ExtendExpr(expr ir.Expression) {
	if t.expr == nil {
		t.expr = expr
	} else {
		t.expr = ir.And(t.expr, expr)
	}
}

func (t *Table) Expression() ir.Expression {
	return t.expr
}

// fulfills columns in row that have not been populated yet.
func (t *Table) fullfilRow(row Row) {
	for _, col := range t.columns {
		// skip if already populated
		if _, ok := row[col.Name()]; ok {
			continue
		}

		col.Populate(row)
	}
}

// returns true along with the name of the constraint if the caller should retry, false and an empty string otherwise.
func (t *Table) applyConstraints(row Row) (bool, string) {
	for _, con := range t.constraints {
		if ok := con.Assert(row); !ok {
			return true, con.Name()
		}
	}

	return false, ""
}

// builds the expression of the table's FKs chained with AND.
func (t *Table) buildFKsExpr() ir.Expression {
	var expr ir.Expression
	for _, fk := range t.fks {
		for _, ref := range fk.refs {
			table := fk.table
			l := ir.NewOperandColumn(t.Schema(), t.Name(), ref.Col.Name())
			r := ir.NewOperandColumn(table.Schema(), table.Name(), ref.ReferencedCol.Name())
			if expr == nil {
				expr = ir.Equal(l, r)
			} else {
				expr = ir.And(expr, ir.Equal(l, r))
			}
		}
	}

	return expr
}

func (c *Column) Name() string {
	return c.name
}

func (c *Column) Populate(row Row) ColumnVal {
	v := c.generateVal()
	row[c.name] = v

	return v
}

func (c *Column) SetValue(row Row, val any) {
	row[c.name] = c.wrapValue(val)
}
