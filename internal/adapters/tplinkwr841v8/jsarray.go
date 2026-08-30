package tplinkwr841v8

import (
	"strconv"
	"strings"
	"unicode"
)

type TokenKind string

const (
	TokenString TokenKind = "string"
	TokenNumber TokenKind = "number"
	TokenRaw    TokenKind = "raw"
)

type Token struct {
	Kind    TokenKind
	Literal string
}

// ExtractArray lexes a JavaScript new Array(...) assignment without evaluating it.
func ExtractArray(html []byte, name string) ([]Token, bool) {
	s := string(html)
	pos := 0
	for {
		i := strings.Index(s[pos:], name)
		if i < 0 {
			return nil, false
		}
		i += pos
		if (i > 0 && (isIdent(s[i-1]))) || (i+len(name) < len(s) && isIdent(s[i+len(name)])) {
			pos = i + len(name)
			continue
		}
		j := skipSpaceComments(s, i+len(name))
		if j >= len(s) || s[j] != '=' {
			pos = i + len(name)
			continue
		}
		j = skipSpaceComments(s, j+1)
		if !strings.HasPrefix(s[j:], "new") || (j+3 < len(s) && isIdent(s[j+3])) {
			pos = i + len(name)
			continue
		}
		j = skipSpaceComments(s, j+3)
		if !strings.HasPrefix(s[j:], "Array") || (j+5 < len(s) && isIdent(s[j+5])) {
			pos = i + len(name)
			continue
		}
		j = skipSpaceComments(s, j+5)
		if j >= len(s) || s[j] != '(' {
			pos = i + len(name)
			continue
		}
		return lexArray(s, j+1)
	}
}

func isIdent(r byte) bool {
	return r == '_' || r == '$' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}

func skipSpaceComments(s string, i int) int {
	for i < len(s) {
		if unicode.IsSpace(rune(s[i])) {
			i++
			continue
		}
		if i+1 < len(s) && s[i:i+2] == "//" {
			i += 2
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(s) && s[i:i+2] == "/*" {
			end := strings.Index(s[i+2:], "*/")
			if end < 0 {
				return len(s)
			}
			i += end + 4
			continue
		}
		break
	}
	return i
}

func lexArray(s string, i int) ([]Token, bool) {
	var tokens []Token
	for {
		i = skipSpaceComments(s, i)
		if i >= len(s) {
			return nil, false
		}
		if s[i] == ')' {
			return tokens, true
		}
		var token Token
		if s[i] == '\'' || s[i] == '"' {
			quote := s[i]
			i++
			var b strings.Builder
			closed := false
			for i < len(s) {
				if s[i] == '\\' {
					if i+1 >= len(s) {
						return nil, false
					}
					if s[i+1] == 'u' && i+5 < len(s) {
						if code, err := strconv.ParseUint(s[i+2:i+6], 16, 16); err == nil {
							b.WriteRune(rune(code))
							i += 6
							continue
						}
					}
					switch s[i+1] {
					case 'n':
						b.WriteByte('\n')
					case 'r':
						b.WriteByte('\r')
					case 't':
						b.WriteByte('\t')
					default:
						b.WriteByte(s[i+1])
					}
					i += 2
					continue
				}
				if s[i] == quote {
					i++
					closed = true
					break
				}
				b.WriteByte(s[i])
				i++
			}
			if !closed {
				return nil, false
			}
			token = Token{Kind: TokenString, Literal: b.String()}
		} else {
			start := i
			for i < len(s) && s[i] != ',' && s[i] != ')' {
				if s[i] == '/' && i+1 < len(s) && (s[i+1] == '/' || s[i+1] == '*') {
					break
				}
				i++
			}
			literal := strings.TrimSpace(s[start:i])
			if literal == "" {
				return nil, false
			}
			if _, err := strconv.Atoi(literal); err == nil {
				token = Token{Kind: TokenNumber, Literal: literal}
			} else {
				token = Token{Kind: TokenRaw, Literal: literal}
			}
		}
		tokens = append(tokens, token)
		i = skipSpaceComments(s, i)
		if i >= len(s) {
			return nil, false
		}
		if s[i] == ',' {
			i++
			continue
		}
		if s[i] == ')' {
			return tokens, true
		}
		return nil, false
	}
}

func Str(tokens []Token, i int) (string, bool) {
	if i < 0 || i >= len(tokens) || tokens[i].Kind != TokenString {
		return "", false
	}
	return tokens[i].Literal, true
}

func Int(tokens []Token, i int) (int, bool) {
	if i < 0 || i >= len(tokens) || tokens[i].Kind != TokenNumber {
		return 0, false
	}
	v, err := strconv.Atoi(tokens[i].Literal)
	return v, err == nil
}
