package client

import (
	"errors"
	"strings"
	"unicode"
)

var (
	ErrMultipleStatements = errors.New("only a single statement can be used here")
	ErrEmptyQuery         = errors.New("query is empty")
)

// Statement is one semicolon-separated statement of a script.
type Statement struct {
	SQL string
	// Keywords are the first few words of the statement, upper-cased, with
	// comments skipped.
	Keywords []string
}

// SplitStatements splits a script on semicolons that are outside string
// literals, quoted identifiers, dollar-quoted bodies and comments. Statements
// that hold nothing but whitespace and comments are dropped.
func SplitStatements(script string) []Statement {
	var stmts []Statement
	start := 0
	flush := func(end int) {
		if st := newStatement(script[start:end]); st != nil {
			stmts = append(stmts, *st)
		}
		start = end + 1
	}

	for i := 0; i < len(script); {
		c := script[i]
		switch {
		case c == ';':
			flush(i)
			i++
		case c == '-' && i+1 < len(script) && script[i+1] == '-':
			i = skipLineComment(script, i)
		case c == '/' && i+1 < len(script) && script[i+1] == '*':
			i = skipBlockComment(script, i)
		case c == '\'':
			escapes := i > 0 && (script[i-1] == 'e' || script[i-1] == 'E') && (i < 2 || !isIdentChar(script[i-2]))
			i = skipQuoted(script, i, '\'', escapes)
		case c == '"':
			i = skipQuoted(script, i, '"', false)
		case c == '$':
			i = skipDollarQuoted(script, i)
		default:
			i++
		}
	}
	if start < len(script) {
		flush(len(script))
	}
	return stmts
}

// SingleStatement returns the only statement in script, or an error when the
// script is empty or holds more than one.
func SingleStatement(script string) (Statement, error) {
	stmts := SplitStatements(script)
	switch len(stmts) {
	case 0:
		return Statement{}, ErrEmptyQuery
	case 1:
		return stmts[0], nil
	default:
		return Statement{}, ErrMultipleStatements
	}
}

// IsTransactionControl reports whether the statement starts, ends or prepares
// a transaction. Savepoint commands are not included: they stay inside the
// current transaction.
func (s Statement) IsTransactionControl() bool {
	kw := s.Keywords
	if len(kw) == 0 {
		return false
	}
	switch kw[0] {
	case "BEGIN", "START", "COMMIT", "END", "ABORT":
		return true
	case "ROLLBACK":
		return len(kw) < 2 || kw[1] != "TO"
	case "PREPARE":
		return len(kw) > 1 && kw[1] == "TRANSACTION"
	}
	return false
}

func newStatement(sql string) *Statement {
	kw := leadingKeywords(sql, 3)
	if len(kw) == 0 {
		return nil
	}
	return &Statement{SQL: strings.TrimSpace(sql), Keywords: kw}
}

// leadingKeywords returns up to n leading words of sql, skipping comments.
func leadingKeywords(sql string, n int) []string {
	var words []string
	for i := 0; i < len(sql) && len(words) < n; {
		c := sql[i]
		switch {
		case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
			i = skipLineComment(sql, i)
		case c == '/' && i+1 < len(sql) && sql[i+1] == '*':
			i = skipBlockComment(sql, i)
		case isIdentChar(c):
			j := i
			for j < len(sql) && isIdentChar(sql[j]) {
				j++
			}
			words = append(words, strings.ToUpper(sql[i:j]))
			i = j
		case unicode.IsSpace(rune(c)):
			i++
		default:
			// Anything else (a parenthesis, a quote) means the words so far are
			// all the keywords there are.
			if len(words) > 0 {
				return words
			}
			words = append(words, string(c))
			return words
		}
	}
	return words
}

func isIdentChar(c byte) bool {
	return c == '_' || c >= 0x80 || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func skipLineComment(s string, i int) int {
	for i < len(s) && s[i] != '\n' {
		i++
	}
	return i
}

// skipBlockComment skips a /* */ comment; PostgreSQL allows them to nest.
func skipBlockComment(s string, i int) int {
	depth := 0
	for i < len(s) {
		switch {
		case s[i] == '/' && i+1 < len(s) && s[i+1] == '*':
			depth++
			i += 2
		case s[i] == '*' && i+1 < len(s) && s[i+1] == '/':
			depth--
			i += 2
			if depth == 0 {
				return i
			}
		default:
			i++
		}
	}
	return i
}

// skipQuoted skips a literal opened by quote at s[i]. A doubled quote is an
// escaped quote; with backslashEscapes (E” strings) a backslash escapes the
// next byte.
func skipQuoted(s string, i int, quote byte, backslashEscapes bool) int {
	i++
	for i < len(s) {
		switch {
		case backslashEscapes && s[i] == '\\':
			i += 2
		case s[i] == quote:
			if i+1 < len(s) && s[i+1] == quote {
				i += 2
				continue
			}
			return i + 1
		default:
			i++
		}
	}
	return i
}

// skipDollarQuoted skips a $tag$ ... $tag$ body starting at s[i]. A '$' that
// does not open one (a positional parameter like $1) is skipped on its own.
func skipDollarQuoted(s string, i int) int {
	if i > 0 && isIdentChar(s[i-1]) {
		return i + 1
	}
	j := i + 1
	for j < len(s) && s[j] != '$' {
		c := s[j]
		if !(c == '_' || c >= 0x80 || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (j > i+1 && c >= '0' && c <= '9')) {
			return i + 1
		}
		j++
	}
	if j >= len(s) {
		return i + 1
	}
	tag := s[i : j+1]
	end := strings.Index(s[j+1:], tag)
	if end < 0 {
		return len(s)
	}
	return j + 1 + end + len(tag)
}
