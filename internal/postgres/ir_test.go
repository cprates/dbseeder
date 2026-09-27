package postgres

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/kr/pretty"
)

func TestXxx(t *testing.T) {
	// getTableF := func(_ context.Context, _, name string) (TableMeta, error) {
	// 	m := map[string]TableMeta{
	// 		"users": {
	// 			name: TableName{
	// 				schema: DefaultSchema,
	// 				name:   "users",
	// 			},
	// 			columns: []*ColumnMeta{
	// 				{Name: "id"},
	// 			},
	// 		},
	// 	}
	// 	return m[name], nil
	// }
	// sql := `SELECT * FROM users WHERE id > 100;`

	// sql := `SELECT * FROM users as usrs
	// JOIN table2 t2 ON usrs.c1 != t2.c2;
	// `

	// TODO: < <= > >= on the join
	sql := `SELECT * FROM users as usrs
	JOIN table2 t2 ON usrs.c1 != t2.c2;
	`

	// sql := `SELECT * FROM users as usrs
	// JOIN table2 t2 ON usrs.c1 = t2.c2 JOIN table3 t3 ON usrs.c3 = t3.c1;
	// `

	// operation OR on the join
	// sql := `SELECT * FROM users as usrs
	// JOIN table2 t2 ON usrs.c1 = t2.c2 AND t2.c3 >= usrs.c2 OR t2.c4 > 42 JOIN table3 t3 ON usrs.c3 = t3.c1;
	// `

	// // check if I can do the second join based on a column from the previous join
	// sql := `SELECT * FROM users as usrs
	// JOIN table2 t2 ON usrs.c1 = t2.c2 JOIN table3 t3 ON usrs.c3 = t2.c3;
	// `

	// multiple joins
	// sql := `SELECT DISTINCT o.* FROM observations o
	// JOIN work_orders wo ON wo.id = o.work_order_id
	// JOIN vessel_jobs vj ON vj.id = wo.vessel_job_id
	// JOIN installed_equipment ie ON ie.id = vj.installed_equipment_id;`

	// where simple
	// sql := `SELECT * FROM table1 WHERE
	// t1 > 1 AND t1 < 10 OR t2 >= 1 AND t2 <= 10 AND cb AND c1 = 'val_c1' OR c2 <> 'not_val_c2' AND c3 IS NULL AND c4 IS NOT NULL;`

	// sql := `SELECT * FROM table1 WHERE c3 IS NULL;`
	// sql := `SELECT * FROM table1 WHERE c3 IS NOT NULL;`
	// sql := `SELECT * FROM table1 WHERE c3;`

	// where complex. Note the c1 column that must be correctly processed in the correct context since both tables
	// have it
	// sql := `SELECT * FROM table1
	// WHERE (c1, c2) IN (
	// 	SELECT id, id2 FROM table2 WHERE c1 = 42
	// );`

	// sql := `SELECT * FROM table1
	// WHERE c1 IN (
	// 	SELECT id FROM table2 WHERE c1 = 42
	// );`

	// sql := `SELECT * FROM table1
	// WHERE c1 = c4 AND EXISTS (
	// 	SELECT 1 FROM table2 WHERE c1 = 42
	// );`

	// getTableF := func(_ context.Context, _, name string) (TableMeta, error) {
	// 	m := map[string]TableMeta{
	// 		"table1": {
	// 			name: TableName{
	// 				schema: DefaultSchema,
	// 				name:   "table1",
	// 			},
	// 			columns: []*ColumnMeta{
	// 				{Name: "c1"}, {Name: "c4"},
	// 			},
	// 		},
	// 		"table2": {
	// 			name: TableName{
	// 				schema: DefaultSchema,
	// 				name:   "table2",
	// 			},
	// 			columns: []*ColumnMeta{
	// 				{Name: "c1"},
	// 			},
	// 		},
	// 	}

	// 	return m[name], nil
	// }
	// sql := `SELECT * FROM table1
	// WHERE c1 = c4 OR EXISTS (
	// 	SELECT * FROM table2 WHERE c1 = 42
	// );`

	// sql := `SELECT * FROM table1
	// WHERE c1 = c4 AND NOT EXISTS (
	// 	SELECT 1 FROM table2 WHERE c1 = 42
	// );`

	// sql := `SELECT * FROM table1
	// WHERE c1 = c4 OR NOT EXISTS (
	// 	SELECT 1 FROM table2 WHERE c1 = 42
	// );`

	// same but for NOT IN
	// sql := `SELECT * FROM table1
	// WHERE (c1, c2) NOT IN (
	// 	SELECT id, id2 FROM table2 WHERE c1 = 42
	// );`

	// Same as above but the projection of the sub-query is within parentheses
	// sql := `SELECT * FROM table1
	// WHERE (c1, c2) NOT IN (
	// 	SELECT (id, id2) FROM table2 WHERE c1 = 42
	// );`

	// sql := `SELECT * FROM table1
	// WHERE c1 NOT IN (
	// 	SELECT id FROM table2 WHERE c1 = 42
	// );`

	// --------------------------------------------------------------------------
	// same as above but with multiple INs
	// sql := `SELECT * FROM table1
	// WHERE c1 IN (
	// 	SELECT id FROM table2 WHERE c1 = 42
	// ) AND c2 IN (
	// 	SELECT id FROM table3 WHERE c1 = 24
	// );`

	// sql := `SELECT * FROM table1
	// WHERE c1 IN (
	// 	SELECT id FROM table2 WHERE c1 = 42
	// ) OR c2 IN (
	// 	SELECT id FROM table3 WHERE c1 = 24
	// );`

	// sql := `SELECT * FROM table1
	// WHERE c1 IN (
	// 	SELECT id FROM table2 WHERE c1 = 42
	// ) AND c2 NOT IN (
	// 	SELECT id FROM table3 WHERE c1 = 24
	// );`

	// Same as above but with EXISTS
	// sql := `SELECT * FROM table1
	// WHERE EXISTS (
	// 	SELECT * FROM table2 WHERE c1 = 42
	// ) AND EXISTS (
	// 	SELECT 1 FROM table3 WHERE c1 = 24
	// );`

	// sql := `SELECT * FROM table1
	// WHERE c1 IN (
	// 	SELECT id FROM table2 WHERE c1 = 42
	// ) OR c2 IN (
	// 	SELECT id FROM table3 WHERE c1 = 24
	// );`

	// sql := `SELECT * FROM table1
	// WHERE EXISTS (
	// 	SELECT id FROM table2 WHERE c1 = 42
	// ) AND NOT EXISTS (
	// 	SELECT id FROM table3 WHERE c1 = 24
	// );`

	// with constant on the left
	// sql := `SELECT * FROM table1
	// WHERE (24, c2) IN (
	// 	SELECT id1, id2 FROM table2 WHERE c1 = 42
	// );`

	// constants on the right
	// sql := `SELECT * FROM table1 WHERE c1 IN (1, 2, 3);`
	// sql := `SELECT * FROM table1 WHERE (c1, c2) IN ((1, 2), (3, 4));`
	// sql := `SELECT * FROM table1 WHERE (c1, c2) NOT IN ((1, 2), (3, 4));`

	// getTableF := func(_ context.Context, _, name string) (TableMeta, error) {
	// 	m := map[string]TableMeta{
	// 		"observations": {
	// 			name: TableName{
	// 				schema: DefaultSchema,
	// 				name:   "observations",
	// 			},
	// 			columns: []*ColumnMeta{
	// 				{Name: "id"}, {Name: "work_order_id"}, {Name: "last_updated_at"}, {Name: "account_id"},
	// 			},
	// 		},
	// 		"work_orders": {
	// 			name: TableName{
	// 				schema: DefaultSchema,
	// 				name:   "work_orders",
	// 			},
	// 			columns: []*ColumnMeta{
	// 				{Name: "id"}, {Name: "vessel_job_id"}, {Name: "last_updated_at"},
	// 			},
	// 		},
	// 		"vessel_jobs": {
	// 			name: TableName{
	// 				schema: DefaultSchema,
	// 				name:   "vessel_jobs",
	// 			},
	// 			columns: []*ColumnMeta{
	// 				{Name: "id"}, {Name: "last_updated_at"}, {Name: "installed_equipment_id"},
	// 			},
	// 		},
	// 		"installed_equipment": {
	// 			name: TableName{
	// 				schema: DefaultSchema,
	// 				name:   "installed_equipment",
	// 			},
	// 			columns: []*ColumnMeta{
	// 				{Name: "id"}, {Name: "vessel_id"}, {Name: "last_updated_at"},
	// 			},
	// 		},
	// 		"table2": {
	// 			name: TableName{
	// 				schema: DefaultSchema,
	// 				name:   "table2",
	// 			},
	// 			columns: []*ColumnMeta{
	// 				{Name: "c1"}, {Name: "updated_at"},
	// 			},
	// 		},
	// 	}

	// 	return m[name], nil
	// }
	// // where with joins and aliases
	// sql := `SELECT DISTINCT o.* FROM observations o
	// JOIN work_orders wo ON wo.id = o.work_order_id
	// JOIN vessel_jobs vj ON vj.id = wo.vessel_job_id
	// JOIN installed_equipment ie ON ie.id = vj.installed_equipment_id
	// WHERE o.last_updated_at > 'since' AND
	// 	o.last_updated_at <= 'til' AND
	// 	o.account_id = 'account_id' AND
	// 	ie.vessel_id = 'vesselid' AND EXISTS (
	// 	 	SELECT * FROM table2 WHERE c1 = 42 AND c1 = ie.last_updated_at
	// 	);`

	// TODO: where clause with ANY and ALL
	// TODO: where clause with LIKE
	// TODO: where clause with BETWEEN?
	// TODO: sql := `SELECT * FROM table1 WHERE (c1, c2) = (1, 2);`
	// TODO: extract scope from a query like 'SELECT * FROM table1, table2 WHERE ...'?

	// Parse the SQL string
	ctx := context.Background()
	// bet, err := Parse(ctx, getTableF, sql)
	bet, err := parse(ctx, true, nil, sql)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	fmt.Println(pretty.Sprint(bet))
}
