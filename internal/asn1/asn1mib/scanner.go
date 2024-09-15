package asn1mib

import (
	"bufio"
	"fmt"
	"io"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

type TokenType int

const (
	WHITESPACE TokenType = iota //should be silently discarded
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

type splitterFunc func(scanner *Scanner, data []byte, atEOF bool) (advance int, token []byte, err error)

var splitterFuncIndex [128]splitterFunc

func init() {
	for _, n := range " \t" {
		splitterFuncIndex[n] = splitSpace
	}
	for _, n := range "\n\r" {
		splitterFuncIndex[n] = splitEOL
	}
	for _, n := range "0123456789" {
		splitterFuncIndex[n] = splitNumber
	}
	for _, n := range "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		splitterFuncIndex[n] = splitIdent
	}
	for _, n := range "\"'" {
		splitterFuncIndex[n] = splitString
	}
	for _, n := range "-{}[](),:;.|" {
		splitterFuncIndex[n] = splitPunct
	}
}

type Position struct {
	Filename string
	Line     int
	Column   int
}

type Scanner struct {
	inner        *bufio.Scanner
	position     Position
	nextPosition Position
	tokenType    TokenType
	skip         map[TokenType]bool
}

type ScannerOption func(scanner *Scanner) error

func WithSkip(tokenTypes ...TokenType) ScannerOption {
	return func(s *Scanner) error {
		for _, t := range tokenTypes {
			if t != WHITESPACE && t != COMMENT {
				panic("only WHITESPACE and COMMENT can be skipped")
			}
			s.skip[t] = true
		}
		return nil
	}
}
func WithFilename(filename string) ScannerOption {
	return func(s *Scanner) error {
		s.position.Filename = filename
		s.nextPosition.Filename = filename
		return nil
	}
}

func NewScanner(r io.Reader, options ...ScannerOption) (*Scanner, error) {
	s := &Scanner{
		inner: bufio.NewScanner(r),
		position: Position{
			Line:   1,
			Column: 1,
		},
		skip: make(map[TokenType]bool),
	}
	s.nextPosition = s.position
	s.inner.Split(s.split)
	for _, opt := range options {
		err := opt(s)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Scanner) split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	s.position = s.nextPosition
	if len(data) == 0 {
		return 0, nil, nil
	}
	if data[0] > byte(len(splitterFuncIndex)) {
		return 0, nil, fmt.Errorf("invalid character: '%c'", data[0])
	}
	if f := splitterFuncIndex[data[0]]; f != nil {
		advance, token, err := f(s, data, atEOF)
		return advance, token, err
	}
	return 0, nil, fmt.Errorf("invalid character: '%c'", data[0])
}

func (s *Scanner) Scan() bool {
	ok := s.inner.Scan()
	for ok {
		if !s.skip[s.tokenType] {
			break
		}
		ok = s.inner.Scan()
	}
	return ok
}

func (s *Scanner) Token() (TokenType, string) {
	return s.tokenType, s.inner.Text()
}

func (s *Scanner) Err() error {
	return s.inner.Err()
}

func (s *Scanner) Position() Position {
	return s.position
}

func (s *Scanner) ScanIdent() (string, error) {
	if !s.Scan() {
		return "", s.Err()
	}
	actualType, actualText := s.Token()
	if actualType != IDENT {
		return "", asn1core.NewUnexpectedError(IDENT, actualType, "token type")
	}
	return actualText, nil
}

func (s *Scanner) ScanAndExpect(expectedTexts ...string) error {
	for _, expectedText := range expectedTexts {
		if !s.Scan() {
			return s.Err()
		}
		_, actualText := s.Token()
		if actualText != expectedText {
			return asn1core.NewUnexpectedError(expectedText, actualText, "token")
		}
	}
	return nil
}

func splitSpace(s *Scanner, data []byte, atEOF bool) (advance int, token []byte, err error) {
	s.tokenType = WHITESPACE
	for i, b := range data {
		if b != ' ' && b != '\t' {
			s.nextPosition.Column += i
			return i, data[:1], nil
		}
	}
	if atEOF {
		s.nextPosition.Column += len(data)
		return len(data), data, nil
	}
	return 0, nil, nil //need more data
}

func splitIdent(s *Scanner, data []byte, atEOF bool) (advance int, token []byte, err error) {
	s.tokenType = IDENT

	for i, b := range data {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || (b == '_') || (b == '-') {
			continue
		}
		s.nextPosition.Column += i
		return i, data[:i], nil
	}
	if atEOF {
		s.nextPosition.Column += len(data)
		return len(data), data, nil
	}
	return 0, nil, nil //need more data
}

func splitNumber(s *Scanner, data []byte, atEOF bool) (advance int, token []byte, err error) {
	s.tokenType = NUMBER
	for i, b := range data {
		if i == 0 {
			continue
		}
		if b >= '0' && b <= '9' {
			continue
		}
		s.nextPosition.Column += i
		return i, data[:i], nil
	}
	if atEOF {
		s.nextPosition.Column += len(data)
		return len(data), data, nil
	}
	return 0, nil, nil //need more data
}

func splitString(s *Scanner, data []byte, atEOF bool) (advance int, token []byte, err error) {
	s.tokenType = STRING
	first := data[0]
	pos := s.position
	for i := 1; i < len(data); i++ {
		b := data[i]
		if b == '\n' {
			if data[i-1] != '\r' {
				pos.Line++
				pos.Column = 1
			}
		} else if b == '\r' {
			if data[i+1] != '\n' {
				pos.Line++
				pos.Column = 1
			}
		} else if b == '\\' {
			i++
			pos.Column += 1
		} else if b == first {
			pos.Column += 1
			return i + 1, data[:i+1], nil
		} else {
			pos.Column += 1
		}
	}
	if atEOF {
		return 0, nil, fmt.Errorf("unterminated string")
	}

	return 0, nil, nil //need more data
}

func splitPunct(s *Scanner, data []byte, atEOF bool) (advance int, token []byte, err error) {
	s.tokenType = PUNCT
	switch data[0] {
	case '-':
		if len(data) < 2 && !atEOF {
			return 0, nil, nil //read some more
		}
		if (len(data) >= 2) && (data[1] == '-') {
			s.tokenType = COMMENT
			//read to EOL
			for i, b := range data {
				if b == '\n' || b == '\r' {
					s.nextPosition.Column += i
					return i, data[:i], nil
				}
			}
			if atEOF {
				s.nextPosition.Column += len(data)
				return len(data), data, nil
			}
			return 0, nil, nil //need more data
		}
		if (len(data) >= 2) && (data[1] >= '0' && data[1] <= '9') {
			return splitNumber(s, data, atEOF)
		}
	case ':':
		if len(data) < 3 && !atEOF {
			return 0, nil, nil //read some more
		}
		if len(data) >= 3 && data[1] == ':' && data[2] == '=' {
			return 3, data[:3], nil
		}
	}
	return 1, data[:1], nil
}

func splitEOL(s *Scanner, data []byte, atEOF bool) (advance int, token []byte, err error) {
	s.tokenType = WHITESPACE
	s.nextPosition.Column = 1
	s.nextPosition.Line++
	if len(data) > 1 && (data[0] != data[1]) && (data[1] == '\n' || data[1] == '\r') {
		return 2, data[:2], nil
	}
	return 1, data[:1], nil
}

func ExtractString(original string) (string, error) {
	if len(original) < 2 {
		return "", fmt.Errorf("string too short")
	}
	if original[0] != original[len(original)-1] {
		return "", fmt.Errorf("string not terminated")
	}
	output := original[1 : len(original)-1]
	for i := 0; i < len(output); i++ {
		switch output[i] {
		case '\\':
			if i+1 >= len(output) {
				return "", fmt.Errorf("invalid escape sequence")
			}
			switch output[i+1] {
			case 'n':
				output = output[:i] + "\n" + output[i+2:]
			case 'r':
				output = output[:i] + "\r" + output[i+2:]
			case 't':
				output = output[:i] + "\t" + output[i+2:]
			case '\\':
				output = output[:i] + "\\" + output[i+2:]
			case '\'':
				output = output[:i] + "'" + output[i+2:]
			case '"':
				output = output[:i] + "\"" + output[i+2:]
			default:
				return "", fmt.Errorf("invalid escape sequence")
			}
		case '\n', '\r':
			base := i
			for i < len(output) {
				b := output[i]
				if b != '\n' && b != '\r' && b != ' ' && b != '\t' {
					break
				}
				i++
			}
			output = output[:base] + " " + output[i:]
		default:
			continue
		}
	}
	return output, nil
}
