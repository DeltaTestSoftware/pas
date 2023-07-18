package pas_test

import (
	"strings"
	"testing"

	"github.com/gonutz/check"

	"github.com/DeltaTestSoftware/pas"
)

func TestTokenize(t *testing.T) {
	checkTokenize := func(code string, want []pas.Token, wantSuccess bool) {
		t.Helper()

		// We always expect and EOF at the last position after the code.
		want = append(want, endOfFile(len(code)))

		have, err := pas.TokenizeString(code)
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
	tokenize := func(code string, want ...pas.Token) {
		t.Helper()
		checkTokenize(code, want, true)
	}
	fail := func(code string, want ...pas.Token) {
		t.Helper()
		checkTokenize(code, want, false)
	}
	utf8 := string([]byte{0xEF, 0xBB, 0xBF})

	tokenize("")
	tokenize(" ", space(0))
	tokenize(" \r\n\t", space(0))
	tokenize("// comment", comment(0))
	tokenize(" //", space(0), comment(1))
	tokenize("{}", comment(0))
	tokenize("{{}", comment(0))
	tokenize("{\r\n}", comment(0))
	fail("{", comment(0))
	tokenize("(**)", comment(0))
	tokenize("(*** )*)x", comment(0), word(8))
	tokenize("(* \t\r\n *)", comment(0))
	tokenize("(***)", comment(0))
	fail("(** )", comment(0))
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
		",;.:=()[]+-*/^><@",
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
		symbol(13),
		symbol(14),
		symbol(15),
		symbol(16),
	)
	tokenize(
		"<> < > <>",
		uneq(0),
		space(2),
		symbol(3),
		space(4),
		symbol(5),
		space(6),
		uneq(7),
	)
	tokenize(`''`, stringLiteral(0))
	tokenize(`'abc'`, stringLiteral(0))
	tokenize(`'' ''`, stringLiteral(0), space(2), stringLiteral(3))
	tokenize(
		utf8+` 'ä' `,
		utf8bom(),
		space(3),
		stringLiteral(4), // ä is 2 bytes.
		space(8),
	)
	tokenize(` '''' `, space(0), stringLiteral(1), space(5))
	tokenize(` 'quote '' is escaped'`, space(0), stringLiteral(1))
	fail(`'`, stringLiteral(0))
	tokenize("1 23 456", number(0), space(1), number(2), space(4), number(5))
	tokenize("$12 $ab $EF", number(0), space(3), number(4), space(7), number(8))
	fail("$", illegal(0))
	fail("$ ", illegal(0), space(1))
	fail("$x", illegal(0), word(1))
	tokenize("#13#10", character(0), character(3))
	tokenize(
		"#$10 #$a #$A",
		character(0),
		space(4),
		character(5),
		space(8),
		character(9),
	)
	fail("#", illegal(0))
	fail("# ", illegal(0), space(1))
	fail("##x", illegal(0), illegal(1), word(2))
	fail("#$", illegal(0))
	fail("#$ ", illegal(0), space(2))
	fail("##$x", illegal(0), illegal(1), word(3))
	bom := string([]byte{0xEF, 0xBB, 0xBF})
	tokenize(bom, utf8bom())
	tokenize(bom+" UTF8", utf8bom(), space(3), word(4))
	fail("&", word(0))
	fail("& ", word(0), space(1))
	tokenize("&begin 9", word(0), space(6), number(7))
	tokenize("&&begin 9", word(0), space(7), number(8))
	fail("&&9", word(0), number(2))
}

func endOfFile(offset int) pas.Token {
	return pas.Token{Type: pas.EOF, Offset: offset}
}

func illegal(offset int) pas.Token {
	return pas.Token{Type: pas.IllegalCharacter, Offset: offset}
}

func comment(offset int) pas.Token {
	return pas.Token{Type: pas.Comment, Offset: offset}
}

func space(offset int) pas.Token {
	return pas.Token{Type: pas.WhiteSpace, Offset: offset}
}

func word(offset int) pas.Token {
	return pas.Token{Type: pas.Word, Offset: offset}
}

func symbol(offset int) pas.Token {
	return pas.Token{Type: pas.Symbol, Offset: offset}
}

func stringLiteral(offset int) pas.Token {
	return pas.Token{Type: pas.String, Offset: offset}
}

func character(offset int) pas.Token {
	return pas.Token{Type: pas.Character, Offset: offset}
}

func number(offset int) pas.Token {
	return pas.Token{Type: pas.Number, Offset: offset}
}

func utf8bom() pas.Token {
	// The BOM always comes first, hence offset 0.
	return pas.Token{Type: pas.UTF8BOM, Offset: 0}
}

func uneq(offset int) pas.Token {
	return pas.Token{Type: pas.Unequal, Offset: offset}
}

func TestTokenizeErrors(t *testing.T) {
	failTokenize := func(code, wantErrorStart string) {
		t.Helper()

		_, err := pas.TokenizeString(code)
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
	// Non-ASCII character in ASCII file (has no UTF-8 BOM)
	failTokenize(string([]byte{128}), "1:1")
}
