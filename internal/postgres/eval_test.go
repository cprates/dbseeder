package postgres_test

import (
	"cmp"
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cprates/dbseeder/internal/ir"
	"github.com/cprates/dbseeder/internal/postgres"
)

func TestEvalTypesCmp(t *testing.T) {
	testsSet := []struct {
		query  string
		row    ir.Row
		expect bool
	}{
		{
			query:  "SELECT * FROM t WHERE t.c1 > 0",
			row:    ir.Row{"c1": typeString("42")},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 > 0",
			row:    ir.Row{"c1": typeInt(42)},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 > 42",
			row:    ir.Row{"c1": typeString("42")},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 > 43",
			row:    ir.Row{"c1": typeString("42")},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE 0 > t.c1",
			row:    ir.Row{"c1": typeInt(42)},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE -1 > t.c1",
			row:    ir.Row{"c1": typeInt(-42)},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 > -3",
			row:    ir.Row{"c1": typeInt(-42)},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 > t.c2",
			row:    ir.Row{"c1": typeInt(42), "c2": typeInt(42)},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 > t.c2",
			row:    ir.Row{"c1": typeString("42"), "c2": typeInt(42)},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 > t.c2",
			row:    ir.Row{"c1": typeString("43"), "c2": typeInt(42)},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 > t.c2",
			row:    ir.Row{"c1": typeString("43"), "c2": nil},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 > t.c2",
			row:    ir.Row{"c1": nil, "c2": typeString("43")},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE 42 > 42",
			row:    ir.Row{},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE 42 > 4",
			row:    ir.Row{},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE 4 > 42",
			row:    ir.Row{},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 > NULL",
			row:    ir.Row{"c1": nil},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE 42 > NULL",
			row:    ir.Row{},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE NULL > 42",
			row:    ir.Row{},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE NULL > t.c1",
			row:    ir.Row{"c1": nil},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 > NULL",
			row:    ir.Row{"c1": nil},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE NULL = NULL",
			row:    ir.Row{},
			expect: true,
		},
	}

	for _, test := range testsSet {
		t.Run(test.query, func(t *testing.T) {
			expr, err := postgres.ParseTest(context.Background(), test.query)
			require.NoError(t, err)
			match := expr.Eval(map[string]ir.Row{"t": test.row})
			require.Equal(t, test.expect, match)
		})
	}
}

func TestEvalOperatorLT(t *testing.T) {
	testsSet := []struct {
		query  string
		row    ir.Row
		expect bool
	}{
		{
			query:  "SELECT * FROM t WHERE t.c1 < 0",
			row:    ir.Row{"c1": typeString("42")},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 < 42",
			row:    ir.Row{"c1": typeString("42")},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE 0 < '42'",
			row:    ir.Row{},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE 0 < 42",
			row:    ir.Row{},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE 0 < 42",
			row:    ir.Row{},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE NULL < 42",
			row:    ir.Row{},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE 42 < NULL",
			row:    ir.Row{},
			expect: false,
		},
	}

	for _, test := range testsSet {
		t.Run(test.query, func(t *testing.T) {
			expr, err := postgres.ParseTest(context.Background(), test.query)
			require.NoError(t, err)
			match := expr.Eval(map[string]ir.Row{"t": test.row})
			require.Equal(t, test.expect, match)
		})
	}
}

func TestEvalOperatorLE(t *testing.T) {
	testsSet := []struct {
		query  string
		row    ir.Row
		expect bool
	}{
		{
			query:  "SELECT * FROM t WHERE t.c1 <= 0",
			row:    ir.Row{"c1": typeString("42")},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 <= 42",
			row:    ir.Row{"c1": typeString("42")},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE 0 <= '42'",
			row:    ir.Row{},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE 0 <= 42",
			row:    ir.Row{},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE 0 <= 42",
			row:    ir.Row{},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE NULL <= 42",
			row:    ir.Row{},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE 42 <= NULL",
			row:    ir.Row{},
			expect: false,
		},
	}

	for _, test := range testsSet {
		t.Run(test.query, func(t *testing.T) {
			expr, err := postgres.ParseTest(context.Background(), test.query)
			require.NoError(t, err)
			match := expr.Eval(map[string]ir.Row{"t": test.row})
			require.Equal(t, test.expect, match)
		})
	}
}

func TestEvalOperatorGT(t *testing.T) {
	testsSet := []struct {
		query  string
		row    ir.Row
		expect bool
	}{
		{
			query:  "SELECT * FROM t WHERE t.c1 > 0",
			row:    ir.Row{"c1": typeString("42")},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 > 42",
			row:    ir.Row{"c1": typeString("42")},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE 0 > '42'",
			row:    ir.Row{},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE 0 > 42",
			row:    ir.Row{},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE 0 > 42",
			row:    ir.Row{},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE NULL > 42",
			row:    ir.Row{},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE 42 > NULL",
			row:    ir.Row{},
			expect: true,
		},
	}

	for _, test := range testsSet {
		t.Run(test.query, func(t *testing.T) {
			expr, err := postgres.ParseTest(context.Background(), test.query)
			require.NoError(t, err)
			match := expr.Eval(map[string]ir.Row{"t": test.row})
			require.Equal(t, test.expect, match)
		})
	}
}

func TestEvalOperatorGE(t *testing.T) {
	testsSet := []struct {
		query  string
		row    ir.Row
		expect bool
	}{
		{
			query:  "SELECT * FROM t WHERE t.c1 >= 0",
			row:    ir.Row{"c1": typeString("42")},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 >= 42",
			row:    ir.Row{"c1": typeString("42")},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE 0 >= '42'",
			row:    ir.Row{},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE 0 >= 42",
			row:    ir.Row{},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE 0 >= 42",
			row:    ir.Row{},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE NULL >= 42",
			row:    ir.Row{},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE 42 >= NULL",
			row:    ir.Row{},
			expect: true,
		},
	}

	for _, test := range testsSet {
		t.Run(test.query, func(t *testing.T) {
			expr, err := postgres.ParseTest(context.Background(), test.query)
			require.NoError(t, err)
			match := expr.Eval(map[string]ir.Row{"t": test.row})
			require.Equal(t, test.expect, match)
		})
	}
}

func TestEvalOperatorIn(t *testing.T) {
	testsSet := []struct {
		query  string
		row    ir.Row
		expect bool
	}{
		{
			query:  "SELECT * FROM t WHERE t.c1 IN (1, 2, 3)",
			row:    ir.Row{"c1": typeInt(42)},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE (t.c1, t.c2) IN ((1, 2), (42, 24))",
			row:    ir.Row{"c1": typeInt(42), "c2": typeInt(7)},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 IN (1, 2, 42)",
			row:    ir.Row{"c1": typeInt(42)},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE (t.c1, t.c2) IN ((1, 2), (42, 24))",
			row:    ir.Row{"c1": typeInt(42), "c2": typeInt(24)},
			expect: true,
		},
	}

	for _, test := range testsSet {
		t.Run(test.query, func(t *testing.T) {
			expr, err := postgres.ParseTest(context.Background(), test.query)
			require.NoError(t, err)
			match := expr.Eval(map[string]ir.Row{"t": test.row})
			require.Equal(t, test.expect, match)
		})
	}
}

func TestEvalOperatorNotIn(t *testing.T) {
	testsSet := []struct {
		query  string
		row    ir.Row
		expect bool
	}{
		{
			query:  "SELECT * FROM t WHERE t.c1 NOT IN (1, 2, 3)",
			row:    ir.Row{"c1": typeInt(42)},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE (t.c1, t.c2) NOT IN ((1, 2), (42, 24))",
			row:    ir.Row{"c1": typeInt(42), "c2": typeInt(7)},
			expect: true,
		},
		{
			query:  "SELECT * FROM t WHERE t.c1 NOT IN (1, 2, 42)",
			row:    ir.Row{"c1": typeInt(42)},
			expect: false,
		},
		{
			query:  "SELECT * FROM t WHERE (t.c1, t.c2) NOT IN ((1, 2), (42, 24))",
			row:    ir.Row{"c1": typeInt(42), "c2": typeInt(24)},
			expect: false,
		},
	}

	for _, test := range testsSet {
		t.Run(test.query, func(t *testing.T) {
			expr, err := postgres.ParseTest(context.Background(), test.query)
			require.NoError(t, err)
			match := expr.Eval(map[string]ir.Row{"t": test.row})
			require.Equal(t, test.expect, match)
		})
	}
}

type typeInt int

func (c typeInt) Val() any {
	return c
}

func typeIntToInt(v any) int {
	switch vT := v.(type) {
	case typeInt:
		return int(vT)
	case string:
		i, _ := strconv.ParseInt(vT, 10, 64)
		return int(i)
	default:
		panic(fmt.Errorf("unexpected type: %T", v))
	}
}

func (c typeInt) Cmp(l, r any) int {
	if l != nil && r != nil {

		return cmp.Compare(typeIntToInt(l), typeIntToInt(r))
	}

	if l == nil {
		return -1
	}
	if r == nil {
		return 1
	}

	return 0
}

type typeString string

func (c typeString) Val() any {
	return c
}

func typeStringToString(v any) string {
	switch vT := v.(type) {
	case typeString:
		return string(vT)
	case string:
		return vT
	case typeInt:
		return strconv.FormatInt(int64(vT), 10)
	default:
		panic(fmt.Errorf("unexpected type: %T", v))
	}
}

func (c typeString) Cmp(l, r any) int {
	if l != nil && r != nil {
		return cmp.Compare(typeStringToString(l), typeStringToString(r))
	}

	if l == nil {
		return -1
	}
	if r == nil {
		return 1
	}

	return 0
}

func TestEvalBooleanPrecedenceIsRespected(t *testing.T) {
	// no parentheses: F and T or T => T
	q := "SELECT * FROM t WHERE (t.c1 IS NULL) AND (t.c1 IS NOT NULL) OR (t.c1 IS NOT NULL)"
	expr, err := postgres.ParseTest(context.Background(), q)
	require.NoError(t, err)
	match := expr.Eval(map[string]ir.Row{"t": {"c1": typeInt(42)}})
	require.True(t, match)

	// parentheses that do not change precedence: (F and T) or T => T
	q = "SELECT * FROM t WHERE ((t.c1 IS NULL) AND (t.c1 IS NOT NULL)) OR (t.c1 IS NOT NULL)"
	expr, err = postgres.ParseTest(context.Background(), q)
	require.NoError(t, err)
	match = expr.Eval(map[string]ir.Row{"t": {"c1": typeInt(42)}})
	require.True(t, match)

	// parentheses that change precedence: F and (T or T) => F
	q = "SELECT * FROM t WHERE (t.c1 IS NULL) AND ((t.c1 IS NOT NULL) OR (t.c1 IS NOT NULL))"
	expr, err = postgres.ParseTest(context.Background(), q)
	match = expr.Eval(map[string]ir.Row{"t": {"c1": typeInt(42)}})
	require.False(t, match)
}
