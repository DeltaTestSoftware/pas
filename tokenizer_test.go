package pas

import (
	"strings"
	"testing"

	"github.com/gonutz/check"
)

func TestTokenize(t *testing.T) {
	checkTokenize := func(code string, want []Token, wantSuccess bool) {
		t.Helper()

		// We always expect and EOF at the last position after the code.
		want = append(want, endOfFile(len(code)))

		have, err := TokenizeString(code)
		if wantSuccess {
			check.Eq(t, err, nil)
		} else {
			check.Neq(t, err, nil)
		}
		for i := range want {
			wantString := want[i].String()
			haveString := "<nothing>"
			if i < len(have) {
				haveString = have[i].String()
			}
			check.Eq(t, haveString, wantString, "token ", i)
		}
	}
	tokenize := func(code string, want ...Token) {
		t.Helper()
		checkTokenize(code, want, true)
	}
	fail := func(code string, want ...Token) {
		t.Helper()
		checkTokenize(code, want, false)
	}

	tokenize("")
	tokenize(" ", space(0))
	tokenize(" \r\n\t", space(0))
	tokenize("// comment", comment(0))
	tokenize(" //", space(0), comment(1))
	fail(`"`, illegal(0))
	tokenize(
		"a A abc ABC _123",
		word(0),
		space(1),
		word(2),
		space(3),
		word(4),
		space(7),
		word(8),
		space(11),
		word(12),
	)
	tokenize(
		",;.:=()[]+-*/",
		symbol(0),
		symbol(1),
		symbol(2),
		symbol(3),
		symbol(4),
		symbol(5),
		symbol(6),
		symbol(7),
		symbol(8),
		symbol(9),
		symbol(10),
		symbol(11),
		symbol(12),
	)
	tokenize(`''`, stringLiteral(0))
	tokenize(`'abc'`, stringLiteral(0))
	tokenize(`'' ''`, stringLiteral(0), space(2), stringLiteral(3))
	tokenize(` 'ä' `, space(0), stringLiteral(1), space(5)) // ä is 2 bytes.
	tokenize(` '''' `, space(0), stringLiteral(1), space(5))
	tokenize(` 'quote '' is escaped'`, space(0), stringLiteral(1))
	fail(`'`, stringLiteral(0))
	tokenize("#13#10", character(0), character(3))
	fail("#", illegal(0))
	fail("##x", illegal(0), illegal(1), word(2))
	tokenize("1 23 456", number(0), space(1), number(2), space(4), number(5))
	bom := string([]byte{0xEF, 0xBB, 0xBF})
	tokenize(bom, utf8bom())
	tokenize(bom+" UTF8", utf8bom(), space(3), word(4))
}

func endOfFile(offset int) Token {
	return Token{Type: EOF, Offset: offset}
}

func illegal(offset int) Token {
	return Token{Type: IllegalCharacter, Offset: offset}
}

func comment(offset int) Token {
	return Token{Type: Comment, Offset: offset}
}

func space(offset int) Token {
	return Token{Type: WhiteSpace, Offset: offset}
}

func word(offset int) Token {
	return Token{Type: Word, Offset: offset}
}

func symbol(offset int) Token {
	return Token{Type: Symbol, Offset: offset}
}

func stringLiteral(offset int) Token {
	return Token{Type: String, Offset: offset}
}

func character(offset int) Token {
	return Token{Type: Character, Offset: offset}
}

func number(offset int) Token {
	return Token{Type: Number, Offset: offset}
}

func utf8bom() Token {
	// The BOM always comes first, hence offset 0.
	return Token{Type: UTF8BOM, Offset: 0}
}

func TestTokenizeErrors(t *testing.T) {
	failTokenize := func(code, wantErrorStart string) {
		t.Helper()

		_, err := TokenizeString(code)
		if err == nil {
			t.Fatal("no error")
		}

		msg := err.Error()
		if !strings.HasPrefix(msg, wantErrorStart) {
			t.Errorf("%q does not start with %q", msg, wantErrorStart)
		}
	}
	fail := func(code, wantErrorStart string) {
		t.Helper()
		bom := string([]byte{0xEF, 0xBB, 0xBF})
		failTokenize(code, wantErrorStart)
		failTokenize(bom+code, wantErrorStart)
	}

	fail(`"`, "1:1: ")
	fail(` "`, "1:2: ")
	fail("\n\"", "2:1: ")
	fail("\n \"", "2:2: ")
	fail(`'`, "1:1: ")
	fail("\n  '\n ", "2:3: ")
	fail("#", "1:1: ")
	fail("\n  #", "2:3: ")
	fail("#x", "1:1: ")
}
