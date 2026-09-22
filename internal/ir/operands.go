package ir

import (
	"fmt"
)

// Operand is the unit of operators and can be of different types. Every different type must implement this interface.
type Operand interface {
	Value(row Row) any
	// Cmp compares rows which is a map of table names to row.
	Cmp(rows map[string]Row, v any) int
}

// OperandColumn is an operand that represents a column in a table.
type OperandColumn struct {
	schema string
	table  string
	col    string
}

func NewOperandColumn(schema, table, column string) OperandColumn {
	return OperandColumn{
		schema: schema,
		table:  table,
		col:    column,
	}
}

func (o OperandColumn) Value(row Row) any {
	if row[o.ColumnName()] == nil {
		return nil
	}

	return row[o.ColumnName()].Val()
}

func (o OperandColumn) Schema() string {
	return o.schema
}

func (o OperandColumn) TableName() string {
	return o.table
}

func (o OperandColumn) ColumnName() string {
	return o.col
}

func (o OperandColumn) FQTableName() string {
	if o.schema == "" {
		return o.table
	}
	return o.schema + "." + o.table
}

func (o OperandColumn) Cmp(rows map[string]Row, v any) int {
	leftRow := rows[o.FQTableName()]
	if leftRow == nil {
		panic(fmt.Errorf("no rows for table %s", o.FQTableName()))
	}
	col := leftRow[o.ColumnName()]
	if col != nil {
		switch vT := v.(type) {
		case nil, OperandNil:
			return col.Cmp(col.Val(), nil)
		case OperandColumn:
			rightRow := rows[vT.FQTableName()]
			return col.Cmp(col.Val(), vT.Value(rightRow))
		case OperandConstant:
			return col.Cmp(col.Val(), vT.value)
		case string:
			return col.Cmp(col.Val(), vT)
		default:
			panic("unknown comparison")
		}

	}

	if _, isNil := v.(OperandNil); isNil || v == nil {
		return 0
	}

	return -1
}

// OperandConstant is an operand that represents a constant in a comparison.
type OperandConstant struct {
	// This is a string, even for numbers because it can be extracted from the SQL queries.
	value string
}

func Constant(val string) Operand {
	return OperandConstant{value: val}
}

func (o OperandConstant) Value(Row) any {
	return o.value
}

func (o OperandConstant) Cmp(rows map[string]Row, v any) int {
	switch vT := v.(type) {
	case nil, OperandNil:
		// nil is less then other values
		return 1
	case OperandColumn:
		// if it's a comparison with a column then the column knows how to compare its own type but, invert the result
		return -1 * vT.Cmp(rows, o.value)
	case OperandConstant:
		// NOTE
		// I don't think this can happen in real world as it covers things like '1=1'.
		// If this turns out to be actually needed, make sure it evaluates the same way Postgres would.
		switch {
		case o.value == vT.value:
			return 0
		case o.value > vT.value:
			return 1
		case o.value < vT.value:
			return -1
		}
	default:
		panic("unknown comparison")
	}

	return 0
}

// OperandNil is used to represent the value nil (or NULL in SQL).
type OperandNil struct{}

func (OperandNil) Value(Row) any {
	return nil
}

func (OperandNil) Cmp(_ map[string]Row, v any) int {
	switch v.(type) {
	case nil, OperandNil:
		return 0
	}

	return -1
}
