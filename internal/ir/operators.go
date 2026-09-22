package ir

type Column interface {
	Val() any
	Cmp(l, r any) int
}

type Row map[string]Column

type Expression interface {
	// Eval evaluates the given rows, a map of table names to row
	Eval(rows map[string]Row) bool
	Left() any
	Right() any
}

type OpAnd struct {
	left  Expression
	right Expression
}

func And(l, r Expression) Expression {
	return OpAnd{
		left:  l,
		right: r,
	}
}

func (o OpAnd) Eval(rows map[string]Row) bool {
	return o.left.Eval(rows) && o.right.Eval(rows)
}

func (o OpAnd) Left() any {
	return o.left
}

func (o OpAnd) Right() any {
	return o.right
}

type OpOr struct {
	left  Expression
	right Expression
}

func Or(l, r Expression) Expression {
	return OpOr{
		left:  l,
		right: r,
	}
}

func (o OpOr) Eval(rows map[string]Row) bool {
	return o.left.Eval(rows) || o.right.Eval(rows)
}

func (o OpOr) Left() any {
	return o.left
}

func (o OpOr) Right() any {
	return o.right
}

type OpNot struct {
	expr Expression
}

func Not(expr Expression) Expression {
	return OpNot{
		expr: expr,
	}
}

func (o OpNot) Eval(rows map[string]Row) bool {
	return !o.expr.Eval(rows)
}

func (o OpNot) Left() any {
	return o.expr.Left()
}

func (o OpNot) Right() any {
	return o.expr.Right()
}

func (o OpNot) Expression() Expression {
	return o.expr
}

// OpEqual handles '=' and 'IS'. Even though they are not exactly the same, should be ok for this purpose.
type OpEqual struct {
	left  Operand
	right Operand
}

func Equal(l, r Operand) OpEqual {
	return OpEqual{left: l, right: r}
}

func (o OpEqual) Eval(rows map[string]Row) bool {
	return o.left.Cmp(rows, o.right) == 0
}

func (o OpEqual) Left() any {
	return o.left
}

func (o OpEqual) Right() any {
	return o.right
}
func (o OpEqual) Negate() OpNotEqual {
	return OpNotEqual(o)
}

// OpNotEqual handles '<>' and '!=' and 'IS NOT'. Even though 'IS NOT' is not exactly the same as the other two,
// should be ok for this purpose.
type OpNotEqual struct {
	left  Operand
	right Operand
}

func NotEqual(l, r Operand) OpNotEqual {
	return OpNotEqual{left: l, right: r}
}

func (o OpNotEqual) Eval(rows map[string]Row) bool {
	return o.left.Cmp(rows, o.right) != 0
}

func (o OpNotEqual) Left() any {
	return o.left
}

func (o OpNotEqual) Right() any {
	return o.right
}

func (o OpNotEqual) Negate() OpEqual {
	return OpEqual(o)
}

type OpGT struct {
	left  Operand
	right Operand
}

func GT(l, r Operand) OpGT {
	return OpGT{left: l, right: r}
}

func (o OpGT) Eval(rows map[string]Row) bool {
	return o.left.Cmp(rows, o.right) == 1
}

func (o OpGT) Left() any {
	return o.left
}

func (o OpGT) Right() any {
	return o.right
}

type OpGE struct {
	left  Operand
	right Operand
}

func GE(l, r Operand) OpGE {
	return OpGE{left: l, right: r}
}

func (o OpGE) Eval(rows map[string]Row) bool {
	r := o.left.Cmp(rows, o.right)

	return r == 0 || r == 1
}

func (o OpGE) Left() any {
	return o.left
}

func (o OpGE) Right() any {
	return o.right
}

type OpLE struct {
	left  Operand
	right Operand
}

func LE(l, r Operand) OpLE {
	return OpLE{left: l, right: r}
}

func (o OpLE) Eval(rows map[string]Row) bool {
	r := o.left.Cmp(rows, o.right)

	return r == -1 || r == 0
}

func (o OpLE) Left() any {
	return o.left
}

func (o OpLE) Right() any {
	return o.right
}

type OpLT struct {
	left  Operand
	right Operand
}

func LT(l, r Operand) OpLT {
	return OpLT{left: l, right: r}
}

func (o OpLT) Eval(rows map[string]Row) bool {
	r := o.left.Cmp(rows, o.right)

	return r == -1
}

func (o OpLT) Left() any {
	return o.left
}

func (o OpLT) Right() any {
	return o.right
}

// OpIn handles IN (...)
type OpIn struct {
	left  Operand
	right []Operand
}

func In(l Operand, r []Operand) OpIn {
	return OpIn{left: l, right: r}
}

func (o OpIn) Eval(rows map[string]Row) bool {
	for _, v := range o.right {
		if o.left.Cmp(rows, v) == 0 {
			return true
		}
	}

	return false
}

func (o OpIn) Left() any {
	return o.left
}

func (o OpIn) Right() any {
	return o.right
}

// OpNoOp always evaluates to true.
type OpNoOp struct{}

func (o OpNoOp) Eval(map[string]Row) bool {
	return true
}

func (o OpNoOp) Left() any {
	return nil
}
func (o OpNoOp) Right() any {
	return nil
}
