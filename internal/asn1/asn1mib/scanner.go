package asn1mib

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
	"golang.org/x/exp/constraints"
)

type splitterFunc func(scanner *Scanner, data []byte, atEOF bool) (advance int, token []byte, err error)

var splitterFuncIndex [128]splitterFunc

const Object_Identifier = "OBJECT IDENTIFIER"
const Octet_String = "OCTET STRING"

var specialTokens = []string{Object_Identifier, Octet_String, "SEQUENCE OF", "SET OF", "TYPE NOTATION", "VALUE NOTATION"}

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

func (p *Position) String() string {
	sb := strings.Builder{}
	if p.Filename != "" {
		sb.WriteString(p.Filename)
	}
	if p.Line > 0 || p.Column > 0 {
		if p.Filename != "" {
			sb.WriteString(" ")
		}
		sb.WriteString(fmt.Sprintf("[ln %d col %d]", p.Line, p.Column))
	}
	return sb.String()
}

type ScannerError struct {
	Position Position
	Err      error
}

func (se *ScannerError) Error() string {
	return fmt.Sprintf("%s: %s", se.Position.String(), se.Err)
}

func (se *ScannerError) Unwrap() error {
	return se.Err
}

func (p *Position) Errorf(format string, args ...interface{}) error {
	return p.WrapError(fmt.Errorf(format, args...))
}

func (p *Position) WrapError(err error) error {
	return &ScannerError{
		Position: *p,
		Err:      err,
	}
}

type Scanner struct {
	inner        *bufio.Scanner
	queue        TokenList
	nextPosition Position
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
		s.nextPosition.Filename = filename
		return nil
	}
}

func NewScanner(r io.Reader, options ...ScannerOption) (*Scanner, error) {
	s := &Scanner{
		inner: bufio.NewScanner(r),
		nextPosition: Position{
			Line:   1,
			Column: 1,
		},
		skip: make(map[TokenType]bool),
	}
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

func Min[T constraints.Ordered](a ...T) T {
	min := a[0]
	for _, v := range a {
		if v < min {
			min = v
		}
	}
	return min
}

func Max[T constraints.Ordered](a ...T) T {
	max := a[0]
	for _, v := range a {
		if v > max {
			max = v
		}
	}
	return max
}

func (s *Scanner) refillQueue(desired int) int {
	desired = Max(desired, 16)
	for len(s.queue) < desired {
		pos := s.nextPosition
		if !s.inner.Scan() {
			break
		}
		tok := Token{
			text:   s.inner.Text(),
			source: pos,
		}
		if s.skip[tok.Type()] {
			continue
		}

		last := len(s.queue) - 1
		merged := false
		if last >= 0 {
			candidate := s.queue[last].String() + " " + tok.String()
			if slices.Contains(specialTokens, candidate) {
				s.queue[last].text = candidate
				merged = true
			}
		}
		if !merged {
			if tok.text == "NOTATION" {
				print("NOTATION!!!!!")
			}
			s.queue = append(s.queue, tok)
		}
	}

	return len(s.queue)
}
func (s *Scanner) Scan() bool {
	if len(s.queue) > 0 {
		//pop the first token off the queue
		s.queue.RemoveHead()
	}
	s.refillQueue(16)
	for len(s.queue) > 0 {
		tType := s.queue[0].Type()
		if !s.skip[tType] {
			break
		}
		//pop the first token off the queue
		s.queue.RemoveHead()
		s.refillQueue(16)
	}
	if len(s.queue) != 0 {
		return true
	}
	return false
}
func (s *Scanner) Pop() Token {
	token := s.LookAhead(0)
	s.queue.DeleteIndex(0)
	return token
}
func (s *Scanner) LookAhead(n int) Token {
	s.refillQueue(n + 2)
	if len(s.queue) < n {
		return Token{text: ""}
	}
	return s.queue[n]
}

func (s *Scanner) TokenIs(text string) bool {
	if len(s.queue) == 0 {
		return false
	}
	return s.queue[0].TokenIs(text)
}

func (s *Scanner) Err() error {
	return s.inner.Err()
}

func (s *Scanner) PopIdent() (Token, error) {
	actual := s.LookAhead(0)
	if actual.Type() != IDENT {
		return Token{}, actual.WrapError(asn1core.NewUnexpectedError(IDENT, actual.Type(), "token type"))
	}
	s.Scan()
	return actual, nil
}

func (s *Scanner) PopExpected(expectedTexts ...string) error {
	for _, expectedText := range expectedTexts {
		actual := s.LookAhead(0)
		if actual.String() != expectedText {
			return asn1core.NewUnexpectedError(expectedText, actual.String(), "token")
		}
		if !s.Scan() {
			return s.Err()
		}
	}
	return nil
}

func splitSpace(s *Scanner, data []byte, atEOF bool) (advance int, token []byte, err error) {
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
	first := data[0]
	for i := 1; i < len(data); i++ {
		b := data[i]
		if b == '\n' {
			if data[i-1] != '\r' {
				s.nextPosition.Line++
				s.nextPosition.Column = 1
			}
		} else if b == '\r' {
			if data[i+1] != '\n' {
				s.nextPosition.Line++
				s.nextPosition.Column = 1
			}
		} else if b == '\\' {
			i++
			s.nextPosition.Column += 1
		} else if b == first {
			s.nextPosition.Column += 1
			return i + 1, data[:i+1], nil
		} else {
			s.nextPosition.Column += 1
		}
	}
	if atEOF {
		return 0, nil, fmt.Errorf("unterminated string")
	}

	return 0, nil, nil //need more data
}

func splitPunct(s *Scanner, data []byte, atEOF bool) (advance int, token []byte, err error) {
	switch data[0] {
	case '-':
		if len(data) < 2 && !atEOF {
			return 0, nil, nil //read some more
		}
		if (len(data) >= 2) && (data[1] == '-') {
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

func (s *Scanner) ScanAllTokens() (TokenList, error) {
	tokens := TokenList{s.LookAhead(0)}
	for s.Scan() {
		tokens = append(tokens, s.LookAhead(0))
	}
	return tokens, s.Err()
}

func (s *Scanner) PopUntil(text string) (TokenList, error) {
	startPosition := *s.position()
	tokens := TokenList{
		s.LookAhead(0),
	}
	for {
		tok := s.Pop()
		if tok.TokenIs("") {
			break
		}
		tokens = append(tokens, tok)
		if tok.TokenIs(text) {
			return tokens, nil
		}
	}
	err := s.Err()
	if err == nil {
		err = startPosition.Errorf("looking for %q but reached end of file", text)
	}
	return nil, startPosition.WrapError(err)
}

func (s *Scanner) position() *Position {
	if len(s.queue) > 0 {
		return &s.queue[0].source
	}
	return &s.nextPosition
}

func (s *Scanner) Errorf(format string, args ...interface{}) error {
	return s.position().Errorf(format, args...)
}
func (s *Scanner) WrapError(err error) error {
	return s.position().WrapError(err)
}

func (s *Scanner) PopBlock(level int, start, end string) (TokenList, error) {
	block := TokenList{}
	for {
		tok := s.Pop()
		block = append(block, tok)
		if tok.String() == start {
			level++
		} else if tok.String() == end {
			level--
			if level == 0 {
				break
			}
		}
	}
	if level != 0 {
		return nil, s.Errorf("unterminated %q", start)
	}
	return block, nil
}
