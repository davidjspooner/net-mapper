package mibtoken

type TokenType int

const (
	UNKNOWN TokenType = iota //should be silently discarded
	WHITESPACE
	COMMENT
	IDENT
	NUMBER
	STRING
	SYMBOL
	EOF
)

func (t TokenType) String() string {
	switch t {
	case WHITESPACE:
		return "WHITESPACE"
	case COMMENT:
		return "COMMENT"
	case IDENT:
		return "IDENT"
	case NUMBER:
		return "NUMBER"
	case STRING:
		return "STRING"
	case SYMBOL:
		return "PUNCT"
	case EOF:
		return "EOF"
	}
	return "UNKNOWN"
}

type Token struct {
	value    string
	position Source
}

func New(value string, position Source) *Token {
	return &Token{value: value, position: position}
}

func EOFToken(filename string) *Token {
	return &Token{position: *EOFPosition(filename)}
}

func (t *Token) String() string {
	return t.value
}

func (t *Token) IsText(s string) bool {
	return t.value == s
}

func (t Token) Source() *Source {
	return &t.position
}

func (t Token) Type() TokenType {
	if t.position.IsEOF() {
		return EOF
	}
	if t.value == "" {
		return UNKNOWN
	}
	c := t.value[0]
	switch c {
	case ' ', '\t', '\n', '\r':
		return WHITESPACE
	case '-':
		if len(t.value) > 1 {
			if t.value[1] == '-' {
				return COMMENT
			}
			return NUMBER
		}
		return SYMBOL
	case '"', '\'':
		return STRING
	case '_':
		return IDENT
	default:
		if c >= '0' && c <= '9' {
			return NUMBER
		}
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			return IDENT
		}
		return SYMBOL
	}
}

func (t *Token) WrapError(err error) error {
	err = t.position.WrapError(err)
	return err
}

func (t *Token) Errorf(format string, args ...interface{}) error {
	return t.position.Errorf(format, args...)
}

func (t *Token) IsEOF() bool {
	return t.position.IsEOF()
}
