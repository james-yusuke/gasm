package parser

import (
	"fmt"
	"gasm/internal/ast"
	"gasm/internal/lexer"
	"io"
	"strconv"
	"strings"
)

type Parser struct {
	lx     *lexer.Lexer
	peek   lexer.Token
	have   bool
	Errors []string
}

func New(r io.Reader) *Parser {
	return &Parser{lx: lexer.New(r)}
}

func (p *Parser) next() lexer.Token {
	if p.have {
		p.have = false
		return p.peek
	}
	t := p.lx.NextToken()
	p.peek = t
	return t
}

func (p *Parser) backup(t lexer.Token) {
	p.have = true
	p.peek = t
}

func (p *Parser) expect(kind lexer.TokenKind) lexer.Token {
	t := p.next()
	if t.Kind != kind {
		p.Errors = append(p.Errors, fmt.Sprintf("expected %s but got %s (%s) at line %d", kind, t.Kind, t.Lit, t.Line))
	}
	return t
}

func (p *Parser) ParseFile() *ast.File {
	f := &ast.File{}
	for {
		t := p.next()
		if t.Kind == lexer.TOK_EOF {
			break
		}
		if t.Kind == lexer.TOK_NEWLINE {
			continue
		}

		if t.Kind == lexer.TOK_IDENT {
			next := p.next()
			if next.Kind == lexer.TOK_COLON {
				f.Items = append(f.Items, &ast.Label{Name: t.Lit, Line: t.Line, Col: t.Col})
				p.consumeLine()
				continue
			}
			p.backup(next)

			node := p.parseStatementStartingWithIdent(t)
			if node != nil {
				f.Items = append(f.Items, node)
			}
			continue
		}
		if t.Kind == lexer.TOK_PERCENT {
			n := p.expect(lexer.TOK_IDENT)
			name := "%" + n.Lit
			args := p.collectRestOfLineTokens()
			f.Items = append(f.Items, &ast.Directive{Name: name, Args: args, Line: t.Line, Col: t.Col})
			continue
		}

		if t.Kind == lexer.TOK_DOT {
			next := p.expect(lexer.TOK_IDENT)
			name := "." + next.Lit
			args := p.collectRestOfLineTokens()
			f.Items = append(f.Items, &ast.Directive{Name: name, Args: args, Line: t.Line, Col: t.Col})
			continue
		}

		p.consumeLine()
	}
	return f
}

func (p *Parser) consumeLine() {
	for {
		t := p.next()
		if t.Kind == lexer.TOK_NEWLINE || t.Kind == lexer.TOK_EOF {
			return
		}
	}
}

func (p *Parser) collectRestOfLineTokens() []string {
	var out []string
	for {
		t := p.next()
		if t.Kind == lexer.TOK_NEWLINE || t.Kind == lexer.TOK_EOF {
			break
		}
		out = append(out, t.Lit)
	}
	return out
}

func (p *Parser) parseStatementStartingWithIdent(first lexer.Token) ast.Node {
	lit := strings.ToLower(first.Lit)
	switch lit {
	case "section", "global", "extern", "bits", "org", "align":
		args := p.collectRestOfLineTokens()
		return &ast.Directive{Name: lit, Args: args, Line: first.Line, Col: first.Col}
	case "db", "dw", "dd", "dq", "resb", "resw", "resd":
		items := p.parseDataItems()
		return &ast.DataDecl{Kind: lit, Items: items, Line: first.Line, Col: first.Col}
	case "%macro":
		nameTok := p.next()
		name := nameTok.Lit
		p.consumeLine()
		body := p.parseUntilEndMacro(name)
		return &ast.Macro{Name: name, Params: nil, Body: body, Line: first.Line, Col: first.Col}
	case "%if":
		exprStr := strings.Join(p.collectRestOfLineTokens(), " ")
		cond := ast.IdentExpr{Name: exprStr}
		then := p.parseUntilEnd("%endif")
		return &ast.IfBlock{Cond: cond, Then: then, Line: first.Line, Col: first.Col}
	default:
		ins := &ast.Instruction{Mnemonic: first.Lit, Line: first.Line, Col: first.Col}
		ops := p.parseOperands()
		ins.Operands = ops
		return ins
	}
}

func (p *Parser) parseUntilEndMacro(name string) []ast.Node {
	var nodes []ast.Node
	for {
		t := p.next()
		if t.Kind == lexer.TOK_EOF {
			break
		}
		if t.Kind == lexer.TOK_NEWLINE {
			continue
		}

		if t.Kind == lexer.TOK_PERCENT {
			n := p.next()
			if strings.ToLower(n.Lit) == "endmacro" {
				p.consumeLine()
				break
			}
			p.backup(n)
			p.backup(t)
		}

		if t.Kind == lexer.TOK_IDENT {
			node := p.parseStatementStartingWithIdent(t)
			if node != nil {
				nodes = append(nodes, node)
			}
		} else {
			p.consumeLine()
		}
	}
	return nodes
}

func (p *Parser) parseUntilEnd(endTok string) []ast.Node {
	var nodes []ast.Node
	for {
		t := p.next()
		if t.Kind == lexer.TOK_EOF {
			break
		}
		if t.Kind == lexer.TOK_PERCENT {
			n := p.next()
			if strings.ToLower(n.Lit) == strings.TrimPrefix(endTok, "%") {
				p.consumeLine()
				break
			}
			p.backup(n)
		}
		if t.Kind == lexer.TOK_IDENT {
			node := p.parseStatementStartingWithIdent(t)
			if node != nil {
				nodes = append(nodes, node)
			}
		} else {
			p.consumeLine()
		}
	}
	return nodes
}

func (p *Parser) parseDataItems() []ast.ExprOrString {
	var out []ast.ExprOrString
	for {
		t := p.next()
		if t.Kind == lexer.TOK_NEWLINE || t.Kind == lexer.TOK_EOF {
			break
		}
		if t.Kind == lexer.TOK_STRING {
			out = append(out, ast.ExprOrString{IsStr: true, Str: t.Lit})
			continue
		}
		if t.Kind == lexer.TOK_NUMBER || t.Kind == lexer.TOK_IDENT || t.Kind == lexer.TOK_PLUS || t.Kind == lexer.TOK_MINUS || t.Kind == lexer.TOK_LPAREN {
			p.backup(t)
			expr := p.parseExpr()
			out = append(out, ast.ExprOrString{Expr: expr})

			n := p.next()
			if n.Kind == lexer.TOK_COMMA {
				continue
			}
			p.backup(n)
			continue
		}

	}
	return out
}

func (p *Parser) parseOperands() []ast.Operand {
	var ops []ast.Operand
	for {
		t := p.next()
		if t.Kind == lexer.TOK_NEWLINE || t.Kind == lexer.TOK_EOF {
			break
		}
		if t.Kind == lexer.TOK_COMMA {
			continue
		}

		if t.Kind == lexer.TOK_NUMBER {
			p.backup(t)
			expr := p.parseExpr()
			ops = append(ops, ast.ImmOperand{Val: expr})
			continue
		}
		if t.Kind == lexer.TOK_HASH {
			n := p.next()
			if n.Kind == lexer.TOK_NUMBER {
				p.backup(n)
				expr := p.parseExpr()
				ops = append(ops, ast.ImmOperand{Val: expr})
				continue
			}
			p.backup(n)
		}
		if t.Kind == lexer.TOK_STRING {
			ops = append(ops, ast.StrOperand{S: t.Lit})
			continue
		}
		if t.Kind == lexer.TOK_LBRACK {
			inner := p.readUntilMatchingBracket()
			exprp := New(strings.NewReader(inner))
			ex := exprp.parseExpr()
			mem := &ast.MemOperand{Disp: ex}
			ops = append(ops, mem)
			continue
		}
		if t.Kind == lexer.TOK_IDENT {
			if isRegister(t.Lit) {
				ops = append(ops, ast.RegOperand{Name: t.Lit})
				continue
			}

			ops = append(ops, ast.LabelOperand{Name: t.Lit})
			continue
		}

	}
	return ops
}

func (p *Parser) readUntilMatchingBracket() string {
	var sb strings.Builder
	depth := 1
	for {
		t := p.next()
		if t.Kind == lexer.TOK_EOF {
			break
		}
		if t.Kind == lexer.TOK_LBRACK {
			depth++
			sb.WriteString(t.Lit)
			continue
		}
		if t.Kind == lexer.TOK_RBRACK {
			depth--
			if depth == 0 {
				break
			}
			sb.WriteString(t.Lit)
			continue
		}
		if t.Kind == lexer.TOK_STRING {
			sb.WriteString("\"")
			sb.WriteString(t.Lit)
			sb.WriteString("\"")
			continue
		}
		sb.WriteString(t.Lit)

		sb.WriteString(" ")
	}
	return sb.String()
}

func isRegister(s string) bool {
	reg := strings.ToLower(s)
	switch reg {
	case "al", "ah", "ax", "eax", "rax",
		"bl", "bh", "bx", "ebx", "rbx",
		"cl", "ch", "cx", "ecx", "rcx",
		"dl", "dh", "dx", "edx", "rdx",
		"si", "esi", "rsi", "di", "edi", "rdi",
		"sp", "esp", "rsp", "bp", "ebp", "rbp",
		"r8", "r9", "r10", "r11", "r12", "r13", "r14", "r15",
		"r8d", "r9d", "r10d", "r11d", "r12d", "r13d", "r14d", "r15d",
		"r8w", "r9w", "r10w", "r11w", "r12w", "r13w", "r14w", "r15w",
		"r8b", "r9b", "r10b", "r11b", "r12b", "r13b", "r14b", "r15b",
		"mm0", "mm1", "mm2", "mm3", "mm4", "mm5", "mm6", "mm7",
		"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6", "xmm7", "xmm8", "xmm9", "xmm10", "xmm11", "xmm12", "xmm13", "xmm14", "xmm15",
		"cr0", "cr2", "cr3", "cr4",
		"dr0", "dr1", "dr2", "dr3", "dr6", "dr7",
		"rip", "eip", "ip", "flags", "rflags", "eflags":
		return true
	}
	return false
}

func (p *Parser) parseExpr() ast.Expr {
	return p.parseExprLevel1()
}

func (p *Parser) parseExprLevel1() ast.Expr {
	left := p.parseExprLevel2()
	for {
		t := p.next()
		if t.Kind == lexer.TOK_PLUS || t.Kind == lexer.TOK_MINUS {
			right := p.parseExprLevel2()
			left = ast.BinaryExpr{Op: t.Lit, Left: left, Right: right}
			continue
		}
		p.backup(t)
		break
	}
	return left
}

func (p *Parser) parseExprLevel2() ast.Expr {
	left := p.parseExprFactor()
	for {
		t := p.next()
		if t.Kind == lexer.TOK_STAR || t.Kind == lexer.TOK_SLASH {
			right := p.parseExprFactor()
			left = ast.BinaryExpr{Op: t.Lit, Left: left, Right: right}
			continue
		}
		p.backup(t)
		break
	}
	return left
}

func (p *Parser) parseExprFactor() ast.Expr {
	t := p.next()
	if t.Kind == lexer.TOK_NUMBER {
		v, _ := parseNumber(t.Lit)
		return ast.NumberExpr{Val: v}
	}
	if t.Kind == lexer.TOK_IDENT {
		return ast.IdentExpr{Name: t.Lit}
	}
	if t.Kind == lexer.TOK_STRING {
		return ast.StringExpr{S: t.Lit}
	}
	if t.Kind == lexer.TOK_LPAREN {
		e := p.parseExpr()
		p.expect(lexer.TOK_RPAREN)
		return e
	}
	if t.Kind == lexer.TOK_PLUS || t.Kind == lexer.TOK_MINUS {
		x := p.parseExprFactor()
		return ast.UnaryExpr{Op: t.Lit, X: x}
	}

	p.backup(t)
	t2 := p.next()
	if t2.Kind == lexer.TOK_IDENT {
		return ast.IdentExpr{Name: t2.Lit}
	}
	return ast.IdentExpr{Name: t.Lit}
}

func parseNumber(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}

	neg := false
	if s[0] == '+' || s[0] == '-' {
		if s[0] == '-' {
			neg = true
		}
		s = s[1:]
	}
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n, err := strconv.ParseInt(s[2:], 16, 64)
		if err != nil {
			return 0, err
		}
		if neg {
			n = -n
		}
		return n, nil
	}

	if strings.HasSuffix(s, "h") || strings.HasSuffix(s, "H") {
		v := s[:len(s)-1]
		n, err := strconv.ParseInt(v, 16, 64)
		if err != nil {
			return 0, err
		}
		if neg {
			n = -n
		}
		return n, nil
	}

	if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		n, err := strconv.ParseInt(s[2:], 2, 64)
		if err != nil {
			return 0, err
		}
		if neg {
			n = -n
		}
		return n, nil
	}
	if strings.HasSuffix(s, "b") || strings.HasSuffix(s, "B") {
		v := s[:len(s)-1]
		n, err := strconv.ParseInt(v, 2, 64)
		if err != nil {
			return 0, err
		}
		if neg {
			n = -n
		}
		return n, nil
	}

	if strings.HasSuffix(s, "o") || strings.HasSuffix(s, "O") {
		v := s[:len(s)-1]
		n, err := strconv.ParseInt(v, 8, 64)
		if err != nil {
			return 0, err
		}
		if neg {
			n = -n
		}
		return n, nil
	}

	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	if neg {
		n = -n
	}
	return n, nil
}
