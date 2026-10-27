package seeder

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/cprates/dbseeder/internal/ir"
	"github.com/cprates/dbseeder/internal/logger"
)

type RowSet map[string]Row

// GetForTable name parameter must be the FQ name of the table
func (r RowSet) GetForTable(name string) Row {
	if row, ok := r[name]; ok {
		return row
	}
	r[name] = Row{}

	return r[name]
}

type RowGenerator struct {
	lgr logger.Logger
	// expression that constraints the generated values. This expressions a BET generated based on an SQL query.
	expr ir.Expression
	// map of table name to list of columns. The map should have the name in the format 'schema.table'.
	columnsFetcher func(schema, tableName string) []Column
	rows           RowSet

	// constants assigned to columns
	stepsConstants []Step
	// columns assigned to columns
	stepsColumns []Step
	// Map of FQ column name to domains of columns constrained by range constraints with constants, e.g.: 'c1 > 42'
	domains map[string]*StepDomain
}

func NewRowGenerator(
	lgr logger.Logger,
	expr ir.Expression,
	columnsFetcher func(schema, tableName string) []Column, // TODO: can I get rid of this shit?
) *RowGenerator {
	valueConst := &RowGenerator{
		lgr:            lgr,
		expr:           expr,
		columnsFetcher: columnsFetcher,
		rows:           RowSet{},
		domains:        map[string]*StepDomain{},
	}

	return valueConst
}

func (r *RowGenerator) Gen() RowSet {
	r.reset()
	r.procExpr(r.expr)
	// domains go last so that columns assigned by equalities, which must also satisfy the domain, take precedence
	for _, step := range r.domains {
		r.stepsConstants = append(r.stepsConstants, step)
	}
	r.procSteps()

	return r.rows
}

func (r *RowGenerator) reset() {
	r.rows = RowSet{}
	r.stepsConstants = nil
	r.stepsColumns = nil
	clear(r.domains)
}

func (r *RowGenerator) procExpr(expr any) {
	switch exprT := expr.(type) {
	case ir.OpAnd:
		r.procExpr(exprT.Left())
		r.procExpr(exprT.Right())
	case ir.OpOr:
		r.procOr(exprT)
	case ir.OpEqual:
		r.procEq(exprT)
	case ir.OpNotEqual:
		r.procNotEq(exprT)
	case ir.OpNot:
		r.procNot(exprT.Expression())
	case ir.OpIn:
		r.procIn(exprT)
	case ir.OpGT, ir.OpGE, ir.OpLT, ir.OpLE:
		r.procRange(exprT.(ir.Expression))
	case ir.OpNoOp:
	default:
		panic(fmt.Errorf("unknown type %T", expr))
	}
}

func (r *RowGenerator) procOr(expr ir.OpOr) {
	randGen := rand.New(rand.NewSource(time.Now().UnixNano()))
	// false for Left, true for Right
	if randGen.Intn(2) == 0 {
		r.procExpr(expr.Left())
	} else {
		r.procExpr(expr.Right())
	}
}

func (r *RowGenerator) procEq(expr ir.OpEqual) {
	leftCol, leftIsColumn := isOperandColumn(expr.Left())
	rightCol, rightIsColumn := isOperandColumn(expr.Right())
	if leftIsColumn && rightIsColumn {
		leftColumns := r.columnsFetcher(leftCol.Schema(), leftCol.TableName())
		if leftColumns == nil {
			panic(fmt.Errorf("somehow we do not have columns for table %s", leftCol.TableName()))
		}
		rightColumns := r.columnsFetcher(rightCol.Schema(), rightCol.TableName())
		if rightColumns == nil {
			panic(fmt.Errorf("somehow we do not have columns for table %s", rightCol.TableName()))
		}
		r.stepsColumns = append(r.stepsColumns, &StepEqualColumn{
			leftColumns:  leftColumns,
			left:         expr.Left().(ir.OperandColumn),
			rightColumns: rightColumns,
			right:        expr.Right().(ir.OperandColumn),
		})

		return
	}

	if !leftIsColumn && !rightIsColumn {
		// There's nothing to be done here. This would be something like 1!=1
		return
	}

	if leftIsColumn {
		columns := r.columnsFetcher(leftCol.Schema(), leftCol.TableName())
		if columns == nil {
			panic(fmt.Errorf("somehow we do not have columns for table %s", leftCol.TableName()))
		}
		r.stepsConstants = append(r.stepsConstants, &StepEqualConstant{
			columnMeta: columnByName(columns, leftCol.ColumnName()),
			column:     expr.Left().(ir.OperandColumn),
			operand:    expr.Right().(ir.Operand),
		})

		return
	}

	if rightIsColumn {
		columns := r.columnsFetcher(rightCol.Schema(), rightCol.TableName())
		if columns == nil {
			panic(fmt.Errorf("somehow we do not have columns for table %s", rightCol.TableName()))
		}
		r.stepsConstants = append(r.stepsConstants, &StepEqualConstant{
			columnMeta: columnByName(columns, rightCol.ColumnName()),
			column:     expr.Right().(ir.OperandColumn),
			operand:    expr.Left().(ir.Operand),
		})

		return
	}
}

func (r *RowGenerator) procNotEq(expr ir.OpNotEqual) {
	leftCol, leftIsColumn := isOperandColumn(expr.Left())
	rightCol, rightIsColumn := isOperandColumn(expr.Right())
	if leftIsColumn && rightIsColumn {
		leftColumns := r.columnsFetcher(leftCol.Schema(), leftCol.TableName())
		if leftColumns == nil {
			panic(fmt.Errorf("somehow we do not have columns for table %s", leftCol.TableName()))
		}
		rightColumns := r.columnsFetcher(rightCol.Schema(), rightCol.TableName())
		if rightColumns == nil {
			panic(fmt.Errorf("somehow we do not have columns for table %s", rightCol.TableName()))
		}
		r.stepsColumns = append(r.stepsColumns, &StepNotEqualColumn{
			leftColumns:  leftColumns,
			left:         expr.Left().(ir.OperandColumn),
			rightColumns: rightColumns,
			right:        expr.Right().(ir.OperandColumn),
		})

		return
	}

	if !leftIsColumn && !rightIsColumn {
		// There's nothing to be done here. This would be something like 1=1
		return
	}

	if leftIsColumn {
		if r.excludeFromDomain(leftCol, expr.Right()) {
			// if excluded from the domain, there's no steps to add
			return
		}
		columns := r.columnsFetcher(leftCol.Schema(), leftCol.TableName())
		if columns == nil {
			panic(fmt.Errorf("somehow we do not have columns for table %s", leftCol.TableName()))
		}
		r.stepsConstants = append(r.stepsConstants, &StepNotEqualConstant{
			columnMeta: columnByName(columns, leftCol.ColumnName()),
			column:     expr.Left().(ir.OperandColumn),
			operand:    expr.Right().(ir.Operand),
		})

		return
	}

	if rightIsColumn {
		if r.excludeFromDomain(rightCol, expr.Left()) {
			// if excluded from the domain, there's no steps to add
			return
		}
		columns := r.columnsFetcher(rightCol.Schema(), rightCol.TableName())
		if columns == nil {
			panic(fmt.Errorf("somehow we do not have columns for table %s", rightCol.TableName()))
		}
		r.stepsConstants = append(r.stepsConstants, &StepNotEqualConstant{
			columnMeta: columnByName(columns, rightCol.ColumnName()),
			column:     expr.Right().(ir.OperandColumn),
			operand:    expr.Left().(ir.Operand),
		})

		return
	}
}

// procRange handles the comparisons '>', '>=', '<' and '<=' between a column and a constant by narrowing the domain
// of the column. Comparisons between columns and comparisons on types that do not support domains are ignored.
func (r *RowGenerator) procRange(expr ir.Expression) {
	var above, inclusive bool
	switch expr.(type) {
	case ir.OpGT:
		above = true
	case ir.OpGE:
		above, inclusive = true, true
	case ir.OpLE:
		inclusive = true
	case ir.OpLT:
	}

	col, isColumn := isOperandColumn(expr.Left())
	operand := expr.Right()
	if !isColumn {
		// e.g.: '42 < c1' is the same as 'c1 > 42'
		col, isColumn = isOperandColumn(expr.Right())
		operand = expr.Left()
		above = !above
	}
	if !isColumn {
		// constant comparison, ignore it
		r.lgr.Log("comparison between constants ignored, left: %+v, right: %+v", expr.Left(), expr.Right())
		return
	}

	var constant string
	switch op := operand.(type) {
	case ir.OperandColumn, *ir.OperandColumn:
		panic("range operators not supported between columns")
	case ir.OperandNil, *ir.OperandNil:
		panic("range operators not supported with NULL")
	case ir.OperandConstant:
		constant = op.Value(nil).(string)
	}

	domain := r.domainFor(col)
	if domain == nil {
		return
	}
	if above {
		domain.Above(constant, inclusive)
		return
	}
	domain.Below(constant, inclusive)

}

// excludeFromDomain removes the constant operand from the domain of col. Returns false if the type of col does not
// support domains, true otherwise.
func (r *RowGenerator) excludeFromDomain(col ir.OperandColumn, operand any) bool {
	domain := r.domainFor(col)
	if domain == nil {
		return false
	}
	// domains never generate NULL so, there's nothing to exclude for 'c1 IS NOT NULL'
	if constant, isConstant := operand.(ir.Operand).Value(nil).(string); isConstant {
		domain.Exclude(constant)
	}

	return true
}

// domainFor returns the domain of col creating it if needed. Returns nil if the type of col does not support domains.
func (r *RowGenerator) domainFor(col ir.OperandColumn) ValueDomain {
	key := col.FQTableName() + "." + col.ColumnName()
	if step, ok := r.domains[key]; ok {
		return step.domain
	}

	columns := r.columnsFetcher(col.Schema(), col.TableName())
	if columns == nil {
		panic(fmt.Errorf("somehow could not find columns for table %s.%s", col.Schema(), col.TableName()))
	}
	columnMeta := columnByName(columns, col.ColumnName())
	if columnMeta.newDomain == nil {
		return nil
	}

	step := &StepDomain{
		columnMeta: columnMeta,
		column:     col,
		domain:     columnMeta.newDomain(),
	}
	r.domains[key] = step

	return step.domain
}

func (r *RowGenerator) procIn(expr ir.OpIn) {
	randGen := rand.New(rand.NewSource(time.Now().UnixNano()))

	column := expr.Left().(ir.OperandColumn)
	operands := expr.Right().([]ir.Operand)

	columns := r.columnsFetcher(column.Schema(), column.TableName())
	if columns == nil {
		panic(fmt.Errorf("somehow we do not have columns for table %s", column.TableName()))
	}

	idx := randGen.Intn(len(operands))
	operand := operands[idx]

	r.stepsConstants = append(r.stepsConstants, &StepEqualConstant{
		columnMeta: columnByName(columns, column.ColumnName()),
		column:     column,
		operand:    operand,
	})
}

// procNot invert/negate the following Operations:
// - OpEqual -> OpNotEqual
// - OpNotEqual -> OpEqual
// - OpGT -> OpLE, OpGE -> OpLT and vice versa
// - OpIn -> ?
func (r *RowGenerator) procNot(expr any) {
	switch exprT := expr.(type) {
	case ir.OpAnd:
		r.procNot(exprT.Left())
		r.procNot(exprT.Right())
	case ir.OpOr:
		r.procOrNot(exprT)
	case ir.OpEqual:
		r.procNotEq(exprT.Negate())
	case ir.OpNotEqual:
		r.procEq(exprT.Negate())
	case ir.OpNot:
		// TODO
		panic("not operator not implemented")
	case ir.OpIn:
		// r.procIn(exprT)
		// TODO
		panic("'NOT IN' not implemented yet")
	case ir.OpGT:
		r.procRange(exprT.Negate())
	case ir.OpGE:
		r.procRange(exprT.Negate())
	case ir.OpLT:
		r.procRange(exprT.Negate())
	case ir.OpLE:
		r.procRange(exprT.Negate())
	case ir.OpNoOp:
	default:
		panic(fmt.Errorf("unknown type %T", expr))
	}
}

var procOrNotRnd = rand.NewSource(time.Now().UnixNano())

func (r *RowGenerator) procOrNot(expr ir.OpOr) {
	randGen := rand.New(procOrNotRnd)
	// false for Left, true for Right
	if randGen.Intn(2) == 0 {
		r.procNot(expr.Left())
	} else {
		r.procNot(expr.Right())
	}
}

func (r *RowGenerator) procSteps() {
	// start with constants
	for _, step := range r.stepsConstants {
		step.Apply(r.rows, true)
		// apply the constant to other columns that may share the same value by pointing to this column
		for _, step := range r.stepsColumns {
			step.Apply(r.rows, false)
		}
	}

	// columns to columns last
	for _, step := range r.stepsColumns {
		step.Apply(r.rows, true)
		// apply the value to other columns that may share the relationship value by pointing to this column
		for _, step := range r.stepsColumns {
			step.Apply(r.rows, false)
		}
	}
}

type Step interface {
	Apply(RowSet, bool)
}

type StepEqualConstant struct {
	done       bool
	columnMeta Column
	column     ir.OperandColumn
	operand    ir.Operand
}

func (s *StepEqualConstant) Apply(rows RowSet, _ bool) {
	if s.done {
		return
	}
	s.done = true

	// if it's a constant it has to be a string because for now it comes as is from the query string
	switch val := s.operand.Value(nil).(type) {
	case string, nil:
		row := rows.GetForTable(s.column.FQTableName())
		s.columnMeta.SetValue(row, val)
	default:
		panic(fmt.Errorf("expected string or nil but got %T", s.operand.Value(nil)))
	}
}

// StepDomain assigns a column a value from its domain, unless the column already has a value.
type StepDomain struct {
	done       bool
	columnMeta Column
	column     ir.OperandColumn
	domain     ValueDomain
}

func (s *StepDomain) Apply(rows RowSet, _ bool) {
	if s.done {
		return
	}
	s.done = true

	row := rows.GetForTable(s.column.FQTableName())
	if _, isSet := row[s.column.ColumnName()]; isSet {
		// e.g.: 'c1 > 42 AND c1 = 100'
		return
	}
	s.columnMeta.SetValue(row, s.domain.Gen())
}

// StepEqualColumn represents a comparison between columns
type StepEqualColumn struct {
	done         bool
	leftColumns  []Column
	left         ir.OperandColumn
	rightColumns []Column
	right        ir.OperandColumn
}

// Apply populates rows with the necessary columns and table rows to satisfy the constraint.
// rows is populated with fully qualified table names, e.g. table names are prefixed with schema. When it is not
// possible to take the schema from the query 'public' is used as default. Column names are not prefixed with the
// table name as they are already in the context of a table.
// onNonEmpty should be true after all constants were applied
func (s *StepEqualColumn) Apply(rows RowSet, onNonEmpty bool) {
	if s.done {
		return
	}

	leftRow := rows.GetForTable(s.left.FQTableName())
	leftCol, leftIsSet := leftRow[s.left.ColumnName()]
	rightRow := rows.GetForTable(s.right.FQTableName())
	rightCol, rightIsSet := rightRow[s.right.ColumnName()]

	switch {
	case leftIsSet && rightIsSet:
		s.done = true
		return
	case !leftIsSet && !rightIsSet:
		if !onNonEmpty {
			// if any of the columns in this step already have a value, search every existing row and set that value
			// everywhere this column is referenced
			s.assignValsToCols(rows, leftRow, rightRow)
			return
		}
		leftCol := columnByName(s.leftColumns, s.left.ColumnName())
		val := leftCol.Populate(leftRow)
		rightCol := columnByName(s.rightColumns, s.right.ColumnName())
		rightRow[rightCol.Name()] = val
		s.done = true
	case !leftIsSet:
		col := columnByName(s.leftColumns, s.left.ColumnName())
		col.SetValue(leftRow, rightCol.Val())
		s.done = true
	case !rightIsSet:
		col := columnByName(s.rightColumns, s.right.ColumnName())
		col.SetValue(rightRow, leftCol.Val())
		s.done = true
	}
}

// Iterates over rows which have populated values and, if any of the columns in this step are part of a row set its
// value into leftRow or rightRow.
func (s *StepEqualColumn) assignValsToCols(rows RowSet, leftRow, rightRow Row) {
	leftCol := columnByName(s.leftColumns, s.left.ColumnName())
	rightCol := columnByName(s.rightColumns, s.right.ColumnName())
	for tableName, row := range rows {
		for colName, val := range row {
			if s.left.FQTableName() == tableName && s.left.ColumnName() == colName {
				leftCol.SetValue(leftRow, val)
			} else if s.right.FQTableName() == tableName && s.right.ColumnName() == colName {
				rightCol.SetValue(rightRow, val)
			}
		}
	}
}

type StepNotEqualConstant struct {
	done       bool
	columnMeta Column
	column     ir.OperandColumn
	operand    ir.Operand
}

func (s *StepNotEqualConstant) Apply(rows RowSet, _ bool) {
	if s.done {
		return
	}
	s.done = true

	// if it's a constant it has to be a string because for now it comes as is from the query string
	switch val := s.operand.Value(nil).(type) {
	case string, nil:
		row := rows.GetForTable(s.column.FQTableName())
		for range 10 {
			col := s.columnMeta.Populate(row)
			if col.Cmp(col.Val(), val) != 0 {
				return
			}
		}
		// I want to flag if this fails to understand if this is enough or it needs a more sophisticated approach
		panic(fmt.Errorf("unable to generate value different from %+v", val))
	default:
		panic(fmt.Errorf("expected string or nil but got %T", s.operand.Value(nil)))
	}
}

type StepNotEqualColumn struct {
	done         bool
	leftColumns  []Column
	left         ir.OperandColumn
	rightColumns []Column
	right        ir.OperandColumn
}

func (s *StepNotEqualColumn) Apply(rows RowSet, onNonEmpty bool) {
	if s.done {
		return
	}

	leftRow := rows.GetForTable(s.left.FQTableName())
	leftCol, leftIsSet := leftRow[s.left.ColumnName()]
	rightRow := rows.GetForTable(s.right.FQTableName())
	rightCol, rightIsSet := rightRow[s.right.ColumnName()]

	switch {
	case leftIsSet && rightIsSet:
		s.done = true
		return
	case !leftIsSet && !rightIsSet:
		if !onNonEmpty {
			// if any of the columns in this step already have a value, search every existing row and set that value
			// everywhere this column is referenced
			s.assignValsToCols(rows, leftRow, rightRow)
			return
		}

		leftColMeta := columnByName(s.leftColumns, s.left.ColumnName())
		rightColMeta := columnByName(s.rightColumns, s.right.ColumnName())
		newLeftCol := leftColMeta.Populate(leftRow)
		for range 10 { // TODO
			newRightCol := rightColMeta.Populate(rightRow)
			if newRightCol.Cmp(newRightCol.Val(), newLeftCol.Val()) != 0 {
				s.done = true
				return
			}
		}
		// I want to flag if this fails to understand if this is enough or it needs a more sophisticated approach
		panic(fmt.Errorf("unable to generate value different from %+v", newLeftCol.Val()))
	case !leftIsSet:
		leftColMeta := columnByName(s.leftColumns, s.left.ColumnName())
		rightVal := rightCol.Val()
		for range 10 {
			newLeftCol := leftColMeta.Populate(leftRow)
			if newLeftCol.Cmp(newLeftCol.Val(), rightVal) != 0 {
				s.done = true
				return
			}
		}
		// I want to flag if this fails to understand if this is enough or it needs a more sophisticated approach
		panic(fmt.Errorf("unable to generate value different from %+v", rightVal))
	case !rightIsSet:
		rightColMeta := columnByName(s.rightColumns, s.right.ColumnName())
		leftVal := leftCol.Val()
		for range 10 {
			newRightCol := rightColMeta.Populate(rightRow)
			if newRightCol.Cmp(newRightCol.Val(), leftVal) != 0 {
				s.done = true
				return
			}
		}
		// I want to flag if this fails to understand if this is enough or it needs a more sophisticated approach
		panic(fmt.Errorf("unable to generate value different from %+v", leftVal))
	}
}

// Iterates over rows which has populated values and, if any of the columns in this step are part of a row set its
// value into leftRow or rightRow.
func (s *StepNotEqualColumn) assignValsToCols(rows RowSet, leftRow, rightRow Row) {
	leftCol := columnByName(s.leftColumns, s.left.ColumnName())
	rightCol := columnByName(s.rightColumns, s.right.ColumnName())
	for tableName, row := range rows {
		for colName, val := range row {
			if s.left.FQTableName() == tableName && s.left.ColumnName() == colName {
				leftCol.SetValue(leftRow, val)
			} else if s.right.FQTableName() == tableName && s.right.ColumnName() == colName {
				rightCol.SetValue(rightRow, val)
			}
		}
	}
}

// Returns the column in columns with name, panics if name is unknown.
// name can be 'column_name' or 'table.column_name'.
func columnByName(columns []Column, name string) Column {
	parts := strings.Split(name, ".")
	colName := parts[0]
	if len(parts) > 1 {
		colName = parts[1]
	}

	for _, col := range columns {
		if colName == col.Name() {
			return col
		}
	}

	panic(fmt.Errorf("somehow I do not know this column %s", name))
}

func isOperandColumn(op any) (ir.OperandColumn, bool) {
	switch col := op.(type) {
	case ir.OperandColumn:
		return col, true
	default:
		return ir.OperandColumn{}, false
	}
}
