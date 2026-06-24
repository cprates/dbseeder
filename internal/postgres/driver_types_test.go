package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cprates/dbseeder/internal/postgres"
)

func TestBoolValue(t *testing.T) {
	b := postgres.Bool(true)
	v, err := b.Value()
	require.NoError(t, err)
	require.Equal(t, postgres.Bool(true), v)

	b = false
	v, err = b.Value()
	require.NoError(t, err)
	require.Equal(t, postgres.Bool(false), v)
}

func TestBoolScan(t *testing.T) {
	testsSet := []struct {
		description  string
		base         postgres.Bool
		input        any
		expectErr    string
		expectOutput postgres.Bool
	}{
		{
			description: "invalid type input errors",
			base:        postgres.Bool(false),
			input:       42,
			expectErr:   "expect 'string', got int",
		},
		{
			description: "malformed input errors",
			base:        postgres.Bool(false),
			input:       "this",
			expectErr:   `expect 'yes' or 'no', got "this"`,
		},
		{
			description:  "lowercase no",
			base:         postgres.Bool(true),
			input:        "no",
			expectOutput: postgres.Bool(false),
		},
		{
			description:  "uppercase no",
			base:         postgres.Bool(true),
			input:        "NO",
			expectOutput: postgres.Bool(false),
		},
		{
			description:  "lowercase yes",
			base:         postgres.Bool(false),
			input:        "yes",
			expectOutput: postgres.Bool(true),
		},
		{
			description:  "uppercase yes",
			base:         postgres.Bool(false),
			input:        "YES",
			expectOutput: postgres.Bool(true),
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			err := test.base.Scan(test.input)
			if test.expectErr != "" {
				require.ErrorContains(t, err, test.expectErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectOutput, test.base)
		})
	}
}

func TestSliceIntValue(t *testing.T) {
	s := postgres.SliceInt{1, 2, 3}
	v, err := s.Value()
	require.NoError(t, err)
	require.Equal(t, postgres.SliceInt{1, 2, 3}, v)
}

func TestSliceIntScan(t *testing.T) {
	testsSet := []struct {
		description  string
		input        any
		expectErr    string
		expectOutput postgres.SliceInt
	}{
		{
			description: "invalid type input errors",
			input:       42,
			expectErr:   "expect 'string', got int",
		},
		{
			description: "malformed input errors",
			input:       "{1, 2, 3,",
			expectErr:   "malformed list: {1, 2, 3,",
		},
		{
			description: "invalid json input errors",
			input:       "{1, 2, 3,}",
			expectErr:   `marshalling "[1, 2, 3,]"`,
		},
		{
			description:  "nil input returns empty SliceInt",
			input:        nil,
			expectOutput: postgres.SliceInt{},
		},
		{
			description:  "empty string input returns empty SliceInt",
			input:        "",
			expectOutput: postgres.SliceInt{},
		},
		{
			description:  "happy path",
			input:        "{1,2}",
			expectOutput: postgres.SliceInt{1, 2},
		},
	}

	for _, test := range testsSet {
		t.Run(test.description, func(t *testing.T) {
			s := postgres.SliceInt{}
			err := s.Scan(test.input)
			if test.expectErr != "" {
				require.ErrorContains(t, err, test.expectErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectOutput, s)
		})
	}
}
