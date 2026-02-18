package ast

type Node interface {
	node()
	Pos() (line, col int)
}

type File struct {
	Items []Node
}

type Label struct {
	Name string
	Line int
	Col  int
}

func (l *Label) node()           {}
func (l *Label) Pos() (int, int) { return l.Line, l.Col }

type Directive struct {
	Name string
	Args []string
	Line int
	Col  int
}

func (d *Directive) node()           {}
func (d *Directive) Pos() (int, int) { return d.Line, d.Col }

type Instruction struct {
	Mnemonic string
	Operands []Operand
	Line     int
	Col      int
}

func (i *Instruction) node()           {}
func (i *Instruction) Pos() (int, int) { return i.Line, i.Col }

type DataDecl struct {
	Kind  string
	Items []ExprOrString
	Line  int
	Col   int
}

func (d *DataDecl) node()           {}
func (d *DataDecl) Pos() (int, int) { return d.Line, d.Col }

type Macro struct {
	Name   string
	Params []string
	Body   []Node
	Line   int
	Col    int
}

func (m *Macro) node()           {}
func (m *Macro) Pos() (int, int) { return m.Line, m.Col }

type IfBlock struct {
	Cond Expr
	Then []Node
	Else []Node
	Line int
	Col  int
}

func (ib *IfBlock) node()           {}
func (ib *IfBlock) Pos() (int, int) { return ib.Line, ib.Col }

type Operand interface{ operand() }

type ExprOrString struct {
	Expr  Expr
	Str   string
	IsStr bool
}

type RegOperand struct{ Name string }

func (RegOperand) operand() {}

type ImmOperand struct{ Val Expr }

func (ImmOperand) operand() {}

type MemOperand struct {
	Base  string
	Index string
	Scale int
	Disp  Expr
	Size  int
	Line  int
	Col   int
}

func (MemOperand) operand() {}

type LabelOperand struct{ Name string }

func (LabelOperand) operand() {}

type StrOperand struct{ S string }

func (StrOperand) operand() {}

type Expr interface{ expr() }

type NumberExpr struct{ Val int64 }

func (NumberExpr) expr() {}

type IdentExpr struct{ Name string }

func (IdentExpr) expr() {}

type StringExpr struct{ S string }

func (StringExpr) expr() {}

type BinaryExpr struct {
	Op          string
	Left, Right Expr
}

func (BinaryExpr) expr() {}

type UnaryExpr struct {
	Op string
	X  Expr
}

func (UnaryExpr) expr() {}
