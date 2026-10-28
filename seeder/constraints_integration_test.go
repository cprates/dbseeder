// go:build integration_tests

package seeder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConUniqueIdx(t *testing.T) {
	t.Parallel()

	testsSet := []struct {
		description string
		table       string
		dbSetup     string
		run         func(t *testing.T, table *Table)
	}{
		{
			description: "no collision adding same row twice if partial index expression evaluation is not true",
			dbSetup: `DROP TABLE IF EXISTS conuniqueidx;
						CREATE TABLE conuniqueidx (
							c1 integer,
							c2 integer
						);
						CREATE UNIQUE INDEX conuniqueidx_uidx1 ON conuniqueidx (c1) WHERE c2 > 42;`,
			table: "conuniqueidx",
			run: func(t *testing.T, table *Table) {
				row := Row{table.FQName(): nil}
				table.columns[0].SetValue(row, 2)
				table.columns[1].SetValue(row, 2)
				cons := table.constraints[0]
				ok := cons.Assert(row)
				require.True(t, ok)
				ok = cons.Assert(row)
				require.True(t, ok)
			},
		},
		{
			description: "finds collision adding same row twice if partial index expression evaluation is true",
			dbSetup: `DROP TABLE IF EXISTS conuniqueidx;
						CREATE TABLE conuniqueidx (
							c1 integer,
							c2 integer
						);
						CREATE UNIQUE INDEX conuniqueidx_uidx1 ON conuniqueidx (c1) WHERE c2 > 42;`,
			table: "conuniqueidx",
			run: func(t *testing.T, table *Table) {
				row := Row{table.FQName(): nil}
				table.columns[0].SetValue(row, 2)
				table.columns[1].SetValue(row, 2)
				cons := table.constraints[0]
				ok := cons.Assert(row)
				require.True(t, ok)
				table.columns[0].SetValue(row, 2)
				table.columns[1].SetValue(row, 43)
				ok = cons.Assert(row)
				require.False(t, ok)
			},
		},
		{
			description: "finds collision adding same row twice",
			dbSetup: `DROP TABLE IF EXISTS conuniqueidx2;
						CREATE TABLE conuniqueidx2 (
							c1 integer,
							c2 integer
						);
						CREATE UNIQUE INDEX conuniqueidx2_uidx1 ON conuniqueidx2 (c1);`,
			table: "conuniqueidx2",
			run: func(t *testing.T, table *Table) {
				row := Row{table.FQName(): nil}
				table.columns[0].SetValue(row, 2)
				table.columns[1].SetValue(row, 2)
				cons := table.constraints[0]
				ok := cons.Assert(row)
				require.True(t, ok)
				ok = cons.Assert(row)
				require.False(t, ok)
			},
		},
		{
			description: "name is correct",
			dbSetup: `DROP TABLE IF EXISTS conuniqueidx3;
						CREATE TABLE conuniqueidx3 (
							c1 integer
						);
						CREATE UNIQUE INDEX conuniqueidx3_uidx1 ON conuniqueidx3 (c1);`,
			table: "conuniqueidx3",
			run: func(t *testing.T, table *Table) {
				cons := table.constraints[0]
				require.Equal(t, "Unique Index conuniqueidx3_uidx1", cons.Name())
			},
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			psql := setupPgTest(t, test.dbSetup)
			table, err := psql.GetTable(context.Background(), TestsSchema, test.table)
			require.NoError(t, err)
			test.run(t, &table)
		})
	}
}

func TestConPK(t *testing.T) {
	t.Parallel()

	testsSet := []struct {
		description string
		table       string
		dbSetup     string
		run         func(t *testing.T, table *Table)
	}{
		{
			description: "finds collision adding same row twice",
			dbSetup: `DROP TABLE IF EXISTS conpkidx1;
						CREATE TABLE conpkidx1 (
							c1 integer PRIMARY KEY
						);`,
			table: "conpkidx1",
			run: func(t *testing.T, table *Table) {
				row := Row{table.FQName(): nil}
				table.columns[0].SetValue(row, 2)
				cons := table.constraints[0]
				ok := cons.Assert(row)
				require.True(t, ok)
				ok = cons.Assert(row)
				require.False(t, ok)
			},
		},
		{
			description: "name is correct",
			dbSetup: `DROP TABLE IF EXISTS conpkidx2;
						CREATE TABLE conpkidx2 (
							c1 integer PRIMARY KEY
						);`,
			table: "conpkidx2",
			run: func(t *testing.T, table *Table) {
				cons := table.constraints[0]
				require.Equal(t, "Primary Key conpkidx2_pkey", cons.Name())
			},
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			psql := setupPgTest(t, test.dbSetup)
			table, err := psql.GetTable(context.Background(), TestsSchema, test.table)
			require.NoError(t, err)
			test.run(t, &table)
		})
	}
}
