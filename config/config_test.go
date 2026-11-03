package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	bTrue := true
	testsSet := []struct {
		description string
		input       Seeder
		expectCfg   Seeder
		expectErr   string
	}{
		{
			description: "no configured tables succeed",
			input:       Seeder{},
			expectCfg: Seeder{
				Schema:     defaultConfig.Schema,
				MaxRows:    defaultConfig.MaxRows,
				MaxRetries: defaultConfig.MaxRetries,
				SeedAll:    &bTrue,
				OutputPath: defaultConfig.OutputPath,
			},
		},
		{
			description: "validates tables",
			input: Seeder{
				Schema: "public",
				Tables: []*Table{{Name: "t1", MaxRows: -1}},
			},
			expectErr: "table t1: MaxRows must be >= 0",
		},
		{
			description: "default values are correct",
			input: Seeder{
				Schema: "public",
				Tables: []*Table{{Name: "t1"}},
			},
			expectCfg: Seeder{
				Schema:     defaultConfig.Schema,
				MaxRows:    defaultConfig.MaxRows,
				MaxRetries: defaultConfig.MaxRetries,
				SeedAll:    &bTrue,
				Tables: []*Table{{
					Name:       "public.t1",
					MaxRows:    defaultConfig.MaxRows,
					MaxRetries: defaultConfig.MaxRetries,
				}},
				OutputPath: defaultConfig.OutputPath,
			},
		},
	}
	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			err := test.input.Validate()
			if test.expectErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expectCfg, test.input)
		})
	}
}

func TestTableConfig(t *testing.T) {
	t.Parallel()

	testsSet := []struct {
		description string
		input       Table
		expectCfg   Table
		expectErr   string
	}{
		{
			description: "invalid MaxRows errors",
			input:       Table{Name: "table1", MaxRows: -1},
			expectErr:   "MaxRows must be >= 0",
		},
		{
			description: "invalid MaxRetries errors",
			input:       Table{Name: "table1", MaxRetries: -1},
			expectErr:   "MaxRetries must be >= 0",
		},
		{
			description: "no defaults when values are provided",
			input: Table{
				Name:       "table1",
				MaxRows:    1,
				MaxRetries: 2,
			},
			expectCfg: Table{
				Name:       "public.table1",
				MaxRows:    1,
				MaxRetries: 2,
			},
		},
	}
	cfg := &Seeder{
		Schema:     "public",
		MaxRows:    defaultConfig.MaxRows,
		MaxRetries: defaultConfig.MaxRetries,
	}
	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			err := test.input.Validate(cfg)
			if test.expectErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expectCfg, test.input)
		})
	}
}

func TestSeedAllTables(t *testing.T) {
	t.Parallel()

	t.Run("nil SeedAll defaults to true", func(t *testing.T) {
		s := &Seeder{}
		require.True(t, s.SeedAllTables())
	})

	t.Run("false SeedAll returns false", func(t *testing.T) {
		bSeedAll := false
		s := &Seeder{SeedAll: &bSeedAll}
		require.NoError(t, s.Validate())
		require.False(t, s.SeedAllTables())
	})

	t.Run("true SeedAll returns true", func(t *testing.T) {
		bSeedAll := true
		s := &Seeder{SeedAll: &bSeedAll}
		require.NoError(t, s.Validate())
		require.True(t, s.SeedAllTables())
	})
}

func TestSimpleName(t *testing.T) {
	t.Parallel()

	tbl := &Table{Name: "schema.name"}
	require.Equal(t, "name", tbl.SimpleName())
}

func TestGetTableCfg(t *testing.T) {
	t.Parallel()

	cdr := &Seeder{Tables: []*Table{
		{Name: "schema.name1"},
		{Name: "schema.name2"},
	}}
	require.Nil(t, cdr.GetTableCfg("schema.namex"))
	require.Equal(t, &Table{Name: "schema.name1"}, cdr.GetTableCfg("schema.name1"))
	require.Equal(t, &Table{Name: "schema.name2"}, cdr.GetTableCfg("schema.name2"))
}

func TestAddDefaultTable(t *testing.T) {
	t.Parallel()

	cdr := &Seeder{}
	require.NoError(t, cdr.Validate())
	cdr.AddDefaultTable("table1")
	cfg := cdr.GetTableCfg("public.table1")
	expectCfg := &Table{
		Name:       "public.table1",
		MaxRows:    defaultConfig.MaxRows,
		MaxRetries: defaultConfig.MaxRetries,
	}
	require.Equal(t, expectCfg, cfg)
}
