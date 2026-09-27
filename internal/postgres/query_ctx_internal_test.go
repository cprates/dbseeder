package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testGetTableF = func(_ context.Context, schema, name string) (TableMeta, error) {
		table1 := TableMeta{
			name: TableName{
				schema: DefaultSchema,
				name:   "table1",
			},
			columns: []*ColumnMeta{
				{Name: "id"}, {Name: "c1"}, {Name: "c2"},
			},
		}
		table4 := TableMeta{
			name: TableName{
				schema: DefaultSchema,
				name:   "table4",
			},
			columns: []*ColumnMeta{
				{Name: "c1"}, {Name: "c3"},
			},
		}
		m := map[string]TableMeta{
			DefaultSchema + ".table1": table1,
			"t1":                      table1, // alias for table1
			DefaultSchema + ".table2": {
				name: TableName{
					schema: DefaultSchema,
					name:   "table2",
				},
				columns: []*ColumnMeta{
					{Name: "id"}, {Name: "c1"}, {Name: "c3"},
				},
			},
			"myschema1.table2": {
				name: TableName{
					schema: "myschema1",
					name:   "table2",
				},
				columns: []*ColumnMeta{
					{Name: "id"}, {Name: "c1"}, {Name: "c3"},
				},
			},
			DefaultSchema + ".table3": {
				name: TableName{
					schema: DefaultSchema,
					name:   "table3",
				},
				columns: []*ColumnMeta{
					{Name: "id"}, {Name: "c1"}, {Name: "c2"}, {Name: "c3"},
				},
			},
			DefaultSchema + ".table4": table4,
			"t4":                      table4, // alias for table4
		}
		meta, ok := m[schema+"."+name]
		if !ok {
			return TableMeta{}, fmt.Errorf("unknown table %s on schema %s", name, schema)
		}

		return meta, nil
	}
)

func TestQueryContext_LookupColumnWithTableName(t *testing.T) {
	for _, test := range []struct {
		description        string
		setupScope         func(qCtx *queryContext)
		lookupSchema       string
		lookupTableOrAlias string
		lookupColumn       string
		expectSchema       string
		expectTableName    string
		expectPanic        string
	}{
		{
			description: "unknown column",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table1", "")
			},
			lookupTableOrAlias: "table1",
			lookupColumn:       "cx",
			expectPanic:        "unknown column public.table1.cx",
		},
		{
			description: "unknown column in different schema",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table1", "")
			},
			lookupSchema:       "myschema",
			lookupTableOrAlias: "table1",
			lookupColumn:       "c1",
			expectPanic:        "unknown column myschema.table1.c1",
		},
		{
			description: "ambiguous column in same schema",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table1", "")
				qCtx.addToScope(DefaultSchema, "table2", "")
			},
			lookupColumn: "c1",
			expectPanic:  `ambiguous column "c1" in scope between public.table1 and public.table2`,
		},
		{
			description: "selects right table from multiple schemas",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table2", "")
				qCtx.addToScope("myschema1", "table2", "")
			},
			lookupSchema:       "myschema1",
			lookupTableOrAlias: "table2",
			lookupColumn:       "c1",
			expectSchema:       "myschema1",
			expectTableName:    "table2",
		},
		{
			description: "selects right table without specifying schema from multiple schemas",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table2", "")
				qCtx.addToScope("myschema1", "table2", "t2")
			},
			lookupSchema:       "myschema1",
			lookupTableOrAlias: "table2",
			lookupColumn:       "c1",
			expectSchema:       "myschema1",
			expectTableName:    "table2",
		},
		{
			description: "selects right table without specifying schema based on alias from multiple schemas",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table2", "")
				qCtx.addToScope("myschema1", "table2", "t2")
			},
			lookupTableOrAlias: "t2",
			lookupColumn:       "c1",
			expectSchema:       "myschema1",
			expectTableName:    "table2",
		},
		{
			description: "finds right table in bottom scope",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table2", "")
				qCtx.pushNewScope()
				qCtx.addToScope(DefaultSchema, "table1", "")
				qCtx.addToScope(DefaultSchema, "table3", "")
			},
			lookupTableOrAlias: "table1",
			lookupColumn:       "id",
			expectSchema:       DefaultSchema,
			expectTableName:    "table1",
		},
		{
			description: "finds right table from alias in bottom scope",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table1", "t1")
				qCtx.pushNewScope()
				qCtx.addToScope(DefaultSchema, "table2", "t2")
				qCtx.addToScope(DefaultSchema, "table3", "t3")
			},
			lookupTableOrAlias: "t1",
			lookupColumn:       "id",
			expectSchema:       DefaultSchema,
			expectTableName:    "table1",
		},
	} {
		t.Run(test.description, func(t *testing.T) {
			qCtx := queryContext{
				ctx:        context.Background(),
				getTableF:  testGetTableF,
				scopeStack: queryScopeStack{queryScopeStackItem{}},
			}
			test.setupScope(&qCtx)
			if test.expectPanic != "" {
				require.PanicsWithError(t, test.expectPanic, func() {
					_, _ = qCtx.lookupColumn(test.lookupSchema, test.lookupTableOrAlias, test.lookupColumn)
				})
				return
			}

			schema, tableName := qCtx.lookupColumn(test.lookupSchema, test.lookupTableOrAlias, test.lookupColumn)
			require.Equal(t, test.expectSchema, schema)
			require.Equal(t, test.expectTableName, tableName)
		})
	}
}

func TestQueryContext_LookupColumnWithoutTableName(t *testing.T) {
	for _, test := range []struct {
		description     string
		setupScope      func(qCtx *queryContext)
		lookupColumn    string
		expectSchema    string
		expectTableName string
		expectPanic     bool
	}{
		{
			description: "unknown column",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table1", "t1")
			},
			lookupColumn: "cx",
			expectPanic:  true,
		},
		{
			description: "ambiguous column in same scope",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table1", "")
				qCtx.addToScope(DefaultSchema, "table3", "")
			},
			lookupColumn: "id",
			expectPanic:  true,
		},
		{
			description: "returns table from the first scope where it's found",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table1", "")
				qCtx.pushNewScope()
				qCtx.addToScope(DefaultSchema, "table3", "")
				qCtx.pushNewScope()
				qCtx.addToScope(DefaultSchema, "table4", "")
			},
			lookupColumn:    "c2",
			expectSchema:    DefaultSchema,
			expectTableName: "table3",
		},
	} {
		t.Run(test.description, func(t *testing.T) {
			qCtx := queryContext{
				ctx:        context.Background(),
				getTableF:  testGetTableF,
				scopeStack: queryScopeStack{queryScopeStackItem{}},
			}
			test.setupScope(&qCtx)
			if test.expectPanic {
				require.Panics(t, func() {
					_, _ = qCtx.lookupColumn("", "", test.lookupColumn)
				})
				return
			}

			schema, tableName := qCtx.lookupColumn("", "", test.lookupColumn)
			require.Equal(t, test.expectSchema, schema)
			require.Equal(t, test.expectTableName, tableName)
		})
	}
}

func TestQueryContext_addToScope(t *testing.T) {
	for _, test := range []struct {
		description string
		setupScope  func(qCtx *queryContext)
		assert      func(qCtx *queryContext)
		expectPanic string
	}{
		{
			description: "errors if added table is in wrong schema",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope("mydomain1", "table1", "")
			},
			expectPanic: "unknown table table1 on schema mydomain1",
		},
		{
			description: "DefaultSchema is used when not given",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope("", "table1", "")
			},
			assert: func(qCtx *queryContext) {
				_, ok := qCtx.scopeStack[0][DefaultSchema+".table1"]
				require.True(t, ok)
			},
		},
		{
			description: "errors if table and alias already exist but alias points to a different table in same scope",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table1", "t1")
				qCtx.addToScope(DefaultSchema, "table2", "t2")
				qCtx.addToScope(DefaultSchema, "table1", "t2")
			},
			expectPanic: `ambiguous table alias "t2" linked to tables "public.table2" and "public.table1"`,
		},
		{
			description: "same table added twice in the same scope does not error",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table1", "t1")
				qCtx.addToScope(DefaultSchema, "table1", "t1")
			},
			assert: func(qCtx *queryContext) {
				require.Len(t, qCtx.scopeStack[0], 2)
				_, ok := qCtx.scopeStack[0][DefaultSchema+".table1"]
				require.True(t, ok)
				_, ok = qCtx.scopeStack[0]["t1"]
				require.True(t, ok)
			},
		},
		{
			description: "alias is set when given",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope("", "table1", "t1")
			},
			assert: func(qCtx *queryContext) {
				_, ok := qCtx.scopeStack[0][DefaultSchema+".table1"]
				require.True(t, ok)
				_, ok = qCtx.scopeStack[0]["t1"]
				require.True(t, ok)
			},
		},
		{
			description: "same table name in different schemas is not seen as ambiguity",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table2", "")
				qCtx.addToScope("myschema1", "table2", "")
			},
			assert: func(qCtx *queryContext) {
				_, ok := qCtx.scopeStack[0][DefaultSchema+".table2"]
				require.True(t, ok)
				_, ok = qCtx.scopeStack[0]["myschema1.table2"]
				require.True(t, ok)
			},
		},
		{
			description: "errors if same alias is assigned to different table names in same scope",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table2", "t2")
				qCtx.addToScope(DefaultSchema, "table1", "t2")
			},
			expectPanic: `ambiguous table alias "t2" linked to tables "public.table2" and "public.table1"`,
		},
		{
			description: "errors if same alias is assigned to same table name in different schemas in same scope",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table2", "t2")
				qCtx.addToScope("myschema1", "table2", "t2")
			},
			expectPanic: `ambiguous table alias "t2" linked to tables "public.table2" and "myschema1.table2"`,
		},
		{
			description: "no errors if same alias is assigned to same table names in different schemas in different scope",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table2", "t2")
				qCtx.pushNewScope()
				qCtx.addToScope("myschema1", "table2", "t2")
			},
			assert: func(*queryContext) {},
		},
		{
			description: "no errors if same alias is assigned to same table names in same schemas in different scope",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table2", "t2")
				qCtx.pushNewScope()
				qCtx.addToScope(DefaultSchema, "table2", "t2")
			},
			assert: func(*queryContext) {},
		},
		{
			description: "no errors if same alias is assigned to different table names in different schemas in different scope",
			setupScope: func(qCtx *queryContext) {
				qCtx.addToScope(DefaultSchema, "table2", "t2")
				qCtx.pushNewScope()
				qCtx.addToScope(DefaultSchema, "table1", "t2")
			},
			assert: func(*queryContext) {},
		},
	} {
		t.Run(test.description, func(t *testing.T) {
			qCtx := queryContext{
				ctx:        context.Background(),
				getTableF:  testGetTableF,
				scopeStack: queryScopeStack{queryScopeStackItem{}},
			}
			if test.expectPanic != "" {
				require.PanicsWithError(t, test.expectPanic, func() {
					test.setupScope(&qCtx)
				})
				return
			}

			test.setupScope(&qCtx)
			test.assert(&qCtx)
		})
	}
}
