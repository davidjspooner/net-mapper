package asn1mib

type TokenType int

const (
	UNKNOWN TokenType = iota //should be silently discarded
	WHITESPACE
	COMMENT
	IDENT
	NUMBER
	STRING
	PUNCT
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
	case PUNCT:
		return "PUNCT"
	}
	return "UNKNOWN"
}

type Token struct {
	text   string
	source Position
}

func (t Token) String() string {
	return t.text
}

func (t Token) TokenIs(s string) bool {
	return t.text == s
}

func (t Token) Source() Position {
	return t.source
}

func (t Token) Type() TokenType {
	if t.text == "" {
		return UNKNOWN
	}
	c := t.text[0]
	switch c {
	case ' ', '\t', '\n', '\r':
		return WHITESPACE
	case '-':
		if len(t.text) > 1 {
			if t.text[1] == '-' {
				return COMMENT
			}
			return NUMBER
		}
		return PUNCT
	case '"', '\'':
		return STRING
	case '_':
		return IDENT
	default:
		if c >= '0' && c <= '9' {
			return NUMBER
		}
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return IDENT
		}
		return PUNCT
	}
}

func (t *Token) WrapError(err error) error {
	return t.source.WrapError(err)
}

func (t *Token) Errorf(format string, args ...interface{}) error {
	return t.source.Errorf(format, args...)
}
