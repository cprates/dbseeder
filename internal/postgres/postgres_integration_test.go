//go:build integration_tests

package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cprates/dbseeder/internal/ir"
)

func TestJoinsCondition(t *testing.T) {
	t.Parallel()

	setupQ := `
DROP TABLE IF EXISTS conditional_joins_1;
CREATE TABLE conditional_joins_1 (
	id integer PRIMARY KEY,
	c1 integer,
	c2 integer
);
DROP TABLE IF EXISTS conditional_joins_2;
CREATE TABLE conditional_joins_2 (
	id integer PRIMARY KEY,
	c1 integer,
	c3 integer
);
`
	t.Run("condition equal", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := "SELECT * FROM conditional_joins_1 as cj1 JOIN conditional_joins_2 cj2 ON cj1.c1 = cj2.c1;"
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.Equal(
			ir.NewOperandColumn("public", "conditional_joins_1", "c1"),
			ir.NewOperandColumn("public", "conditional_joins_2", "c1"),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("condition not equal", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := "SELECT * FROM conditional_joins_1 as cj1 JOIN conditional_joins_2 cj2 ON cj1.c1 <> cj2.c1;"
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.NotEqual(
			ir.NewOperandColumn("public", "conditional_joins_1", "c1"),
			ir.NewOperandColumn("public", "conditional_joins_2", "c1"),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("condition lt", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := "SELECT * FROM conditional_joins_1 as cj1 JOIN conditional_joins_2 cj2 ON cj1.c1 < cj2.c1;"
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.LT(
			ir.NewOperandColumn("public", "conditional_joins_1", "c1"),
			ir.NewOperandColumn("public", "conditional_joins_2", "c1"),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("condition le", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := "SELECT * FROM conditional_joins_1 as cj1 JOIN conditional_joins_2 cj2 ON cj1.c1 <= cj2.c1;"
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.LE(
			ir.NewOperandColumn("public", "conditional_joins_1", "c1"),
			ir.NewOperandColumn("public", "conditional_joins_2", "c1"),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("condition gt", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := "SELECT * FROM conditional_joins_1 as cj1 JOIN conditional_joins_2 cj2 ON cj1.c1 > cj2.c1;"
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.GT(
			ir.NewOperandColumn("public", "conditional_joins_1", "c1"),
			ir.NewOperandColumn("public", "conditional_joins_2", "c1"),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("condition ge", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := "SELECT * FROM conditional_joins_1 as cj1 JOIN conditional_joins_2 cj2 ON cj1.c1 >= cj2.c1;"
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.GE(
			ir.NewOperandColumn("public", "conditional_joins_1", "c1"),
			ir.NewOperandColumn("public", "conditional_joins_2", "c1"),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("condition with and", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := "SELECT * FROM conditional_joins_1 as cj1 JOIN conditional_joins_2 cj2 ON cj1.c1 = cj2.c1 AND cj1.c2 = cj2.c3;"
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.And(
			ir.Equal(
				ir.NewOperandColumn("public", "conditional_joins_1", "c1"),
				ir.NewOperandColumn("public", "conditional_joins_2", "c1"),
			),
			ir.Equal(
				ir.NewOperandColumn("public", "conditional_joins_1", "c2"),
				ir.NewOperandColumn("public", "conditional_joins_2", "c3"),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("condition with or", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := "SELECT * FROM conditional_joins_1 as cj1 JOIN conditional_joins_2 cj2 ON cj1.c1 = cj2.c1 OR cj1.c2 = cj2.c3;"
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.Or(
			ir.Equal(
				ir.NewOperandColumn("public", "conditional_joins_1", "c1"),
				ir.NewOperandColumn("public", "conditional_joins_2", "c1"),
			),
			ir.Equal(
				ir.NewOperandColumn("public", "conditional_joins_1", "c2"),
				ir.NewOperandColumn("public", "conditional_joins_2", "c3"),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})
}

func TestMultipleJoins(t *testing.T) {
	t.Parallel()

	setupQ := `
DROP TABLE IF EXISTS multiple_joins_1;
CREATE TABLE multiple_joins_1 (
	id integer PRIMARY KEY,
	c1 integer,
	c2 integer
);
DROP TABLE IF EXISTS multiple_joins_2;
CREATE TABLE multiple_joins_2 (
	id integer PRIMARY KEY,
	c1 integer,
	c3 integer
);
DROP TABLE IF EXISTS multiple_joins_3;
CREATE TABLE multiple_joins_3 (
	id integer PRIMARY KEY,
	c1 integer
);
DROP TABLE IF EXISTS multiple_joins_4;
CREATE TABLE multiple_joins_4 (
	id integer PRIMARY KEY,
	c1 integer
);
`

	t.Run("multiple joins", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM multiple_joins_1
JOIN multiple_joins_2 mj2 ON multiple_joins_1.c1 = mj2.c1
JOIN multiple_joins_3 mj3 ON mj2.c1 = mj3.c1
JOIN multiple_joins_4 ON multiple_joins_4.c1 = multiple_joins_1.id
;`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.And(
			ir.And(
				ir.Equal(
					ir.NewOperandColumn("public", "multiple_joins_1", "c1"),
					ir.NewOperandColumn("public", "multiple_joins_2", "c1"),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "multiple_joins_2", "c1"),
					ir.NewOperandColumn("public", "multiple_joins_3", "c1"),
				),
			),
			ir.Equal(
				ir.NewOperandColumn("public", "multiple_joins_4", "c1"),
				ir.NewOperandColumn("public", "multiple_joins_1", "id"),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})
}

func TestWhere(t *testing.T) {
	t.Parallel()

	setupQ := `
DROP TABLE IF EXISTS where_1;
CREATE TABLE where_1 (
	id integer PRIMARY KEY,
	c1 integer,
	c2 integer,
	c3 integer,
	c4 integer,
	cb boolean
);
DROP TABLE IF EXISTS where_2;
CREATE TABLE where_2 (
	id integer PRIMARY KEY,
	c1 integer,
	c2 integer,
	c3 integer
);
DROP TABLE IF EXISTS where_3;
CREATE TABLE where_3 (
	id integer PRIMARY KEY,
	c1 integer
);
`

	t.Run("where with mix os supported operators", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1 WHERE
	id = 0 OR
	id != 42 OR
	id > 1 OR
	id >= 1 OR
	id < 10 OR
	id <= 10 OR
	c2 > c1 OR
	c2 >= c1 OR
	c2 < c1 OR
	c2 <= c1 OR
	cb OR
	c3 IS NULL AND
	c4 IS NOT NULL
;`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		// probably should break this down one of these days...
		expectedExpr := ir.Or(
			ir.Or(
				ir.Or(
					ir.Or(
						ir.Or(
							ir.Or(
								ir.Or(
									ir.Or(
										ir.Or(
											ir.Or(
												ir.Or(
													ir.Equal(
														ir.NewOperandColumn("public", "where_1", "id"),
														ir.NewOperandConstant("0"),
													),
													ir.NotEqual(
														ir.NewOperandColumn("public", "where_1", "id"),
														ir.NewOperandConstant("42"),
													),
												),
												ir.GT(
													ir.NewOperandColumn("public", "where_1", "id"),
													ir.NewOperandConstant("1"),
												),
											),
											ir.GE(
												ir.NewOperandColumn("public", "where_1", "id"),
												ir.NewOperandConstant("1"),
											),
										),
										ir.LT(
											ir.NewOperandColumn("public", "where_1", "id"),
											ir.NewOperandConstant("10"),
										),
									),
									ir.LE(
										ir.NewOperandColumn("public", "where_1", "id"),
										ir.NewOperandConstant("10"),
									),
								),
								ir.GT(
									ir.NewOperandColumn("public", "where_1", "c2"),
									ir.NewOperandColumn("public", "where_1", "c1"),
								),
							),
							ir.GE(
								ir.NewOperandColumn("public", "where_1", "c2"),
								ir.NewOperandColumn("public", "where_1", "c1"),
							),
						),
						ir.LT(
							ir.NewOperandColumn("public", "where_1", "c2"),
							ir.NewOperandColumn("public", "where_1", "c1"),
						),
					),
					ir.LE(
						ir.NewOperandColumn("public", "where_1", "c2"),
						ir.NewOperandColumn("public", "where_1", "c1"),
					),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "cb"),
					ir.NewOperandConstant("true"),
				),
			),
			ir.And(
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c3"),
					ir.OperandNil{},
				),
				ir.NotEqual(
					ir.NewOperandColumn("public", "where_1", "c4"),
					ir.OperandNil{},
				),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("is null", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `SELECT * FROM where_1 WHERE c1 IS NULL;`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.Equal(
			ir.NewOperandColumn("public", "where_1", "c1"),
			ir.OperandNil{},
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("is not null", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `SELECT * FROM where_1 WHERE c1 IS NOT NULL;`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.NotEqual(
			ir.NewOperandColumn("public", "where_1", "c1"),
			ir.OperandNil{},
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("bool assertion on single column", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `SELECT * FROM where_1 WHERE c3;`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.Equal(
			ir.NewOperandColumn("public", "where_1", "c3"),
			ir.NewOperandConstant("true"),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("in operator simple", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		// c1 column that must be correctly processed in the correct context since both tables
		q := `
SELECT * FROM where_1
WHERE c1 IN (
	SELECT id FROM where_2 WHERE c1 = 42
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.And(
			ir.Equal(
				ir.NewOperandColumn("public", "where_1", "c1"),
				ir.NewOperandColumn("public", "where_2", "id"),
			),
			ir.Equal(
				ir.NewOperandColumn("public", "where_2", "c1"),
				ir.NewOperandConstant("42"),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("not in operator simple", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		// c1 column that must be correctly processed in the correct context since both tables
		q := `
SELECT * FROM where_1
WHERE c1 NOT IN (
	SELECT id FROM where_2 WHERE c1 = 42
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.Not(
			ir.And(
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c1"),
					ir.NewOperandColumn("public", "where_2", "id"),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "where_2", "c1"),
					ir.NewOperandConstant("42"),
				),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("in operator with tuple", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		// c1 column that must be correctly processed in the correct context since both tables
		q := `
SELECT * FROM where_1
WHERE (c1, c2) IN (
	SELECT id, c1 FROM where_2 WHERE c1 = 42
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.And(
			ir.And(
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c1"),
					ir.NewOperandColumn("public", "where_2", "id"),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c2"),
					ir.NewOperandColumn("public", "where_2", "c1"),
				),
			),
			ir.Equal(
				ir.NewOperandColumn("public", "where_2", "c1"),
				ir.NewOperandConstant("42"),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("in operator with tuple and constants on the left", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE (42, 24) IN (
	SELECT id, c1 FROM where_2 WHERE c1 = 42
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.And(
			ir.And(
				ir.Equal(
					ir.NewOperandConstant("42"),
					ir.NewOperandColumn("public", "where_2", "id"),
				),
				ir.Equal(
					ir.NewOperandConstant("24"),
					ir.NewOperandColumn("public", "where_2", "c1"),
				),
			),
			ir.Equal(
				ir.NewOperandColumn("public", "where_2", "c1"),
				ir.NewOperandConstant("42"),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("in operator with tuple and constants on the right", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE (c1, c2) IN ((1, 2), (3, 4))
;`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.Or(
			ir.And(
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c1"),
					ir.NewOperandConstant("1"),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c2"),
					ir.NewOperandConstant("2"),
				),
			),
			ir.And(
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c1"),
					ir.NewOperandConstant("3"),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c2"),
					ir.NewOperandConstant("4"),
				),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("(not) in operator with tuple and constants on the right", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE (c1, c2) NOT IN ((1, 2), (3, 4))
;`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.Not(
			ir.Or(
				ir.And(
					ir.Equal(
						ir.NewOperandColumn("public", "where_1", "c1"),
						ir.NewOperandConstant("1"),
					),
					ir.Equal(
						ir.NewOperandColumn("public", "where_1", "c2"),
						ir.NewOperandConstant("2"),
					),
				),
				ir.And(
					ir.Equal(
						ir.NewOperandColumn("public", "where_1", "c1"),
						ir.NewOperandConstant("3"),
					),
					ir.Equal(
						ir.NewOperandColumn("public", "where_1", "c2"),
						ir.NewOperandConstant("4"),
					),
				),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("in operator constants on the right", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE c1 IN (1, 2, 3)
;`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.Or(
			ir.Or(
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c1"),
					ir.NewOperandConstant("1"),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c1"),
					ir.NewOperandConstant("2"),
				),
			),
			ir.Equal(
				ir.NewOperandColumn("public", "where_1", "c1"),
				ir.NewOperandConstant("3"),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("in operator with tuple - inner select projection within parenthesis", func(t *testing.T) {
		// the parser generated a different representation...
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE (c1, c2) IN (
	SELECT (id, c1) FROM where_2 WHERE c1 = 42
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.And(
			ir.And(
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c1"),
					ir.NewOperandColumn("public", "where_2", "id"),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c2"),
					ir.NewOperandColumn("public", "where_2", "c1"),
				),
			),
			ir.Equal(
				ir.NewOperandColumn("public", "where_2", "c1"),
				ir.NewOperandConstant("42"),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("multiple operators in with and", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE c1 IN (
	SELECT id FROM where_2 WHERE c1 = 42
) AND c2 IN (
	SELECT id FROM where_3 WHERE c1 = 24
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.And(
			ir.And(
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c1"),
					ir.NewOperandColumn("public", "where_2", "id"),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "where_2", "c1"),
					ir.NewOperandConstant("42"),
				),
			),
			ir.And(
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c2"),
					ir.NewOperandColumn("public", "where_3", "id"),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "where_3", "c1"),
					ir.NewOperandConstant("24"),
				),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("multiple operators in with or", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE c1 IN (
	SELECT id FROM where_2 WHERE c1 = 42
) OR c2 IN (
	SELECT id FROM where_3 WHERE c1 = 24
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.Or(
			ir.And(
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c1"),
					ir.NewOperandColumn("public", "where_2", "id"),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "where_2", "c1"),
					ir.NewOperandConstant("42"),
				),
			),
			ir.And(
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c2"),
					ir.NewOperandColumn("public", "where_3", "id"),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "where_3", "c1"),
					ir.NewOperandConstant("24"),
				),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("multiple operators (not) in", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE c1 IN (
	SELECT id FROM where_2 WHERE c1 = 42
) AND c2 NOT IN (
	SELECT id FROM where_3 WHERE c1 = 24
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.And(
			ir.And(
				ir.Equal(
					ir.NewOperandColumn("public", "where_1", "c1"),
					ir.NewOperandColumn("public", "where_2", "id"),
				),
				ir.Equal(
					ir.NewOperandColumn("public", "where_2", "c1"),
					ir.NewOperandConstant("42"),
				),
			),
			ir.Not(
				ir.And(
					ir.Equal(
						ir.NewOperandColumn("public", "where_1", "c2"),
						ir.NewOperandColumn("public", "where_3", "id"),
					),
					ir.Equal(
						ir.NewOperandColumn("public", "where_3", "c1"),
						ir.NewOperandConstant("24"),
					),
				),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("exists operator simple", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE EXISTS (
	SELECT 1 FROM where_2 WHERE c1 = where_1.c4
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.Equal(
			ir.NewOperandColumn("public", "where_2", "c1"),
			ir.NewOperandColumn("public", "where_1", "c4"),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("exists operator with boolean expression", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE id = c1 AND EXISTS (
	SELECT 1 FROM where_2 WHERE c1 = where_1.c4
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.And(
			ir.Equal(
				ir.NewOperandColumn("public", "where_1", "id"),
				ir.NewOperandColumn("public", "where_1", "c1"),
			),
			ir.Equal(
				ir.NewOperandColumn("public", "where_2", "c1"),
				ir.NewOperandColumn("public", "where_1", "c4"),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("not exists operator with boolean expression", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE id = c1 AND NOT EXISTS (
	SELECT 1 FROM where_2 WHERE c1 = where_1.c4
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.And(
			ir.Equal(
				ir.NewOperandColumn("public", "where_1", "id"),
				ir.NewOperandColumn("public", "where_1", "c1"),
			),
			ir.Not(
				ir.Equal(
					ir.NewOperandColumn("public", "where_2", "c1"),
					ir.NewOperandColumn("public", "where_1", "c4"),
				),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("multiple operators exist with and", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE EXISTS (
	SELECT id FROM where_2 WHERE c1 = 42
) AND EXISTS (
	SELECT id FROM where_3 WHERE c1 = 24
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.And(
			ir.Equal(
				ir.NewOperandColumn("public", "where_2", "c1"),
				ir.NewOperandConstant("42"),
			),
			ir.Equal(
				ir.NewOperandColumn("public", "where_3", "c1"),
				ir.NewOperandConstant("24"),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("multiple operators exist with or", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE EXISTS (
	SELECT id FROM where_2 WHERE c1 = 42
) OR EXISTS (
	SELECT id FROM where_3 WHERE c1 = 24
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.Or(
			ir.Equal(
				ir.NewOperandColumn("public", "where_2", "c1"),
				ir.NewOperandConstant("42"),
			),
			ir.Equal(
				ir.NewOperandColumn("public", "where_3", "c1"),
				ir.NewOperandConstant("24"),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})

	t.Run("multiple operators (not) exist with and", func(t *testing.T) {
		inspect := setupTest(t, setupQ)
		q := `
SELECT * FROM where_1
WHERE EXISTS (
	SELECT id FROM where_2 WHERE c1 = 42
) AND NOT EXISTS (
	SELECT id FROM where_3 WHERE c1 = 24
);`
		expr, err := Parse(context.Background(), inspect.TableWithSchema, q)
		require.NoError(t, err)
		expectedExpr := ir.And(
			ir.Equal(
				ir.NewOperandColumn("public", "where_2", "c1"),
				ir.NewOperandConstant("42"),
			),
			ir.Not(
				ir.Equal(
					ir.NewOperandColumn("public", "where_3", "c1"),
					ir.NewOperandConstant("24"),
				),
			),
		)
		require.Equal(t, expectedExpr, expr)
	})
}
