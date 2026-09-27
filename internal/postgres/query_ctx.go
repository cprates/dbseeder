package postgres

import (
	"context"
	"fmt"
	"slices"
)

type (
	queryScopeStackItem map[string]TableMeta
	queryScopeStack     []queryScopeStackItem
)

func (q queryScopeStack) lookup(schema, tableOrAlias, colName string) (string, string) {
	// if tableOrAlias is given do the direct lookup in the current scope
	if tableOrAlias != "" {
		if schema == "" {
			schema = DefaultSchema
		}
		for _, scope := range slices.Backward(q) {
			tableSchema, tableName := scope.lookupByTableName(schema, tableOrAlias, colName)
			if tableName != "" {
				return tableSchema, tableName
			}
		}
		panic(fmt.Errorf("unknown column %s.%s.%s", schema, tableOrAlias, colName))
	}

	// unknown table, search the whole scope
	for _, scope := range slices.Backward(q) {
		tableSchema, tableName := scope.lookupScope(colName)
		if tableName != "" {
			return tableSchema, tableName
		}
	}
	panic(fmt.Errorf("unknown column %s", colName))
}

// Expects schema to not be empty. Returns empty strings if unable to find colName.
func (q queryScopeStackItem) lookupByTableName(schema, tableOrAlias, colName string) (string, string) {
	// if it's a known table alias no need to do anything else
	if tableMeta, ok := q[tableOrAlias]; ok {
		if col := tableMeta.ColumnByName(colName); col == nil {
			return "", ""
		}
		return tableMeta.Name().schema, tableMeta.Name().name
	}

	// at this point tableOrAlias is an alias in another scope or it's a table name
	tableMeta, exists := q[schema+"."+tableOrAlias]
	if !exists {
		return "", ""
	}

	if col := tableMeta.ColumnByName(colName); col == nil {
		return "", ""
	}

	return tableMeta.Name().schema, tableMeta.Name().name
}

func (q queryScopeStackItem) lookupScope(colName string) (string, string) {
	var targetTable *TableMeta
	for _, scopeTableMeta := range q {
		for _, metaColName := range scopeTableMeta.columns {
			if colName == metaColName.Name {
				if targetTable != nil && targetTable.name.FQ() != scopeTableMeta.Name().FQ() {
					panic(fmt.Errorf(
						"ambiguous column %q in scope between %s and %s",
						colName, targetTable.name.FQ(), scopeTableMeta.Name().FQ(),
					))
				}

				targetTable = &scopeTableMeta
			}
		}
	}

	if targetTable != nil {
		return targetTable.Name().Schema(), targetTable.Name().Simple()
	}

	return "", ""
}

type queryContext struct {
	ctx context.Context
	// used to make unit tests easier without the need for the getTableF
	test       bool
	getTableF  tableGetter
	scopeStack queryScopeStack
}

func (q *queryContext) addToScope(schema, tableName, tableAlias string) {
	if q.test {
		return
	}
	if schema == "" {
		schema = DefaultSchema
	}

	table, err := q.getTableF(q.ctx, schema, tableName)
	if err != nil {
		panic(err)
	}
	currScope := len(q.scopeStack) - 1
	metaFromTableName, tableNameExists := q.scopeStack[currScope][schema+"."+tableName]
	if tableAlias != "" {
		metaFromTableAlias, aliasExists := q.scopeStack[currScope][tableAlias]
		if aliasExists && !tableNameExists {
			panic(fmt.Errorf(
				"ambiguous table alias %q linked to tables %q and %q",
				tableAlias, metaFromTableAlias.Name().FQ(), schema+"."+tableName,
			))
		}
		if metaFromTableAlias.name.FQ() != metaFromTableName.Name().FQ() {
			panic(fmt.Errorf(
				"ambiguous table alias %q linked to tables %q and %q",
				tableAlias, metaFromTableAlias.Name().FQ(), metaFromTableName.Name().FQ(),
			))
		}
	}

	q.scopeStack[currScope][table.Name().FQ()] = table
	if tableAlias != "" {
		q.scopeStack[currScope][tableAlias] = table
	}
}

// newScope adds a new query scope to the context. It should be called on every 'SELECT' found in a query.
func (q *queryContext) newScope() {
	q.scopeStack = append(q.scopeStack, map[string]TableMeta{})
}

func (q *queryContext) lookupColumn(schema, tableOrAlias, columnName string) (string, string) {
	if q.test {
		return schema, tableOrAlias
	}
	return q.scopeStack.lookup(schema, tableOrAlias, columnName)
}
