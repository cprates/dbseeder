//go:build integration_tests

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

func TestGetTableMetadata_UnknownTableOrNoColumnsErrors(t *testing.T) {
	t.Parallel()

	inspector := setupTest(t, "")
	_, err := inspector.Table(context.Background(), "some_dummy_table_that_is_not_created")
	require.Error(t, err)
	require.Equal(t, "getting columns for public.some_dummy_table_that_is_not_created: ERROR: relation \"some_dummy_table_that_is_not_created\" does not exist (SQLSTATE 42P01)", err.Error())
}

func TestGetTableMetadata_PrimaryKeyWithSingleColumn(t *testing.T) {
	t.Parallel()

	table := strings.ToLower(t.Name())
	setupQ := `
DROP TABLE IF EXISTS $name;
CREATE TABLE $name (
	id integer PRIMARY KEY,
	c1 integer
);`
	psql := setupTest(t, strings.ReplaceAll(setupQ, "$name", DefaultSchema+"."+table))
	td, err := psql.Table(context.Background(), table)
	require.NoError(t, err)

	uniqueIndexes := td.UniqueIndexes()
	require.Len(t, uniqueIndexes, 1)
	pkIndex := uniqueIndexes[0]
	require.Len(t, pkIndex.ConCols, 1)
	require.Equal(t, "id", pkIndex.ConCols[0].Name)
	require.Equal(t, "1", pkIndex.ConColsPosition)
	require.Contains(t, pkIndex.Definition, "CREATE UNIQUE INDEX")
	require.Equal(t, table+"_pkey", pkIndex.Name)
	require.True(t, pkIndex.PK)
	require.True(t, pkIndex.Unique)
	require.False(t, pkIndex.Partial)

	pk := td.PrimaryKey()
	require.Len(t, pk, 1)
	require.Equal(t, "id", pk[0].Name)

	pkColumn := td.ColumnByName("id")
	require.NotNil(t, pkColumn.PKConstraint)
	require.Equal(t, table+"_pkey", pkColumn.PKConstraint.Name)
}

func TestGetTableMetadata_ComposedPK(t *testing.T) {
	t.Parallel()

	table := strings.ToLower(t.Name())
	setupQ := `
DROP TABLE IF EXISTS $name;
CREATE TABLE $name (
	id integer,
	c1 text,
	PRIMARY KEY (id, c1)
);`
	psql := setupTest(t, strings.ReplaceAll(setupQ, "$name", DefaultSchema+"."+table))
	td, err := psql.Table(context.Background(), table)
	require.NoError(t, err)

	uniqueIndexes := td.UniqueIndexes()
	require.Len(t, uniqueIndexes, 1)
	pkIndex := uniqueIndexes[0]
	require.Len(t, pkIndex.ConCols, 2)
	require.Equal(t, "id", pkIndex.ConCols[0].Name)
	require.Equal(t, "c1", pkIndex.ConCols[1].Name)
	require.Equal(t, "1 2", pkIndex.ConColsPosition)
	require.Contains(t, pkIndex.Definition, "CREATE UNIQUE INDEX")
	require.Equal(t, table+"_pkey", pkIndex.Name)
	require.True(t, pkIndex.PK)
	require.True(t, pkIndex.Unique)
	require.False(t, pkIndex.Partial)

	pk := td.PrimaryKey()
	require.Len(t, pk, 2)
	require.Equal(t, "id", pk[0].Name)
	require.Equal(t, "c1", pk[1].Name)

	pkColumn := td.ColumnByName("id")
	require.NotNil(t, pkColumn.PKConstraint)
	require.Equal(t, table+"_pkey", pkColumn.PKConstraint.Name)
	pkColumn = td.ColumnByName("c1")
	require.NotNil(t, pkColumn.PKConstraint)
	require.Equal(t, table+"_pkey", pkColumn.PKConstraint.Name)
}

func TestGetTableMetadata_SingleFK(t *testing.T) {
	t.Parallel()

	table := strings.ToLower(t.Name())
	setupQ := `
DROP TABLE IF EXISTS $name;
DROP TABLE IF EXISTS $name_child;
CREATE TABLE $name_child (
	idc integer PRIMARY KEY,
	c1c text
);
CREATE TABLE $name (
	id integer,
	c1 integer REFERENCES $name_child (idc)
);`
	psql := setupTest(t, strings.ReplaceAll(setupQ, "$name", DefaultSchema+"."+table))
	td, err := psql.Table(context.Background(), table)
	require.NoError(t, err)

	fks := td.ForeignKeys()
	require.Len(t, fks, 1)
	fk := fks[0]
	require.Len(t, fk.ConCols, 1)
	require.Equal(t, table+"_c1_fkey", fk.Name)
}

func TestGetTableMetadata_ComposedFK(t *testing.T) {
	t.Parallel()

	table := strings.ToLower(t.Name())
	setupQ := `
DROP TABLE IF EXISTS $name;
DROP TABLE IF EXISTS $name_child;

CREATE TABLE $name_child (
	idc integer,
	c1c text,
	c2c integer,
	PRIMARY KEY (idc, c1c)
);

CREATE TABLE $name (
	id integer PRIMARY KEY,
	c1 integer,
	c2 text,
	FOREIGN KEY (c1, c2) REFERENCES $name_child
);`
	psql := setupTest(t, strings.ReplaceAll(setupQ, "$name", DefaultSchema+"."+table))
	td, err := psql.Table(context.Background(), table)
	require.NoError(t, err)

	fks := td.ForeignKeys()
	require.Len(t, fks, 1)
	fk := fks[0]
	require.Len(t, fk.ConCols, 2)
	require.Equal(t, table+"_c1_c2_fkey", fk.Name)
}

func TestGetTableMetadata_MultipleFKs(t *testing.T) {
	t.Parallel()

	table := strings.ToLower(t.Name())
	setupQ := `
DROP TABLE IF EXISTS $name;
DROP TABLE IF EXISTS $name_child1;
DROP TABLE IF EXISTS $name_child2;

CREATE TABLE $name_child1 (
	idc integer PRIMARY KEY,
	c1c text
);

CREATE TABLE $name_child2 (
	idc integer PRIMARY KEY,
	c1c text
);

CREATE TABLE $name (
	id integer PRIMARY KEY,
	c1 integer REFERENCES $name_child1,
	c2 integer REFERENCES $name_child2
);`
	psql := setupTest(t, strings.ReplaceAll(setupQ, "$name", DefaultSchema+"."+table))
	td, err := psql.Table(context.Background(), table)
	require.NoError(t, err)

	fks := td.ForeignKeys()
	require.Len(t, fks, 2)
	fk0 := fks[0]
	require.Len(t, fk0.ConCols, 1)
	require.Equal(t, table+"_c1_fkey", fk0.Name)
	fk1 := fks[1]
	require.Len(t, fk1.ConCols, 1)
	require.Equal(t, table+"_c2_fkey", fk1.Name)
}

func TestGetTableMetadata_UniqueIndexes(t *testing.T) {
	t.Parallel()

	table := strings.ToLower(t.Name())
	setupQ := `
DROP TABLE IF EXISTS $name;
CREATE TABLE $name (
    id integer,
    c1 text,
    c2 text,
	c3 text,
	c4 text,
	c5 text PRIMARY KEY,
	c6 text UNIQUE,
	deleted_at timestamp,
	UNIQUE (c3, c4)
);
CREATE UNIQUE INDEX gtd_unique_indexes_idx1 ON $name (id);
CREATE UNIQUE INDEX gtd_unique_indexes_idx2 ON $name (id, c1);
CREATE UNIQUE INDEX gtd_unique_indexes_partial_idx3 ON $name (deleted_at) WHERE (deleted_at != NULL AND c1 = c4);`
	psql := setupTest(t, strings.ReplaceAll(setupQ, "$name", DefaultSchema+"."+table))
	td, err := psql.Table(context.Background(), table)
	require.NoError(t, err)

	// check indexes
	uniqueIndexes := td.UniqueIndexes()
	require.Len(t, uniqueIndexes, 6)
	pkIndex := uniqueIndexes[0]
	require.Len(t, pkIndex.ConCols, 1)
	require.Equal(t, "c5", pkIndex.ConCols[0].Name)
	require.Equal(t, "6", pkIndex.ConColsPosition)
	require.Equal(t, table+"_pkey", pkIndex.Name)
	require.True(t, pkIndex.PK)
	require.True(t, pkIndex.Unique)
	require.False(t, pkIndex.Partial)

	idx := uniqueIndexes[1]
	require.Len(t, idx.ConCols, 1)
	require.Equal(t, "c6", idx.ConCols[0].Name)
	require.Equal(t, "7", idx.ConColsPosition)
	require.Equal(t, table+"_c6_key", idx.Name)

	idx = uniqueIndexes[2]
	require.Len(t, idx.ConCols, 2)
	require.Equal(t, "c3", idx.ConCols[0].Name)
	require.Equal(t, "c4", idx.ConCols[1].Name)
	require.Equal(t, "4 5", idx.ConColsPosition)
	require.Equal(t, table+"_c3_c4_key", idx.Name)

	idx = uniqueIndexes[3]
	require.Len(t, idx.ConCols, 1)
	require.Equal(t, "id", idx.ConCols[0].Name)
	require.Equal(t, "1", idx.ConColsPosition)
	require.Equal(t, "gtd_unique_indexes_idx1", idx.Name)

	idx = uniqueIndexes[4]
	require.Len(t, idx.ConCols, 2)
	require.Equal(t, "id", idx.ConCols[0].Name)
	require.Equal(t, "c1", idx.ConCols[1].Name)
	require.Equal(t, "1 2", idx.ConColsPosition)
	require.Equal(t, "gtd_unique_indexes_idx2", idx.Name)

	idx = uniqueIndexes[5]
	require.Len(t, idx.ConCols, 1)
	require.Equal(t, "deleted_at", idx.ConCols[0].Name)
	require.Equal(t, "8", idx.ConColsPosition)
	require.Equal(t, "gtd_unique_indexes_partial_idx3", idx.Name)

	// check columns
	col := td.ColumnByName("id")
	require.Len(t, col.UniqueIndexes, 2)
	require.Equal(t, "gtd_unique_indexes_idx1", col.UniqueIndexes[0].Name)
	require.Equal(t, "gtd_unique_indexes_idx2", col.UniqueIndexes[1].Name)

	col = td.ColumnByName("c1")
	require.Len(t, col.UniqueIndexes, 1)
	require.Equal(t, "gtd_unique_indexes_idx2", col.UniqueIndexes[0].Name)

	col = td.ColumnByName("c2")
	require.Len(t, col.UniqueIndexes, 0)

	col = td.ColumnByName("c3")
	require.Len(t, col.UniqueIndexes, 1)
	require.Equal(t, table+"_c3_c4_key", col.UniqueIndexes[0].Name)

	col = td.ColumnByName("c4")
	require.Len(t, col.UniqueIndexes, 1)
	require.Equal(t, table+"_c3_c4_key", col.UniqueIndexes[0].Name)

	col = td.ColumnByName("c5")
	require.Len(t, col.UniqueIndexes, 1)
	require.Equal(t, table+"_pkey", col.UniqueIndexes[0].Name)

	col = td.ColumnByName("c6")
	require.Len(t, col.UniqueIndexes, 1)
	require.Equal(t, table+"_c6_key", col.UniqueIndexes[0].Name)

	col = td.ColumnByName("deleted_at")
	require.Len(t, col.UniqueIndexes, 1)
	require.Equal(t, "gtd_unique_indexes_partial_idx3", col.UniqueIndexes[0].Name)
}

func TestGetTableMetadata_CheckConstraints(t *testing.T) {
	t.Parallel()

	table := strings.ToLower(t.Name())
	setupQ := `
DROP TABLE IF EXISTS $name;
CREATE TABLE $name (
    c1 numeric CHECK (c1 > 0),
    c2 numeric,
    c3 numeric,
    CHECK (c2 > c1 AND c3 > c2)
);`
	psql := setupTest(t, strings.ReplaceAll(setupQ, "$name", DefaultSchema+"."+table))
	td, err := psql.Table(context.Background(), table)
	require.NoError(t, err)

	cons := td.CheckConstraints()
	require.Len(t, cons, 2)
	require.Equal(t, table+"_c1_check", cons[0].Name)
	require.Equal(t, table+"_check", cons[1].Name)

	col := td.ColumnByName("c1")
	require.Len(t, col.CheckConstraints, 2)
	require.Equal(t, table+"_c1_check", col.CheckConstraints[0].Name)
	require.Equal(t, table+"_check", col.CheckConstraints[1].Name)

	col = td.ColumnByName("c2")
	require.Len(t, col.CheckConstraints, 1)
	require.Equal(t, table+"_check", col.CheckConstraints[0].Name)

	col = td.ColumnByName("c3")
	require.Len(t, col.CheckConstraints, 1)
	require.Equal(t, table+"_check", col.CheckConstraints[0].Name)
}

func TestGetCustomTypesEnum(t *testing.T) {
	t.Parallel()

	table := strings.ToLower(t.Name())
	setupQ := `
DROP TABLE IF EXISTS $name;
DROP TYPE IF EXISTS "td_custom_enum1";
DROP TYPE IF EXISTS "td_custom_enum2";
CREATE TYPE "td_custom_enum1" AS ENUM ('LBL00', 'LBL01', 'LBL02');
CREATE TYPE "td_custom_enum2" AS ENUM ('LBL10', 'LBL11');
CREATE TABLE $name (
    c1 td_custom_enum1,
	c2 td_custom_enum2,
	c3 text
);`
	psql := setupTest(t, strings.ReplaceAll(setupQ, "$name", DefaultSchema+"."+table))
	ctx := context.Background()
	err := psql.getCustomTypes(ctx, DefaultSchema)
	columns, err := psql.getColumns(ctx, TableName{schema: DefaultSchema, name: table})
	require.NoError(t, err)
	require.Len(t, columns, 3)

	col := columns[0]
	require.Equal(t, "c1", col.Name)
	require.Equal(t, "enum", col.TypeName)
	require.Equal(t, "td_custom_enum1", col.UserDefinedTypeName)
	require.Equal(t, col.TypeOID, col.TypeDef.OID)
	subType, ok := col.TypeDef.SubType.(typeEnum)
	require.True(t, ok)
	require.Equal(t, []string{"LBL00", "LBL01", "LBL02"}, subType.Labels)

	col = columns[1]
	require.Equal(t, "c2", col.Name)
	require.Equal(t, "enum", col.TypeName)
	require.Equal(t, "td_custom_enum2", col.UserDefinedTypeName)
	require.Equal(t, col.TypeOID, col.TypeDef.OID)
	subType, ok = col.TypeDef.SubType.(typeEnum)
	require.True(t, ok)
	require.Equal(t, []string{"LBL10", "LBL11"}, subType.Labels)

	col = columns[2]
	require.Equal(t, "c3", col.Name)
	require.Equal(t, "text", col.TypeName)
}

func TestConstraints_SelfRefFK(t *testing.T) {
	t.Parallel()

	table := strings.ToLower(t.Name())
	psql := setupTest(t, strings.ReplaceAll(`
DROP TABLE IF EXISTS $name;
CREATE TABLE $name (
	id integer PRIMARY KEY,
	c1 integer REFERENCES $name
);`, "$name", DefaultSchema+"."+table))

	cons, err := psql.getTableConstraints(context.Background(), TableName{schema: DefaultSchema, name: table})
	require.NoError(t, err)
	slices.SortFunc(cons, func(a, b *ConstraintMeta) int {
		if a.Name < b.Name {
			return -1
		}
		return 1
	})

	expectCon := &ConstraintMeta{
		Name:                table + "_c1_fkey",
		ConstraintExpr:      "FOREIGN KEY (c1) REFERENCES " + table + "(id)",
		Type:                "f",
		FKReferencedColumns: SliceInt{1},
		ConstrainedColumns:  SliceInt{2},
	}

	for _, con := range cons {
		if con.Type != "f" {
			continue
		}
		require.NotZero(t, con.OID)
		require.NotZero(t, con.IndexOID)
		require.Equal(t, expectCon.Name, con.Name)
		require.Equal(t, expectCon.ConstraintExpr, con.ConstraintExpr)
		require.Equal(t, expectCon.Type, con.Type)
		require.NotZero(t, con.FKTargetTableOID)
		require.Equal(t, expectCon.FKReferencedColumns, con.FKReferencedColumns)
		require.Equal(t, expectCon.ConstrainedColumns, con.ConstrainedColumns)

		return
	}

	t.Fatal("fk constraint not found")
}

func TestConstraints_PKsAreFKs(t *testing.T) {
	t.Parallel()

	table := strings.ToLower(t.Name())
	psql := setupTest(t, strings.ReplaceAll(`
DROP TABLE IF EXISTS $name;
DROP TABLE IF EXISTS $name_child1;
DROP TABLE IF EXISTS $name_child2;

CREATE TABLE $name_child1 (
    id integer PRIMARY KEY,
    c1 text
);

CREATE TABLE $name_child2 (
    id integer PRIMARY KEY,
    c1 text
);

CREATE TABLE $name (
    id integer REFERENCES $name_child1,
    c1 integer REFERENCES $name_child2,
    c2 text,
    PRIMARY KEY (id, c1)
);`, "$name", DefaultSchema+"."+table))

	cons, err := psql.getTableConstraints(context.Background(), TableName{schema: DefaultSchema, name: table})
	require.NoError(t, err)
	slices.SortFunc(cons, func(a, b *ConstraintMeta) int {
		if a.Name < b.Name {
			return -1
		}
		return 1
	})

	expectCons := []*ConstraintMeta{
		{
			Name:                table + "_c1_fkey",
			ConstraintExpr:      "FOREIGN KEY (c1) REFERENCES " + table + "_child2(id)",
			Type:                "f",
			FKReferencedColumns: SliceInt{1},
			ConstrainedColumns:  SliceInt{2},
		},
		{
			Name:                table + "_id_fkey",
			ConstraintExpr:      "FOREIGN KEY (id) REFERENCES " + table + "_child1(id)",
			Type:                "f",
			FKReferencedColumns: SliceInt{1},
			ConstrainedColumns:  SliceInt{1},
		},
		{
			Name:               table + "_pkey",
			ConstraintExpr:     "PRIMARY KEY (id, c1)",
			Type:               "p",
			ConstrainedColumns: SliceInt{1, 2},
		},
	}

	expectConsI := 0
	for _, con := range cons {
		expectCon := expectCons[expectConsI]
		require.NotZero(t, con.OID)
		require.NotZero(t, con.IndexOID)
		require.Equal(t, expectCon.Name, con.Name)
		require.Equal(t, expectCon.ConstraintExpr, con.ConstraintExpr)
		require.Equal(t, expectCon.Type, con.Type)
		if con.Type == "f" {
			require.NotZero(t, con.FKTargetTableOID)
			require.Equal(t, expectCon.FKReferencedColumns, con.FKReferencedColumns)
		}
		require.Equal(t, expectCon.ConstrainedColumns, con.ConstrainedColumns)

		expectConsI++
	}
	if expectConsI != len(expectCons) {
		t.Fatal("some constraints not found")
	}
}

func setupTest(t *testing.T, q string) *Inspector {
	t.Helper()

	insp := NewInspector(getDB(t))
	if q != "" {
		exec(t, insp.db, q)
	}

	return insp
}

func getDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_DB"),
	)
	db, err := sql.Open(DriverName, dsn)
	require.NoError(t, err)
	return db
}

func exec(t *testing.T, db *sqlx.DB, q string) {
	t.Helper()
	_, err := db.Exec(q)
	require.NoError(t, err)
}

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../.env"); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}
