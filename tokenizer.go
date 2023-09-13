package pas

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func TokenizeBytes(code []byte) ([]Token, error) {
	return tokenize(code)
}

func TokenizeString(code string) ([]Token, error) {
	return tokenize([]byte(code))
}

type Token struct {
	Type   TokenType
	Offset int // Byte offset into the code.
}

type TokenType int

const (
	EOF TokenType = iota
	IllegalCharacter
	UTF8BOM
	WhiteSpace
	Comment
	Word      // E.g. Count, I, begin, procedure
	Symbol    // Single letters like ; , + -
	String    // Unescaped and in single quotes, e.g. 'this is a ''string''.'
	Character // E.g. #13 #$D but NOT single-letter strings like 'A'.
	Number    // E.g. 5, 123, -89, $FF00
	Unequal   // Operator <>
)

func tokenize(code []byte) ([]Token, error) {
	t := newTokenizer(code)
	t.tokenizeAll()
	return t.tokens, t.err
}

func newTokenizer(code []byte) *tokenizer {
	return &tokenizer{code: code}
}

type tokenizer struct {
	code    []byte
	bomSize int
	start   int // Start of current token.
	end     int // The position right before pos.
	pos     int
	cur     rune
	tokens  []Token
	err     error
}

const eof = utf8.RuneError

func (t *tokenizer) next() {
	t.end = t.pos
	var size int
	t.cur, size = utf8.DecodeRune(t.code[t.pos:])
	t.pos += size
}

func (t *tokenizer) tokenizeAll() {
	if !t.checkFileFormat() {
		return
	}
	t.startOrUTF8BOM()
	for t.cur != eof {
		if isWhiteSpace(t.cur) {
			t.whiteSpace()
		} else if t.cur == '/' {
			t.divisionOrLineComment()
		} else if t.cur == '(' {
			t.openParenOrMultiLineComment()
		} else if t.cur == '{' {
			t.multiLineComment()
		} else if isWordStart(t.cur) {
			t.word()
		} else if t.cur == '&' {
			t.escapedWord()
		} else if unicode.IsDigit(t.cur) {
			t.number()
		} else if t.cur == '<' {
			t.unequalOrLessThan()
		} else if isSymbol(t.cur) {
			t.symbol()
		} else if t.cur == '\'' {
			t.stringLiteral()
		} else if t.cur == '#' {
			t.character()
		} else if t.cur == '$' {
			t.hexNumber()
		} else {
			t.illegalCharacter()
		}
	}
	t.eof()
}

var utf8bom = []byte{0xEF, 0xBB, 0xBF}

func (t *tokenizer) checkFileFormat() bool {
	isUTF8 := bytes.HasPrefix(t.code, utf8bom)
	isAscii := !isUTF8
	if isAscii {
		for i, b := range t.code {
			if b >= 128 {
				t.failAtf(i, "illegal character code(%d) in ASCII file "+
					"(please use UTF-8 encoding instead)", b)
				return false
			}
		}
	}
	return true
}

func (t *tokenizer) startOrUTF8BOM() {
	if bytes.HasPrefix(t.code, utf8bom) {
		t.bomSize = 3
		t.next()
		t.next()
		t.emit(UTF8BOM)
	} else {
		t.next()
	}
}

func (t *tokenizer) emit(typ TokenType) {
	t.tokens = append(t.tokens, Token{Type: typ, Offset: t.start})
	t.start = t.end
}

func isWhiteSpace(r rune) bool {
	return r == ' ' || r == '\r' || r == '\n' || r == '\t'
}

func (t *tokenizer) whiteSpace() {
	for isWhiteSpace(t.cur) {
		t.next()
	}
	t.emit(WhiteSpace)
}

func isWordStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func (t *tokenizer) word() {
	for unicode.IsLetter(t.cur) || unicode.IsDigit(t.cur) || t.cur == '_' {
		t.next()
	}
	t.emit(Word)
}

func (t *tokenizer) escapedWord() {
	t.next() // Skip '&'.
	for t.cur == '&' {
		t.next()
	}
	if isWordStart(t.cur) {
		t.next()
		for unicode.IsLetter(t.cur) || unicode.IsDigit(t.cur) || t.cur == '_' {
			t.next()
		}
	} else {
		t.failf("invalid word after ampersand")
	}
	t.emit(Word)
}

func (t *tokenizer) number() {
	t.next()
	for unicode.IsDigit(t.cur) {
		t.next()
	}
	t.emit(Number)
}

func isSymbol(r rune) bool {
	return strings.ContainsRune(",;.:=()[]+-*/^><@", r)
}

func (t *tokenizer) unequalOrLessThan() {
	t.next()
	if t.cur == '>' {
		t.next()
		t.emit(Unequal)
	} else {
		t.emit(Symbol)
	}
}

func (t *tokenizer) symbol() {
	t.next()
	t.emit(Symbol)
}

func (t *tokenizer) stringLiteral() {
	t.next() // Skip opening quote.
	for {
		if t.cur == eof {
			if t.err == nil {
				t.failf("unclosed string literal")
			}
			break
		}
		if t.cur == '\'' {
			t.next() // Skip closing quote.
			if t.cur != '\'' {
				break
			}
		}
		t.next()
	}
	t.emit(String)
}

func (t *tokenizer) character() {
	t.next() // Skip '#'.
	validNumber := unicode.IsDigit
	if t.cur == '$' {
		t.next() // Skip '$'.
		validNumber = isHexDigit
	}
	if validNumber(t.cur) {
		t.next()
		for validNumber(t.cur) {
			t.next()
		}
		t.emit(Character)
	} else {
		if t.err == nil {
			t.failf("missing number in character")
		}
		t.emit(IllegalCharacter)
	}
}

func (t *tokenizer) hexNumber() {
	t.next() // Skip '$'.
	if isHexDigit(t.cur) {
		t.next()
		for isHexDigit(t.cur) {
			t.next()
		}
		t.emit(Number)
	} else {
		if t.err == nil {
			t.failf("missing hex digit in hex number")
		}
		t.emit(IllegalCharacter)
	}
}

func isHexDigit(r rune) bool {
	return unicode.IsDigit(r) ||
		'a' <= r && r <= 'f' ||
		'A' <= r && r <= 'F'
}

func (t *tokenizer) illegalCharacter() {
	// We report the first ever error that occurs while tokenizing to the end.
	// If we have no error yet, we create it.
	if t.err == nil {
		t.failf("illegal character %q", t.cur)
	}
	t.next()
	t.emit(IllegalCharacter)
}

func (t *tokenizer) failf(format string, a ...any) {
	t.failAtf(t.start, format, a...)
}

func (t *tokenizer) failAtf(offset int, format string, a ...any) {
	line, col := t.positionAtOffset(offset)
	t.err = fmt.Errorf("%d:%d: %s", line, col, fmt.Sprintf(format, a...))
}

// positionAtOffset converts the given offset to line and column numbers, both
// starting at 1.
func (t *tokenizer) positionAtOffset(offset int) (line, col int) {
	prefix := t.code[:offset]
	line = 1 + bytes.Count(prefix, []byte{'\n'})
	// Note that this works on the first line, because bytes.LastIndexByte
	// returns -1 in that case.
	col = len(prefix) - bytes.LastIndexByte(prefix, '\n')
	if line == 1 {
		col -= t.bomSize
	}
	return line, col
}

func (t *tokenizer) divisionOrLineComment() {
	t.next() // Skip first slash.
	if t.cur == '/' {
		t.next() // Skip second slash.
		t.lineComment()
	} else {
		t.emit(Symbol)
	}
}

func (t *tokenizer) lineComment() {
	for !(t.cur == '\n' || t.cur == eof) {
		t.next()
	}
	t.emit(Comment)
}

func (t *tokenizer) openParenOrMultiLineComment() {
	t.next() // Skip opening '('.
	if t.cur == '*' {
		t.next() // Skip '*'.
		lastWasClosingStar := false
		for t.cur != eof {
			if lastWasClosingStar && t.cur == ')' {
				t.next() // Skip closing ')'.
				t.emit(Comment)
				return
			}
			lastWasClosingStar = t.cur == '*'
			t.next()
		}
		t.failf("unclosed comment")
		t.emit(Comment)
	} else {
		t.emit(Symbol)
	}
}

func (t *tokenizer) multiLineComment() {
	for !(t.cur == '}' || t.cur == eof) {
		t.next()
	}
	if t.cur == eof {
		t.failf("unclosed comment")
	}
	t.next() // Skip closing '}'.
	t.emit(Comment)
}

func (t *tokenizer) eof() {
	t.emit(EOF)
}

func (t Token) String() string {
	return fmt.Sprintf("%s at offset %d", t.Type, t.Offset)
}

func (t TokenType) String() string {
	switch t {
	case EOF:
		return "EOF"
	case IllegalCharacter:
		return "IllegalCharacter"
	case UTF8BOM:
		return "UTF8BOM"
	case WhiteSpace:
		return "WhiteSpace"
	case Comment:
		return "Comment"
	case Word:
		return "Word"
	case Symbol:
		return "Symbol"
	case String:
		return "String"
	case Character:
		return "Character"
	case Number:
		return "Number"
	case Unequal:
		return "Unequal"
	}
	return fmt.Sprintf("unknown TokenType(%d)", int(t))
}
