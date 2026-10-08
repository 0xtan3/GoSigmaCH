package ast

import (
	"fmt"
	"strings"
	"text/scanner"
)

type Token struct {
	Type  string
	Value string
}

func LexCondition(condition string) []Token {
	var s scanner.Scanner
	s.Init(strings.NewReader(condition))
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanInts
	var tokens []Token
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		val := s.TokenText()
		typ := "IDENT"
		switch strings.ToLower(val) {
		case "and":
			typ = "AND"
		case "or":
			typ = "OR"
		case "not":
			typ = "NOT"
		case "all":
			typ = "ALL"
		case "any":
			typ = "ANY"
		case "of":
			typ = "OF"
		case "1":
			typ = "1"
		}
		if val == "(" {
			typ = "LPAREN"
		} else if val == ")" {
			typ = "RPAREN"
		} else if val == "*" {
			typ = "STAR"
		}
		tokens = append(tokens, Token{Type: typ, Value: val})
	}
	return tokens
}

type Expr interface {
	String() string
}

type AndExpr struct {
	Left  Expr
	Right Expr
}
func (e AndExpr) String() string { return fmt.Sprintf("(%s AND %s)", e.Left, e.Right) }

type OrExpr struct {
	Left  Expr
	Right Expr
}
func (e OrExpr) String() string { return fmt.Sprintf("(%s OR %s)", e.Left, e.Right) }

type NotExpr struct {
	Expr Expr
}
func (e NotExpr) String() string { return fmt.Sprintf("NOT %s", e.Expr) }

type IdentExpr struct {
	Name string
}
func (e IdentExpr) String() string { return e.Name }

type Parser struct {
	tokens []Token
	pos    int
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: "EOF"}
	}
	return p.tokens[p.pos]
}

func (p *Parser) next() Token {
	tok := p.peek()
	p.pos++
	return tok
}

func ParseCondition(condition string, searchKeys []string) (Expr, error) {
	tokens := LexCondition(condition)
	// pre-process ALL OF *, 1 OF *, ANY OF *
	// For simplicity, expand them before parsing if we see the pattern
	var expanded []Token
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if (t.Type == "ALL" || t.Type == "1" || t.Type == "ANY") && i+2 < len(tokens) && tokens[i+1].Type == "OF" {
			pattern := tokens[i+2].Value
			if i+3 < len(tokens) && tokens[i+3].Type == "STAR" {
				pattern += "*"
				i++
			}
			isAll := t.Type == "ALL"
			var matches []string
			for _, k := range searchKeys {
				if matchPattern(k, pattern) {
					matches = append(matches, k)
				}
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("no matches for pattern %s", pattern)
			}
			expanded = append(expanded, Token{Type: "LPAREN", Value: "("})
			for j, m := range matches {
				if j > 0 {
					op := "OR"
					if isAll {
						op = "AND"
					}
					expanded = append(expanded, Token{Type: op, Value: op})
				}
				expanded = append(expanded, Token{Type: "IDENT", Value: m})
			}
			expanded = append(expanded, Token{Type: "RPAREN", Value: ")"})
			i += 2 // skip OF and pattern
		} else {
			expanded = append(expanded, t)
		}
	}

	p := &Parser{tokens: expanded}
	return p.parseOr()
}

func matchPattern(s, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, strings.TrimSuffix(pattern, "*"))
	}
	return s == pattern
}

func (p *Parser) parseOr() (Expr, error) {
	expr, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == "OR" {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		expr = OrExpr{Left: expr, Right: right}
	}
	return expr, nil
}

func (p *Parser) parseAnd() (Expr, error) {
	expr, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == "AND" {
		p.next()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		expr = AndExpr{Left: expr, Right: right}
	}
	return expr, nil
}

func (p *Parser) parseNot() (Expr, error) {
	if p.peek().Type == "NOT" {
		p.next()
		expr, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return NotExpr{Expr: expr}, nil
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() (Expr, error) {
	tok := p.next()
	if tok.Type == "LPAREN" {
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.next().Type != "RPAREN" {
			return nil, fmt.Errorf("expected RPAREN")
		}
		return expr, nil
	}
	if tok.Type == "IDENT" {
		return IdentExpr{Name: tok.Value}, nil
	}
	return nil, fmt.Errorf("unexpected token: %v", tok)
}
