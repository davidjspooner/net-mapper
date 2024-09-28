package mibtoken

type TokenType int

const (
	UNKNOWN TokenType = iota //should be silently discarded
	WHITESPACE
	COMMENT
	IDENT
	KEYWORD //entirely uppercase ( included macros )
	NUMBER
	STRING
	PUNCT
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
	case KEYWORD:
		return "KEYWORD"
	case NUMBER:
		return "NUMBER"
	case STRING:
		return "STRING"
	case PUNCT:
		return "PUNCT"
	case EOF:
		return "EOF"
	}
	return "UNKNOWN"
}

type Token struct {
	value    string
	position Position
}

func EOFToken(filename string) *Token {
	return &Token{position: *EOFPosition(filename)}
}

type Queue interface {
	Pop() (*Token, error)
	LookAhead(n int) (*Token, error)
	PopBlock(start, end string) (*List, error) //dont include start and end in list
	PopUntil(end string) (*List, error)        //dont include end in list
	PopExpected(elems ...string) error
	IsEOF() bool
	Source() *Position
	WrapError(err error) error
}

func (t *Token) String() string {
	return t.value
}

func (t *Token) IsText(s string) bool {
	return t.value == s
}

func (t Token) Source() *Position {
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
		return PUNCT
	case '"', '\'':
		return STRING
	case '_':
		return IDENT
	default:
		if c >= '0' && c <= '9' {
			return NUMBER
		}
		upperCount := 0
		lowerCount := 0
		for _, c := range t.value {
			if c >= 'A' && c <= 'Z' {
				upperCount++
			} else if c >= 'a' && c <= 'z' {
				lowerCount++
			}
		}
		if lowerCount == 0 && upperCount > 0 {
			return KEYWORD
		}
		if lowerCount > 0 {
			return IDENT
		}

		return PUNCT
	}
}

func (t *Token) WrapError(err error) error {
	return t.position.WrapError(err)
}

func (t *Token) Errorf(format string, args ...interface{}) error {
	return t.position.Errorf(format, args...)
}

func (t *Token) IsEOF() bool {
	return t.position.IsEOF()
}
