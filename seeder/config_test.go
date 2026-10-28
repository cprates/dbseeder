package seeder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	testsSet := []struct {
		description string
		input       Config
		expectCfg   Config
		expectErr   string
	}{
		{
			description: "no configured tables error",
			input:       Config{},
			expectErr:   "no tables configured",
		},
		{
			description: "validates tables",
			input: Config{
				Tables: []*TableConfig{{Name: "public.t1", MaxRows: -1}},
			},
			expectErr: "table public.t1: MaxRows must be >= 0",
		},
		{
			description: "default values are correct",
			input: Config{
				Tables: []*TableConfig{{Name: "public.t1"}},
			},
			expectCfg: Config{
				Schema:     defaultConfig.Schema,
				MaxRows:    defaultConfig.MaxRows,
				MaxRetries: defaultConfig.MaxRetries,
				Tables: []*TableConfig{{
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
		input       TableConfig
		expectCfg   TableConfig
		expectErr   string
	}{
		{
			description: "invalid MaxRows errors",
			input:       TableConfig{Name: "public.table1", MaxRows: -1},
			expectErr:   "MaxRows must be >= 0",
		},
		{
			description: "invalid MaxRetries errors",
			input:       TableConfig{Name: "public.table1", MaxRetries: -1},
			expectErr:   "MaxRetries must be >= 0",
		},
		{
			description: "no defaults when values are provided",
			input: TableConfig{
				Name:       "public.table1",
				MaxRows:    1,
				MaxRetries: 2,
			},
			expectCfg: TableConfig{
				Name:       "public.table1",
				MaxRows:    1,
				MaxRetries: 2,
			},
		},
	}
	cfg := &Config{
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
