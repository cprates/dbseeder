package seeder

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cprates/dbseeder/internal/ir"
)

// conUniqueIdx asserts rows against unique index constraints, this includes explicit, implicit (column defined)
// and partial indexes, excludes primary keys which have their own type.
type conUniqueIdx struct {
	name       string
	schema     string
	tableName  string
	columns    []Column
	cachedKeys map[string]struct{}
	// Condition expression of partial unique indexes. Nil if the index has no condition
	condition ir.Expression
}

// newConUniqueIdx returns a new stateful conUniqueIdx.
func newConUniqueIdx(name string, columns []Column, schema, tableName string, condition ir.Expression) *conUniqueIdx {
	return &conUniqueIdx{
		name:       name,
		schema:     schema,
		tableName:  tableName,
		columns:    columns,
		cachedKeys: map[string]struct{}{},
		condition:  condition,
	}
}

// Assert asserts that row is unique in columns constrained by c.con. Returns true if row is unique, false otherwise.
// It makes a few assumptions and panics if any of them is not true:
// - row always have keys for constrained columns in c
// - values in row are all json serialisable
func (c *conUniqueIdx) Assert(row Row) bool {
	var keyBuilder strings.Builder
	for _, col := range c.columns {
		v, err := json.Marshal(row[col.name].Val())
		if err != nil {
			panic(fmt.Errorf("unique index %s, column %s: %w", c.name, col.name, err))
		}

		// ignore err as strings.Builder never errors (at least at the moment)
		keyBuilder.Write(v)
		keyBuilder.WriteByte('_')
	}
	key := keyBuilder.String()

	_, valExists := c.cachedKeys[key]
	if valExists && c.condition == nil {
		// if the value is duplicate and is not partial just fail the assertion
		return false
	}

	// for this assertion to be successful in case of a partial index, the value must not be exist **or** the value
	// may exist **but** the evaluation of the condition must be false otherwise that's a collision
	conditionOK := true
	if c.condition != nil {
		fqName := c.schema + "." + c.tableName
		r := map[string]ir.Row{
			fqName: seederToParserRow(row),
		}
		conditionOK = c.condition.Eval(r)
	}

	c.cachedKeys[key] = struct{}{}

	// when the value matches the condition then values must be unique otherwise it will result in a collision
	if conditionOK {
		return !valExists
	}

	return true
}

func (c *conUniqueIdx) Name() string {
	return "Unique Index " + c.name
}

func seederToParserRow(row Row) ir.Row {
	newRow := ir.Row{}
	for k, v := range row {
		newRow[k] = v
	}

	return newRow
}

// conPK is the same as a unique index at this point.
type conPK struct {
	conUniqueIdx
}

func (c *conPK) Name() string {
	return "Primary Key " + c.name
}

// newConPK returns a new stateful conPK.
func newConPK(name string, columns []Column, schema, tableName string) *conPK {
	return &conPK{conUniqueIdx{
		name:       name,
		schema:     schema,
		tableName:  tableName,
		columns:    columns,
		cachedKeys: map[string]struct{}{},
	}}
}
