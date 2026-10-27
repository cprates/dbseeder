// go:build integration_tests

package seeder

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"

	"github.com/cprates/dbseeder/internal/ir"
	"github.com/cprates/dbseeder/internal/logger"
	"github.com/cprates/dbseeder/internal/postgres"
)

const (
	TestsSchema = "public"
)

func TestGenValues(t *testing.T) {
	t.Parallel()

	testsSet := []struct {
		description string
		tables      []string
		dbSetup     string
		sqlFilter   string
		run         func(t *testing.T, vConst *RowGenerator)
	}{
		{
			description: "same column compared more than once against other columns",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text
						);
						CREATE TABLE genvaluestable2 (
							id text,
							c1 text
						);`,
			sqlFilter: `
			select * from genvaluestable1 join genvaluestable2 on genvaluestable1.id = genvaluestable2.id where
				genvaluestable2.c1 = genvaluestable1.id;`,
			tables: []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.Equal(t, row1["id"].Val(), row2["id"].Val())
				require.Equal(t, row1["id"].Val(), row2["c1"].Val())
			},
		},
		{
			description: "true condition with constants does not generates rows",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable;
						CREATE TABLE genvaluestable (
							id int
						);`,
			sqlFilter: "SELECT * FROM genvaluestable WHERE 1=1;",
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 0)
			},
		},
		{
			description: "false condition with constants does not generates rows",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable;
						CREATE TABLE genvaluestable (
							id int
						);`,
			sqlFilter: "SELECT * FROM genvaluestable WHERE id<>42;",
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 1)
			},
		},
		{
			description: "or branches are randomly chosen",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable;
						CREATE TABLE genvaluestable (
							id text
						);`,
			sqlFilter: `select * from genvaluestable where genvaluestable.id = 'a' OR genvaluestable.id = 'b' OR genvaluestable.id = 'c';`,
			run: func(t *testing.T, vConst *RowGenerator) {
				observed := map[string]struct{}{}
				for range 50 {
					rowSet := vConst.Gen()
					require.Len(t, rowSet, 1)
					row := rowSet.GetForTable("public.genvaluestable")
					val := row["id"].Val().(string)
					switch val {
					case "a", "b", "c":
						observed[val] = struct{}{}
					default:
						t.Fatalf("value %s not expected", val)
					}
				}
				require.Contains(t, observed, "a")
				require.Contains(t, observed, "b")
				require.Contains(t, observed, "c")
			},
		},
		{
			description: "mix of comparisons between tables constants ANDs and ORs work",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text
						);
						CREATE TABLE genvaluestable2 (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable1 join genvaluestable2 on genvaluestable1.id = genvaluestable2.id where (genvaluestable1.id = 'abc' OR genvaluestable2.c1 = 'qwerty');`,
			tables:    []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.Equal(t, row1["id"].Val(), row2["id"].Val())
				if row1["id"].Val() != "abc" {
					require.Equal(t, "qwerty", row2["c1"].Val())
				}
			},
		},
		{
			description: "comparison between columns from the same table",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable;
						CREATE TABLE genvaluestable (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable where genvaluestable.id = genvaluestable.c1;`,
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 1)
				row := rowSet.GetForTable("public.genvaluestable")
				require.Equal(t, row["id"].Val(), row["c1"].Val())
			},
		},
		{
			description: "comparison of a column with itself",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable;
						CREATE TABLE genvaluestable (
							id text not null
						);`,
			sqlFilter: `select * from genvaluestable where genvaluestable.id = genvaluestable.id;`,
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 1)
				row := rowSet.GetForTable("public.genvaluestable")
				require.NotNil(t, row["id"].Val())
			},
		},
		{
			description: "comparison between columns and constants",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable;
						CREATE TABLE genvaluestable (
							id text not null,
							c1 text not null
						);`,
			sqlFilter: `select * from genvaluestable where genvaluestable.id = 'abc' AND genvaluestable.c1 = 'qwerty';`,
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 1)
				row := rowSet.GetForTable("public.genvaluestable")
				require.Equal(t, "abc", row["id"].Val())
				require.Equal(t, "qwerty", row["c1"].Val())
			},
		},
		{
			description: "test constant assignment - case 1",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text,
							c1 text
						);
						CREATE TABLE genvaluestable2 (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable1 join genvaluestable2
							  on genvaluestable1.c1 = genvaluestable1.id where
							    genvaluestable1.id = genvaluestable2.id AND
								(genvaluestable1.id = 'abc' OR genvaluestable1.id = genvaluestable2.c1);`,
			tables: []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.Equal(t, row1["id"].Val(), row1["c1"].Val())
				require.Equal(t, row1["id"].Val(), row2["id"].Val())
				require.True(t, row1["id"].Val() == "abc" || row1["id"].Val() == row2["c1"].Val())
			},
		},
		{
			description: "test constant assignment - case 2",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text,
							c1 text
						);
						CREATE TABLE genvaluestable2 (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable1
							join genvaluestable2 on genvaluestable1.c1 = genvaluestable1.id where
							genvaluestable1.id = genvaluestable2.id AND
							(genvaluestable1.id = 'abc' OR genvaluestable2.c1 = 'qwerty');`,
			tables: []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.Equal(t, row1["id"].Val(), row1["c1"].Val())
				require.Equal(t, row1["id"].Val(), row2["id"].Val())
				require.True(t, row1["id"].Val() == "abc" || row2["c1"].Val() == "qwerty")
			},
		},
		{
			description: "test constant assignment - case 3",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text,
							c1 text
						);
						CREATE TABLE genvaluestable2 (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable1
							join genvaluestable2 on	genvaluestable1.c1 = genvaluestable1.id where
							genvaluestable1.id = genvaluestable2.id AND
							(genvaluestable1.id = 'abc' OR genvaluestable2.id = 'qwerty');`,
			tables: []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.Equal(t, row1["id"].Val(), row1["c1"].Val())
				require.Equal(t, row1["id"].Val(), row2["id"].Val())
				require.True(t, row1["id"].Val() == "abc" || row2["id"].Val() == "qwerty")
			},
		},
		{
			description: "test constant assignment - case 4",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text,
							c1 text
						);
						CREATE TABLE genvaluestable2 (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable1
							join genvaluestable2 on genvaluestable1.id = genvaluestable2.id where
							genvaluestable1.c1 = genvaluestable1.id AND
							(genvaluestable1.id = 'abc' OR genvaluestable2.c1 = 'qwerty');`,
			tables: []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.Equal(t, row1["id"].Val(), row1["c1"].Val())
				require.Equal(t, row1["id"].Val(), row2["id"].Val())
				require.True(t, row1["id"].Val() == "abc" || row2["c1"].Val() == "qwerty")
			},
		},
		{
			description: "test constant assignment - case 5",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text,
							c1 text
						);
						CREATE TABLE genvaluestable2 (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable1
							join genvaluestable2 on genvaluestable1.id = genvaluestable2.id where
							genvaluestable1.id = genvaluestable1.c1 AND
							(genvaluestable1.id = 'abc' OR genvaluestable2.id = 'qwerty');`,
			tables: []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.Equal(t, row1["id"].Val(), row1["c1"].Val())
				require.Equal(t, row1["id"].Val(), row2["id"].Val())
				require.True(t, row1["id"].Val() == "abc" || row2["id"].Val() == "qwerty")
			},
		},
		{
			description: "test constant assignment - case 6",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text,
							c1 text
						);
						CREATE TABLE genvaluestable2 (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable1
							join genvaluestable2 on genvaluestable2.id = genvaluestable1.id where
							genvaluestable1.id = genvaluestable1.c1 AND
							(genvaluestable1.id = 'abc' OR genvaluestable2.id = 'qwerty');`,
			tables: []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.Equal(t, row1["id"].Val(), row1["c1"].Val())
				require.Equal(t, row1["id"].Val(), row2["id"].Val())
				require.True(t, row1["id"].Val() == "abc" || row2["id"].Val() == "qwerty")
			},
		},
		{
			description: "test constant assignment - case 7",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text,
							c1 text
						);
						CREATE TABLE genvaluestable2 (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable1
							join genvaluestable2 on genvaluestable1.c1 = genvaluestable1.id where
							genvaluestable1.id = genvaluestable2.id AND
							genvaluestable2.c1 = 'qwerty';`,
			tables: []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.Equal(t, row1["c1"].Val(), row1["id"].Val())
				require.Equal(t, row1["id"].Val(), row2["id"].Val())
				require.Equal(t, "qwerty", row2["c1"].Val())
			},
		},
		{
			description: "test constant assignment - case 8",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text,
							c1 text
						);
						CREATE TABLE genvaluestable2 (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable1
							join genvaluestable2 on genvaluestable1.id = genvaluestable2.id where
							genvaluestable1.c1 = NULL AND
							genvaluestable2.c1 = 'qwerty';`,
			tables: []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.Nil(t, row1["c1"].Val())
				require.Equal(t, row1["id"].Val(), row2["id"].Val())
				require.Equal(t, "qwerty", row2["c1"].Val())
			},
		},
		{
			description: "test constant assignment - case 9",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text,
							c1 text
						);
						CREATE TABLE genvaluestable2 (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable1
							join genvaluestable2 on genvaluestable2.id = genvaluestable1.id where
							('abc' = genvaluestable1.id OR genvaluestable2.id = genvaluestable1.c1);`,
			tables: []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.Equal(t, row1["id"].Val(), row2["id"].Val())
				require.True(t, row1["id"].Val() == "abc" || row2["id"].Val() == row1["c1"].Val())
			},
		},
		{
			description: "test constant assignment - case 10",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text,
							c1 text
						);
						CREATE TABLE genvaluestable2 (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable1
							join genvaluestable2 on genvaluestable1.id = genvaluestable2.id where
							(genvaluestable1.id = 'abc' OR genvaluestable1.id = genvaluestable2.c1);`,
			tables: []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.Equal(t, row1["id"].Val(), row2["id"].Val())
				require.True(t, row1["id"].Val() == "abc" || row2["c1"].Val() == row1["id"].Val())
			},
		},
		{
			description: "test columns not equal in same table",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable;
						CREATE TABLE genvaluestable (
							id text,
							c1 text
						);`,
			sqlFilter: `select * from genvaluestable where genvaluestable.id != genvaluestable.c1;`,
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 1)
				row1 := rowSet.GetForTable("public.genvaluestable")
				require.NotEqual(t, row1["id"].Val(), row1["c1"].Val())
			},
		},
		{
			description: "test columns not equal in different tables",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text
						);
						CREATE TABLE genvaluestable2 (
							id text
						);`,
			sqlFilter: `select * from genvaluestable1 join genvaluestable2 on genvaluestable1.id != genvaluestable2.id;`,
			tables:    []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.NotEqual(t, row1["id"].Val(), row2["id"].Val())
			},
		},
		{
			description: "test columns not equal in different tables - left is set",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text
						);
						CREATE TABLE genvaluestable2 (
							id text
						);`,
			sqlFilter: `select * from genvaluestable1 join genvaluestable2 on genvaluestable1.id != genvaluestable2.id where genvaluestable1.id = 'qwerty';`,
			tables:    []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.NotEqual(t, row1["id"].Val(), row2["id"].Val())
				require.Equal(t, "qwerty", row1["id"].Val())
			},
		},
		{
			description: "test columns not equal in different tables - right is set",
			dbSetup: `DROP TABLE IF EXISTS genvaluestable1;
						DROP TABLE IF EXISTS genvaluestable2;
						CREATE TABLE genvaluestable1 (
							id text
						);
						CREATE TABLE genvaluestable2 (
							id text
						);`,
			sqlFilter: `select * from genvaluestable1 join genvaluestable2 on genvaluestable1.id != genvaluestable2.id where genvaluestable2.id = 'qwerty';`,
			tables:    []string{"genvaluestable1", "genvaluestable2"},
			run: func(t *testing.T, vConst *RowGenerator) {
				rowSet := vConst.Gen()
				require.Len(t, rowSet, 2)
				row1 := rowSet.GetForTable("public.genvaluestable1")
				row2 := rowSet.GetForTable("public.genvaluestable2")
				require.NotEqual(t, row1["id"].Val(), row2["id"].Val())
				require.Equal(t, "qwerty", row2["id"].Val())
			},
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			psql := setupPgTest(t, test.dbSetup)
			if len(test.tables) > 0 {
				for _, table := range test.tables {
					_, err := psql.GetTable(context.Background(), TestsSchema, table)
					require.NoError(t, err)
				}
			} else {
				_, err := psql.GetTable(context.Background(), TestsSchema, "genvaluestable")
				require.NoError(t, err)
			}

			expr, err := postgres.Parse(t.Context(), psql.inspector.TableWithSchema, test.sqlFilter)
			require.NoError(t, err)
			vc := NewRowGenerator(logger.NewNullLogger(), expr, psql.getTableColumnsAdapter(t.Context()))
			for range 100 {
				test.run(t, vc)
			}
		})
	}
}

func TestGenValuesRanges(t *testing.T) {
	t.Parallel()

	// unique table names per test as tests run in parallel
	testsSet := []struct {
		description string
		table       string
		dbSetup     string
		sqlFilter   string
		// text columns are not populated by the generator so, the filter can't be evaluated
		skipEval bool
		run      func(t *testing.T, row Row)
	}{
		{
			description: "integer range with exclusion",
			table:       "genrangestable1",
			dbSetup: `DROP TABLE IF EXISTS genrangestable1;
						CREATE TABLE genrangestable1 (c1 int2, c2 int4, c3 int8);`,
			sqlFilter: `select * from genrangestable1 where
								c1 > 42 and c1 <> 100 and c1 <= 123 and c2 = 42 and
								c3 >= -5 and c3 < 5 and c3 <> 0;`,
			run: func(t *testing.T, row Row) {
				c1 := row["c1"].Val().(int16)
				require.Greater(t, c1, int16(42))
				require.LessOrEqual(t, c1, int16(123))
				require.NotEqual(t, int16(100), c1)
				require.Equal(t, "42", row["c2"].Val())
				c3 := row["c3"].Val().(int64)
				require.GreaterOrEqual(t, c3, int64(-5))
				require.Less(t, c3, int64(5))
				require.NotEqual(t, int64(0), c3)
			},
		},
		{
			description: "equality takes precedence over range",
			table:       "genrangestable2",
			dbSetup: `DROP TABLE IF EXISTS genrangestable2;
						CREATE TABLE genrangestable2 (c1 int4);`,
			sqlFilter: `select * from genrangestable2 where c1 > 42 and c1 = 100 and c1 <= 123;`,
			run: func(t *testing.T, row Row) {
				require.Equal(t, "100", row["c1"].Val())
			},
		},
		{
			description: "constant on the left side",
			table:       "genrangestable3",
			dbSetup: `DROP TABLE IF EXISTS genrangestable3;
						CREATE TABLE genrangestable3 (c1 int4);`,
			sqlFilter: `select * from genrangestable3 where 42 < c1 and 45 >= c1 and 43 <> c1;`,
			run: func(t *testing.T, row Row) {
				c1 := row["c1"].Val().(int32)
				require.Contains(t, []int32{44, 45}, c1)
			},
		},
		{
			description: "negated ranges",
			table:       "genrangestable4",
			dbSetup: `DROP TABLE IF EXISTS genrangestable4;
						CREATE TABLE genrangestable4 (c1 int4);`,
			sqlFilter: `select * from genrangestable4 where c1 <= 45 AND NOT c1 <= 42;`,
			run: func(t *testing.T, row Row) {
				c1 := row["c1"].Val().(int32)
				require.Contains(t, []int32{43, 44, 45}, c1)
			},
		},
		{
			description: "floats",
			table:       "genrangestable5",
			dbSetup: `DROP TABLE IF EXISTS genrangestable5;
						CREATE TABLE genrangestable5 (c1 float4, c2 float8);`,
			sqlFilter: `select * from genrangestable5 where
								c1 > 42 and c1 <= 42.5 and c2 >= -1.5 and c2 < -1.25 and c2 <> -1.3;`,
			run: func(t *testing.T, row Row) {
				c1 := row["c1"].Val().(float32)
				require.Greater(t, c1, float32(42))
				require.LessOrEqual(t, c1, float32(42.5))
				c2 := row["c2"].Val().(float64)
				require.GreaterOrEqual(t, c2, -1.5)
				require.Less(t, c2, -1.25)
				require.NotEqual(t, -1.3, c2)
			},
		},
		{
			description: "numeric",
			table:       "genrangestable6",
			dbSetup: `DROP TABLE IF EXISTS genrangestable6;
						CREATE TABLE genrangestable6 (c1 numeric(6, 2), c2 numeric);`,
			sqlFilter: `select * from genrangestable6 where
								c1 > 10.005 and c1 < 10.04 and c1 <> 10.02 and c2 >= 1000;`,
			run: func(t *testing.T, row Row) {
				require.Contains(t, []string{"10.01", "10.03"}, row["c1"].Val())
				c2, err := strconv.ParseFloat(row["c2"].Val().(string), 64)
				require.NoError(t, err)
				require.GreaterOrEqual(t, c2, 1000.0)
			},
		},
		{
			description: "dates and timestamps",
			table:       "genrangestable7",
			dbSetup: `DROP TABLE IF EXISTS genrangestable7;
						CREATE TABLE genrangestable7 (c1 date, c2 timestamp, c3 timestamptz);`,
			sqlFilter: `select * from genrangestable7 where
								c1 > '2024-01-01' and c1 <= '2024-01-03' and c1 <> '2024-01-02' and
								c2 >= '2024-01-01 10:00:00' and c2 < '2024-01-01 11:00:00' and
								c3 > '2024-01-01T10:00:00Z';`,
			run: func(t *testing.T, row Row) {
				require.Equal(t, time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), row["c1"].Val())
				c2 := row["c2"].Val().(time.Time)
				require.False(t, c2.Before(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)))
				require.True(t, c2.Before(time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)))
				c3 := row["c3"].Val().(time.Time)
				require.True(t, c3.After(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)))
			},
		},
		{
			description: "ranges on text columns are ignored",
			table:       "genrangestable8",
			dbSetup: `DROP TABLE IF EXISTS genrangestable8;
						CREATE TABLE genrangestable8 (c1 text, c2 int4);`,
			sqlFilter: `select * from genrangestable8 where c1 > 'abc' and c2 > 0 and c2 < 2;`,
			skipEval:  true,
			run: func(t *testing.T, row Row) {
				require.NotContains(t, row, "c1")
				require.Equal(t, int32(1), row["c2"].Val())
			},
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			psql := setupPgTest(t, test.dbSetup)
			_, err := psql.GetTable(t.Context(), TestsSchema, test.table)
			require.NoError(t, err)

			expr, err := postgres.Parse(t.Context(), psql.inspector.TableWithSchema, test.sqlFilter)
			require.NoError(t, err)
			vc := NewRowGenerator(logger.NewNullLogger(), expr, psql.getTableColumnsAdapter(t.Context()))
			for range 200 {
				rowSet := vc.Gen()
				require.Len(t, rowSet, 1)
				row := rowSet.GetForTable(TestsSchema + "." + test.table)
				test.run(t, row)
				if test.skipEval {
					continue
				}
				// generated rows must satisfy the filter
				require.True(t, expr.Eval(map[string]ir.Row{TestsSchema + "." + test.table: seederToParserRow(row)}))
			}
		})
	}
}

func setupPgTest(t *testing.T, q string) *Postgres {
	t.Helper()

	db := getDB(t)
	insp := postgres.NewInspector(db)
	if q != "" {
		exec(t, db, q)
	}

	return &Postgres{inspector: insp}
}

func exec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	_, err := db.Exec(q)
	require.NoError(t, err)
}

func getDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_DB"),
	)
	db, err := sql.Open(postgres.DriverName, dsn)
	require.NoError(t, err)
	return db
}

func TestMain(m *testing.M) {
	if err := godotenv.Load("../.env"); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}
