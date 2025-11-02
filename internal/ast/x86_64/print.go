package x86_64

import (
	"fmt"
	"strconv"
)

func PrintAST(f *File) {
	for _, it := range f.Items {
		switch n := it.(type) {
		case *Label:
			fmt.Printf("Label: %s\n", n.Name)
		case *Directive:
			fmt.Printf("Directive: %s %v\n", n.Name, n.Args)
		case *DataDecl:
			fmt.Printf("DataDecl: %s\n", n.Kind)
			for _, it := range n.Items {
				if it.IsStr {
					fmt.Printf("  string: %q\n", it.Str)
				} else {
					fmt.Printf("  expr: %v\n", prettyExpr(it.Expr))
				}
			}
		case *Instruction:
			fmt.Printf("Instr: %s\n", n.Mnemonic)
			for i, op := range n.Operands {
				fmt.Printf("  op%d: %s\n", i, prettyOperand(op))
			}
		case *Macro:
			fmt.Printf("Macro: %s (body %d nodes)\n", n.Name, len(n.Body))
		case *IfBlock:
			fmt.Printf("If: %v then(%d)\n", prettyExpr(n.Cond), len(n.Then))
		default:
			fmt.Printf("Unknown node: %T\n", n)
		}
	}
}

func prettyOperand(op Operand) string {
	switch v := op.(type) {
	case RegOperand:
		return fmt.Sprintf("Reg(%s)", v.Name)
	case ImmOperand:
		return fmt.Sprintf("Imm(%v)", prettyExpr(v.Val))
	case MemOperand:
		return fmt.Sprintf("Mem(%v)", prettyExpr(v.Expr))
	case LabelOperand:
		return fmt.Sprintf("Label(%s)", v.Name)
	case StrOperand:
		return fmt.Sprintf("String(%q)", v.S)
	default:
		return fmt.Sprintf("<unknown op %T>", v)
	}
}

func prettyExpr(e Expr) string {
	switch v := e.(type) {
	case NumberExpr:
		return fmt.Sprintf("%d", v.Val)
	case IdentExpr:
		return v.Name
	case StringExpr:
		return strconv.Quote(v.S)
	case BinaryExpr:
		return fmt.Sprintf("(%s %s %s)", prettyExpr(v.Left), v.Op, prettyExpr(v.Right))
	case UnaryExpr:
		return fmt.Sprintf("(%s%s)", v.Op, prettyExpr(v.X))
	default:
		return fmt.Sprintf("<expr %T>", v)
	}
}
