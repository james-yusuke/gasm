package x86_64

import (
	"bufio"
	"io"
	"strings"
	"unicode"
)

type TokenKind int

const (
	TOK_ILLEGAL TokenKind = iota
	TOK_EOF
	TOK_NEWLINE
	TOK_IDENT
	TOK_NUMBER
	TOK_STRING
	TOK_COLON
	TOK_COMMA
	TOK_LBRACK
	TOK_RBRACK
	TOK_LPAREN
	TOK_RPAREN
	TOK_PLUS
	TOK_MINUS
	TOK_STAR
	TOK_SLASH
	TOK_PERCENT
	TOK_DOT
	TOK_HASH
	TOK_OTHER
)

func (k TokenKind) String() string {
	s := map[TokenKind]string{
		TOK_ILLEGAL: "ILLEGAL",
		TOK_EOF:     "EOF",
		TOK_NEWLINE: "NEWLINE",
		TOK_IDENT:   "IDENT",
		TOK_NUMBER:  "NUMBER",
		TOK_STRING:  "STRING",
		TOK_COLON:   ":",
		TOK_COMMA:   ",",
		TOK_LBRACK:  "[",
		TOK_RBRACK:  "]",
		TOK_LPAREN:  "(",
		TOK_RPAREN:  ")",
		TOK_PLUS:    "+",
		TOK_MINUS:   "-",
		TOK_STAR:    "*",
		TOK_SLASH:   "/",
		TOK_PERCENT: "%",
		TOK_DOT:     ".",
		TOK_HASH:    "#",
		TOK_OTHER:   "OTHER",
	}
	if v, ok := s[k]; ok {
		return v
	}
	return "?"
}

type Token struct {
	Kind TokenKind
	Lit  string
	Line int
	Col  int
}

type Lexer struct {
	r        *bufio.Reader
	line     int
	col      int
	peekRune rune
	havePeek bool
}

func NewLexer(r io.Reader) *Lexer {
	return &Lexer{r: bufio.NewReader(r), line: 1, col: 0}
}

func (lx *Lexer) read() (rune, error) {
	if lx.havePeek {
		lx.havePeek = false
		r := lx.peekRune
		if r == '\n' {
			lx.line++
			lx.col = 0
		} else {
			lx.col++
		}
		return r, nil
	}
	r, _, err := lx.r.ReadRune()
	if err != nil {
		return 0, err
	}
	if r == '\n' {
		lx.line++
		lx.col = 0
	} else {
		lx.col++
	}
	return r, nil
}

func (lx *Lexer) unread(r rune) {
	lx.havePeek = true
	lx.peekRune = r
	if r == '\n' {
		lx.line--
	} else {
		lx.col--
	}
}

func (lx *Lexer) peek() (rune, error) {
	r, err := lx.read()
	if err != nil {
		return 0, err
	}
	lx.unread(r)
	return r, nil
}

func (lx *Lexer) NextToken() Token {
	for {
		r, err := lx.read()
		if err != nil {
			return Token{Kind: TOK_EOF, Lit: "", Line: lx.line, Col: lx.col}
		}

		if r == ' ' || r == '\t' || r == '\r' {
			continue
		}
		if r == '\n' {
			return Token{Kind: TOK_NEWLINE, Lit: "\n", Line: lx.line - 1, Col: lx.col}
		}

		if r == ';' {

			var sb strings.Builder
			for {
				r2, err := lx.read()
				if err != nil || r2 == '\n' {
					if r2 == '\n' {
						lx.unread(r2)
					}
					break
				}
				sb.WriteRune(r2)
			}
			continue
		}

		if r == '"' || r == '\'' {
			quote := r
			var sb strings.Builder
			for {
				r2, err := lx.read()
				if err != nil {
					break
				}
				if r2 == quote {
					break
				}
				if r2 == '\\' {
					r3, _ := lx.read()
					sb.WriteRune(r3)
					continue
				}
				sb.WriteRune(r2)
			}
			return Token{Kind: TOK_STRING, Lit: sb.String(), Line: lx.line, Col: lx.col}
		}

		if unicode.IsLetter(r) || r == '_' || r == '.' || r == '@' {

			var sb strings.Builder
			sb.WriteRune(r)
			for {
				r2, err := lx.read()
				if err != nil {
					break
				}
				if !(unicode.IsLetter(r2) || unicode.IsDigit(r2) || r2 == '_' || r2 == '.' || r2 == '@' || r2 == '$') {
					lx.unread(r2)
					break
				}
				sb.WriteRune(r2)
			}
			return Token{Kind: TOK_IDENT, Lit: sb.String(), Line: lx.line, Col: lx.col}
		}
		if unicode.IsDigit(r) {
			var sb strings.Builder
			sb.WriteRune(r)
			for {
				r2, err := lx.read()
				if err != nil {
					break
				}
				if !(unicode.IsDigit(r2) || (r2 >= 'a' && r2 <= 'f') || (r2 >= 'A' && r2 <= 'F') || r2 == 'x' || r2 == 'b' || r2 == 'o' || r2 == 'h' || r2 == '.') {
					lx.unread(r2)
					break
				}
				sb.WriteRune(r2)
			}
			return Token{Kind: TOK_NUMBER, Lit: sb.String(), Line: lx.line, Col: lx.col}
		}

		switch r {
		case ':':
			return Token{Kind: TOK_COLON, Lit: ":", Line: lx.line, Col: lx.col}
		case ',':
			return Token{Kind: TOK_COMMA, Lit: ",", Line: lx.line, Col: lx.col}
		case '[':
			return Token{Kind: TOK_LBRACK, Lit: "[", Line: lx.line, Col: lx.col}
		case ']':
			return Token{Kind: TOK_RBRACK, Lit: "]", Line: lx.line, Col: lx.col}
		case '(':
			return Token{Kind: TOK_LPAREN, Lit: "(", Line: lx.line, Col: lx.col}
		case ')':
			return Token{Kind: TOK_RPAREN, Lit: ")", Line: lx.line, Col: lx.col}
		case '+':
			return Token{Kind: TOK_PLUS, Lit: "+", Line: lx.line, Col: lx.col}
		case '-':
			return Token{Kind: TOK_MINUS, Lit: "-", Line: lx.line, Col: lx.col}
		case '*':
			return Token{Kind: TOK_STAR, Lit: "*", Line: lx.line, Col: lx.col}
		case '/':
			return Token{Kind: TOK_SLASH, Lit: "/", Line: lx.line, Col: lx.col}
		case '%':
			return Token{Kind: TOK_PERCENT, Lit: "%", Line: lx.line, Col: lx.col}
		case '.':
			return Token{Kind: TOK_DOT, Lit: ".", Line: lx.line, Col: lx.col}
		case '#':
			return Token{Kind: TOK_HASH, Lit: "#", Line: lx.line, Col: lx.col}
		default:

			return Token{Kind: TOK_OTHER, Lit: string(r), Line: lx.line, Col: lx.col}
		}
	}
}
