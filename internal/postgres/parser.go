package postgres

import (
	"context"
	"fmt"

	"github.com/pgplex/pgparser/nodes"
	"github.com/pgplex/pgparser/parser"

	"github.com/cprates/dbseeder/internal/ir"
)

type tableGetter func(ctx context.Context, schema, name string) (TableMeta, error)

type ColumnIdentifier struct {
	Schema       string
	TableOrAlias string
	Name         string
}

var (
	// AllColumnsIdentifier is used to represent '*' in a 'SELECT' projection.
	AllColumnsIdentifier = ColumnIdentifier{
		Name: "*",
	}
)

func Parse(ctx context.Context, getTableF tableGetter, query string) (ir.Expression, error) {
	return parse(ctx, false, getTableF, query)
}

// ParseTest must be used for tests only.
func ParseTest(ctx context.Context, query string) (ir.Expression, error) {
	return parse(ctx, true, nil, query)
}

func parse(ctx context.Context, test bool, getTableF tableGetter, query string) (ir.Expression, error) {
	stmts, err := parser.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("parsing query: %w", err)
	}

	if len(stmts.Items) != 1 {
		return nil, fmt.Errorf("expects one statement, got %d", len(stmts.Items))
	}
	qCtx := &queryContext{
		ctx:        ctx,
		test:       test,
		getTableF:  getTableF,
		scopeStack: queryScopeStack{},
	}
	_, queryExpr := selectToBet(qCtx, nodeToStmt[*nodes.SelectStmt](stmts.Items[0]))
	if queryExpr == nil {
		queryExpr = ir.OpNoOp{}
	}

	return queryExpr, nil
}

func selectToBet(qCtx *queryContext, selectStmt *nodes.SelectStmt) ([]ir.Operand, ir.Expression) {
	fromBet := fromToBet(qCtx, selectStmt.FromClause.Items)
	whereBet := whereToBet(qCtx, selectStmt.WhereClause)
	projection := nodesToBetOperand(qCtx, selectStmt.TargetList.Items)

	return projection, reduceAnd(fromBet, whereBet)
}

func fromToBet(qCtx *queryContext, list []nodes.Node) ir.Expression {
	if len(list) == 0 {
		return nil
	}

	qCtx.pushNewScope()

	item := list[0]
	switch node := item.(type) {
	case *nodes.RangeVar:
		alias := ""
		if node.Alias != nil {
			alias = node.Alias.Aliasname
		}
		// RangeVar is a table name or alias, e.g.: 'FROM table1 AS t1'
		qCtx.addToScope(node.Schemaname, node.Relname, alias)
		return nil
	case *nodes.JoinExpr:
		return reduceAnd(
			joinToBet(qCtx, node),
			fromToBet(qCtx, list[1:]),
		)
	default:
		panic(fmt.Sprintf("unsupported item in 'from' clause: %T", item))
	}
}

func joinToBet(qCtx *queryContext, joinNode *nodes.JoinExpr) ir.Expression {
	// the structure of the tree generated with the current parser isn't optimised to be traversed recursively imho,
	// e.g. mixes operations with qualifiers, so I need to make this in multiple steps
	lBet, err := joinArgToBet(qCtx, joinNode.Larg)
	if err != nil {
		panic(err)
	}
	rBet, err := joinArgToBet(qCtx, joinNode.Rarg)
	if err != nil {
		panic(err)
	}
	joinBet := reduceAnd(lBet, rBet)

	switch qual := joinNode.Quals.(type) {
	case *nodes.A_Expr:
		// join operator, e.g.: JOIN ON t1.c1 = t2.c1
		return reduceAnd(
			joinBet,
			aExprToBet(qCtx, qual),
		)
	case *nodes.BoolExpr:
		// join with composite condition, e.g.: JOIN ON expr AND expr
		return reduceAnd(
			joinBet,
			boolExprToBet(qCtx, qual.Boolop, qual.Args.Items),
		)
	default:
		panic(fmt.Sprintf("unsupported item in 'join' clause: %T", joinNode.Quals))
	}
}

func joinArgToBet(qCtx *queryContext, joinArg nodes.Node) (ir.Expression, error) {
	switch nd := joinArg.(type) {
	case *nodes.JoinExpr:
		return joinToBet(qCtx, nd), nil
	case *nodes.RangeVar:
		alias := ""
		if nd.Alias != nil {
			alias = nd.Alias.Aliasname
		}
		qCtx.addToScope(nd.Schemaname, nd.Relname, alias)

		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported type in 'join' arg: %T", joinArg)
	}
}

func whereToBet(qCtx *queryContext, wNode nodes.Node) ir.Expression {
	if wNode == nil {
		return nil
	}

	switch nd := wNode.(type) {
	case *nodes.BoolExpr:
		return boolExprToBet(qCtx, nd.Boolop, nd.Args.Items)
	case *nodes.A_Expr:
		switch nd.Kind {
		case nodes.AEXPR_IN:
			return inToBet(qCtx, nd)
		default:
			return aExprToBet(qCtx, nd)
		}
	case *nodes.ColumnRef:
		if nd.Fields.Len() != 1 {
			panic(fmt.Errorf("unexpected number of items for a ColumnRef: %d", nd.Fields.Len()))
		}
		// e.g.: condition on a boolean column - 'WHERE cb;'
		return ir.Equal(nodeToBetOperand(qCtx, nd), ir.NewOperandConstant("true"))
	case *nodes.SubLink:
		// at the time of writing this there was no SubLinkType custom type
		switch nd.SubLinkType {
		case 0:
			// EXISTS (SELECT ...)
			return existsToBet(qCtx, nd)
		case 2:
			// c1 IN (SELECT ...)
			return inToBet(qCtx, nd)
		default:
			// TODO: find out what are the other types, at least '1'
			panic(fmt.Errorf("unexpected subquery type: %d", nd.SubLinkType))
		}
	case *nodes.NullTest:
		// IS [NOT] NULL assertion
		return nullTestToBet(qCtx, nd.Nulltesttype, nd.Arg)
	default:
		panic(fmt.Errorf("unsupported type in 'where' clause: %T", wNode))
	}
}

func existsToBet(qCtx *queryContext, subQuery *nodes.SubLink) ir.Expression {
	subSelect := nodeToStmt[*nodes.SelectStmt](subQuery.Subselect)
	_, subQueryExpr := selectToBet(qCtx, subSelect)
	qCtx.popScope()

	return subQueryExpr
}

func inToBet(qCtx *queryContext, node nodes.Node) ir.Expression {
	switch in := node.(type) {
	case *nodes.SubLink:
		leftOpers := testExprToOperands(qCtx, in.Testexpr)
		subSelect := nodeToStmt[*nodes.SelectStmt](in.Subselect)
		projection, subSelectExpr := selectToBet(qCtx, subSelect)
		defer func() {
			qCtx.popScope()
		}()

		var argsExpr ir.Expression
		// trusting the parser will throw an error if left and right operators here do not match
		for i, lOp := range leftOpers {
			argsExpr = reduceAnd(argsExpr, ir.Equal(lOp, projection[i]))
		}

		return reduceAnd(argsExpr, subSelectExpr)
	case *nodes.A_Expr:
		// handles queries like 'c1 IN (1, 2, 3)'
		return aExprToBet(qCtx, in)
	default:
		panic(fmt.Errorf("unexpected node type for in oper: %T", node))
	}
}

// TODO: probably can be replaced by nodesToBetOperand. Tst it when all tests are in place
func testExprToOperands(qCtx *queryContext, testExpr nodes.Node) []ir.Operand {
	// nodes.SubLink.Testexpr varies it's type depending on whether it's a single value or a tuple, etc
	switch args := testExpr.(type) {
	case *nodes.RowExpr:
		// tuple
		return nodesToBetOperand(qCtx, args.Args.Items)
	case *nodes.ColumnRef:
		// single column
		return []ir.Operand{nodeToBetOperand(qCtx, args)}
	default:
		panic(fmt.Errorf("unexpected type of SubLink.Testexpr: %T", testExpr))
	}
}

func reduceAnd(l, r ir.Expression) ir.Expression {
	if l != nil && r != nil {
		return ir.And(l, r)
	}

	if l != nil {
		return l
	}

	if r != nil {
		return r
	}

	return nil
}

func reduceOr(l, r ir.Expression) ir.Expression {
	if l != nil && r != nil {
		return ir.Or(l, r)
	}

	if l != nil {
		return l
	}

	if r != nil {
		return r
	}

	return nil
}

func aExprToBet(qCtx *queryContext, expr *nodes.A_Expr) ir.Expression {
	switch expr.Kind {
	case nodes.AEXPR_OP:
		l := nodeToBetOperand(qCtx, expr.Lexpr)
		r := nodeToBetOperand(qCtx, expr.Rexpr)
		op := expr.Name.Items[0].(*nodes.String)
		switch op.Str {
		case "=":
			return ir.Equal(l, r)
		case "<>":
			return ir.NotEqual(l, r)
		case ">":
			return ir.GT(l, r)
		case ">=":
			return ir.GE(l, r)
		case "<":
			return ir.LT(l, r)
		case "<=":
			return ir.LE(l, r)
		default:
			panic(fmt.Sprintf("unsupported normal operator in 'A_Expr': %s", op.Str))
		}
	case nodes.AEXPR_IN:
		var inExpr ir.Expression
		lOpers := nodesToBetOperand(qCtx, []nodes.Node{expr.Lexpr})
		rOpers := nodesToBetOperand(qCtx, []nodes.Node{expr.Rexpr})
		op := expr.Name.Items[0].(*nodes.String)
		// counting on the parser to make sure the operands are balanced
		for ri := 0; ri < len(rOpers); ri += len(lOpers) {
			var subExpr ir.Expression
			for li, lOper := range lOpers {
				// NOTE: this is a bit ugly as op.Str is constant throughout these iterations...
				switch op.Str {
				// both built the same way, '<>' is negated at the end
				case "=", "<>":
					subExpr = reduceAnd(subExpr, ir.Equal(lOper, rOpers[ri+li]))
				default:
					panic(fmt.Sprintf("unsupported operator in 'AEXPR_IN': %s", op.Str))
				}
			}
			inExpr = reduceOr(inExpr, subExpr)
		}

		if op.Str == "<>" {
			return ir.Not(inExpr)
		}
		return inExpr
	default:
		panic(fmt.Sprintf("unsupported expression kind: %d", expr.Kind))
	}
}

func boolExprToBet(qCtx *queryContext, op nodes.BoolExprType, args []nodes.Node) ir.Expression {
	if len(args) == 0 {
		return nil
	}

	switch lArg := args[0].(type) {
	case *nodes.A_Expr:
		return boolOpToBet(
			op,
			aExprToBet(qCtx, lArg),
			boolExprToBet(qCtx, op, args[1:]),
		)
	case *nodes.BoolExpr:
		return boolOpToBet(
			op,
			boolExprToBet(qCtx, lArg.Boolop, lArg.Args.Items),
			boolExprToBet(qCtx, lArg.Boolop, args[1:]),
		)
	case *nodes.ColumnRef:
		if lArg.Fields.Len() != 1 {
			panic(fmt.Errorf("unexpected number of items for a ColumnRef: %d", lArg.Fields.Len()))
		}
		// e.g.: condition on a boolean column - 'WHERE cb;'
		return ir.Equal(nodeToBetOperand(qCtx, lArg), ir.NewOperandConstant("true"))
	case *nodes.NullTest:
		// IS [NOT] NULL assertion
		return boolOpToBet(
			op,
			nullTestToBet(qCtx, lArg.Nulltesttype, lArg.Arg),
			boolExprToBet(qCtx, op, args[1:]),
		)
	case *nodes.SubLink:
		return boolOpToBet(
			op,
			whereToBet(qCtx, lArg),
			boolExprToBet(qCtx, op, args[1:]),
		)
	default:
		panic(fmt.Sprintf("unsupported bool arg: %T", args[0]))
	}
}

func boolOpToBet(op nodes.BoolExprType, l, r ir.Expression) ir.Expression {
	switch op {
	case nodes.AND_EXPR:
		return reduceAnd(l, r)
	case nodes.OR_EXPR:
		return reduceOr(l, r)
	case nodes.NOT_EXPR:
		return ir.Not(l)
	default:
		panic(fmt.Sprintf("unsupported bool expression: %d", op))
	}
}

func nodesToBetOperand(qCtx *queryContext, nds []nodes.Node) []ir.Operand {
	opers := make([]ir.Operand, 0, len(nds))
	for _, nd := range nds {
		switch n := nd.(type) {
		case *nodes.ResTarget:
			// e.g.: when the caller is the extractor of a select projection
			opers = append(opers, nodesToBetOperand(qCtx, []nodes.Node{n.Val})...)
		case *nodes.RowExpr:
			opers = append(opers, nodesToBetOperand(qCtx, n.Args.Items)...)
		case *nodes.List:
			// e.g.: when the caller is the extractor of a select projection within parentheses
			opers = append(opers, nodesToBetOperand(qCtx, n.Items)...)
		default:
			opers = append(opers, nodeToBetOperand(qCtx, nd))
		}
	}

	return opers
}

func nodeToBetOperand(qCtx *queryContext, node nodes.Node) ir.Operand {
	switch operand := node.(type) {
	case *nodes.ColumnRef:
		colID := columnIdentifierFromColumnRef(operand)
		if colID.Name == AllColumnsIdentifier.Name {
			return ir.NewOperandColumn("", "", colID.Name)
		}
		schema, tableName := qCtx.lookupColumn(colID.Schema, colID.TableOrAlias, colID.Name)
		return ir.NewOperandColumn(schema, tableName, colID.Name)
	case *nodes.A_Const:
		if operand.Isnull {
			return ir.OperandNil{}
		}
		switch constVal := operand.Val.(type) {
		case *nodes.String:
			return ir.NewOperandConstant(constVal.Str)
		default:
			return ir.NewOperandConstant(nodes.NodeToString(operand.Val))
		}
	default:
		panic(fmt.Sprintf("unsupported operand: %T", node))
	}
}

func columnIdentifierFromColumnRef(colRef *nodes.ColumnRef) ColumnIdentifier {
	cid := ColumnIdentifier{}
	items := colRef.Fields.Items
	l := len(items)
	idx := 0
	switch l {
	case 3:
		cid.Schema = nodeToStmt[*nodes.String](items[idx]).Str
		idx++
		fallthrough
	case 2:
		cid.TableOrAlias = nodeToStmt[*nodes.String](items[idx]).Str
		idx++
		fallthrough
	case 1:
		switch name := items[idx].(type) {
		case *nodes.String:
			cid.Name = name.Str
		case *nodes.A_Star:
			cid = AllColumnsIdentifier
		default:
			panic(fmt.Sprintf("unexpected nodes.ColumnRef identifier name: %T", items[idx]))
		}
	default:
		panic(fmt.Sprintf("unexpected list size in nodes.ColumnRef: %d", l))
	}

	return cid
}

func nodeToStmt[T any](node nodes.Node) T {
	stmt, ok := node.(T)
	if !ok {
		var t T
		panic(fmt.Errorf("expect %T but got %T", t, node))
	}

	return stmt
}

func nullTestToBet(qCtx *queryContext, op nodes.NullTestType, arg nodes.Node) ir.Expression {
	switch op {
	case nodes.IS_NULL:
		return ir.Equal(nodeToBetOperand(qCtx, arg), ir.OperandNil{})
	case nodes.IS_NOT_NULL:
		return ir.NotEqual(nodeToBetOperand(qCtx, arg), ir.OperandNil{})
	default:
		panic(fmt.Errorf("unknown null test operator: %d", op))
	}
}
