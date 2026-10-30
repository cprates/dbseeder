package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	testsSet := []struct {
		description string
		input       Seeder
		expectCfg   Seeder
		expectErr   string
	}{
		{
			description: "no configured tables error",
			input:       Seeder{},
			expectErr:   "no tables configured",
		},
		{
			description: "validates tables",
			input: Seeder{
				Tables: []*Table{{Name: "public.t1", MaxRows: -1}},
			},
			expectErr: "table public.t1: MaxRows must be >= 0",
		},
		{
			description: "default values are correct",
			input: Seeder{
				Tables: []*Table{{Name: "public.t1"}},
			},
			expectCfg: Seeder{
				Schema:     defaultConfig.Schema,
				MaxRows:    defaultConfig.MaxRows,
				MaxRetries: defaultConfig.MaxRetries,
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
			input:       Table{Name: "public.table1", MaxRows: -1},
			expectErr:   "MaxRows must be >= 0",
		},
		{
			description: "invalid MaxRetries errors",
			input:       Table{Name: "public.table1", MaxRetries: -1},
			expectErr:   "MaxRetries must be >= 0",
		},
		{
			description: "no defaults when values are provided",
			input: Table{
				Name:       "public.table1",
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
