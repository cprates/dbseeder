package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cprates/dbseeder/internal/ir"
	"github.com/cprates/dbseeder/internal/postgres"
)

func TestParseQueryWithoutExpressionEvaluatesToTrue(t *testing.T) {
	q := "SELECT * FROM t;"
	expr, err := postgres.ParseTest(context.Background(), q)
	require.NoError(t, err)
	expectedExpr := ir.OpNoOp{}
	require.Equal(t, expectedExpr, expr)
}

func TestParseIsOper(t *testing.T) {
	q := "SELECT * FROM t WHERE t.c1 IS NULL;"
	expr, err := postgres.ParseTest(context.Background(), q)
	require.NoError(t, err)
	expectedExpr := ir.Equal(
		ir.NewOperandColumn("", "t", "c1"),
		ir.OperandNil{},
	)
	require.Equal(t, expectedExpr, expr)
}

func TestParseIsNotOper(t *testing.T) {
	q := "SELECT * FROM t WHERE t.c1 IS NOT NULL;"
	expr, err := postgres.ParseTest(context.Background(), q)
	require.NoError(t, err)
	expectedExpr := ir.NotEqual(
		ir.NewOperandColumn("", "t", "c1"),
		ir.OperandNil{},
	)
	require.Equal(t, expectedExpr, expr)
}

func TestParseEqualOnConstant(t *testing.T) {
	q := "SELECT * FROM t WHERE c1 = 42;"
	expr, err := postgres.ParseTest(context.Background(), q)
	require.NoError(t, err)
	expectedExpr := ir.Equal(
		ir.NewOperandColumn("", "", "c1"),
		ir.NewOperandConstant("42"),
	)
	require.Equal(t, expectedExpr, expr)
}

func TestParseEqualOnString(t *testing.T) {
	q := "SELECT * FROM t WHERE t.c1 = 'dummy';"
	expr, err := postgres.ParseTest(context.Background(), q)
	require.NoError(t, err)
	expectedExpr := ir.Equal(
		ir.NewOperandColumn("", "t", "c1"),
		ir.NewOperandConstant("dummy"),
	)
	require.Equal(t, expectedExpr, expr)
}

func TestParseNotEqual(t *testing.T) {
	queries := []string{
		"SELECT * FROM t WHERE c1 <> 42;",
		"SELECT * FROM t WHERE c1 != 42;",
	}
	for _, q := range queries {
		expr, err := postgres.ParseTest(context.Background(), q)
		require.NoError(t, err, q)
		expectedExpr := ir.NotEqual(
			ir.NewOperandColumn("", "", "c1"),
			ir.NewOperandConstant("42"),
		)
		require.Equal(t, expectedExpr, expr)
	}
}

func TestParseBooleanPrecedenceIsRespected(t *testing.T) {
	q := "SELECT * FROM t WHERE (c1 IS NULL) AND (c1 IS NOT NULL) OR (c1 IS NOT NULL)"
	expr, err := postgres.ParseTest(context.Background(), q)
	require.NoError(t, err)
	expectedExpr := ir.Or(
		ir.And(
			ir.Equal(ir.NewOperandColumn("", "", "c1"), ir.OperandNil{}),
			ir.NotEqual(ir.NewOperandColumn("", "", "c1"), ir.OperandNil{}),
		),
		ir.NotEqual(ir.NewOperandColumn("", "", "c1"), ir.OperandNil{}),
	)
	require.Equal(t, expectedExpr, expr)

	q = "SELECT * FROM t WHERE ((c1 IS NULL) AND (c1 IS NOT NULL)) OR (c1 IS NOT NULL)"
	expr, err = postgres.ParseTest(context.Background(), q)
	require.NoError(t, err)
	expectedExpr = ir.Or(
		ir.And(
			ir.Equal(ir.NewOperandColumn("", "", "c1"), ir.OperandNil{}),
			ir.NotEqual(ir.NewOperandColumn("", "", "c1"), ir.OperandNil{}),
		),
		ir.NotEqual(ir.NewOperandColumn("", "", "c1"), ir.OperandNil{}),
	)
	require.Equal(t, expectedExpr, expr)

	q = "SELECT * FROM t WHERE (c1 IS NULL) AND ((c1 IS NOT NULL) OR (c1 IS NOT NULL))"
	expr, err = postgres.ParseTest(context.Background(), q)
	expectedExpr = ir.And(
		ir.Equal(ir.NewOperandColumn("", "", "c1"), ir.OperandNil{}),
		ir.Or(
			ir.NotEqual(ir.NewOperandColumn("", "", "c1"), ir.OperandNil{}),
			ir.NotEqual(ir.NewOperandColumn("", "", "c1"), ir.OperandNil{}),
		),
	)
	require.Equal(t, expectedExpr, expr)
}

func TestParseComparisonBetweenColumns(t *testing.T) {
	q := "SELECT * FROM t WHERE c1 = c11 AND c2 = c22 AND c3 != c33 OR c4 = c3;"
	expr, err := postgres.ParseTest(context.Background(), q)
	require.NoError(t, err)
	expectedExpr := ir.Or(
		ir.And(
			ir.And(
				ir.Equal(ir.NewOperandColumn("", "", "c1"), ir.NewOperandColumn("", "", "c11")),
				ir.Equal(ir.NewOperandColumn("", "", "c2"), ir.NewOperandColumn("", "", "c22")),
			),
			ir.NotEqual(ir.NewOperandColumn("", "", "c3"), ir.NewOperandColumn("", "", "c33")),
		),
		ir.Equal(ir.NewOperandColumn("", "", "c4"), ir.NewOperandColumn("", "", "c3")),
	)
	require.Equal(t, expectedExpr, expr)
}

func TestParseBooleanPrecedenceIsRespected2(t *testing.T) {
	// no parentheses: F and T or T => T
	q := "SELECT * FROM t WHERE (c1 IS NULL) AND (c1 IS NOT NULL) OR (c1 IS NOT NULL)"
	expr, err := postgres.ParseTest(context.Background(), q)
	require.NoError(t, err)
	expectedExpr := ir.Or(
		ir.And(
			ir.Equal(ir.NewOperandColumn("", "", "c1"), ir.OperandNil{}),
			ir.NotEqual(ir.NewOperandColumn("", "", "c1"), ir.OperandNil{}),
		),
		ir.NotEqual(ir.NewOperandColumn("", "", "c1"), ir.OperandNil{}),
	)
	require.Equal(t, expectedExpr, expr)
}
