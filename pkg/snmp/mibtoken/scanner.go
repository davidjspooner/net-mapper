package mibtoken

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
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
	queue        List
	nextPosition Position
	skip         map[TokenType]bool
	ctx          context.Context
}

var _ Queue = &Scanner{}

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
func WithSource(filename string) ScannerOption {
	return func(s *Scanner) error {
		s.nextPosition.Filename = filename
		s.queue.Filename = filename
		return nil
	}
}

func WithContext(ctx context.Context) ScannerOption {
	return func(s *Scanner) error {
		s.ctx = ctx
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
		ctx:  context.Background(),
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
	for len(s.queue.elements) < desired {
		pos := s.nextPosition
		if !s.inner.Scan() {
			break
		}
		tok := Token{
			value:    s.inner.Text(),
			position: pos,
		}
		if s.skip[tok.Type()] {
			continue
		}

		last := len(s.queue.elements) - 1
		merged := false
		if last >= 0 {
			candidate := s.queue.elements[last].String() + " " + tok.String()
			if slices.Contains(specialTokens, candidate) {
				s.queue.elements[last].value = candidate
				merged = true
			}
		}
		if !merged {
			s.queue.AppendTokens(&tok)
		}
	}

	return len(s.queue.elements)
}

func (s *Scanner) Pop() (*Token, error) {
	tok, err := s.LookAhead(0)
	if err == nil {
		s.queue.RemoveIndex(0)
	}
	return tok, err
}
func (s *Scanner) LookAhead(n int) (*Token, error) {
	if s.ctx.Err() != nil {
		return nil, s.ctx.Err()
	}
	s.refillQueue(n + 2)
	return s.queue.LookAhead(n)
}

func (s *Scanner) IsText(text string) bool {
	if s.refillQueue(1) == 0 {
		return false
	}
	return s.queue.elements[0].IsText(text)
}

func (s *Scanner) Err() error {
	return s.inner.Err()
}

func (s *Scanner) PopType(tType TokenType) (*Token, error) {
	actual, err := s.Pop()
	if err == nil {
		if actual.Type() != tType {
			err = actual.WrapError(asn1core.NewUnexpectedError(tType, actual.Type(), "token type"))
		}
	}
	return actual, err
}

func (s *Scanner) PopExpected(expectedTexts ...string) error {
	for n, expectedText := range expectedTexts {
		s.refillQueue(len(expectedTexts) - n)
		actual, err := s.Pop()
		if err != nil {
			return err
		}
		if actual.String() != expectedText {
			return asn1core.NewUnexpectedError(expectedText, actual.String(), "token")
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
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || (b == '_') || (b == '-') || (b == '.') {
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

func UnquoteString(t *Token) (string, error) {
	if t.Type() != STRING {
		return "", t.Errorf("not a string")
	}
	original := t.String()
	if len(original) < 2 {
		return "", t.Errorf("string too short")
	}
	if original[0] != original[len(original)-1] {
		return "", t.Errorf("string not terminated")
	}
	output := original[1 : len(original)-1]
	for i := 0; i < len(output); i++ {
		switch output[i] {
		case '\\':
			if i+1 >= len(output) {
				return "", t.Errorf("invalid escape sequence")
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
				return "", t.Errorf("invalid escape sequence")
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

func (s *Scanner) PopUntil(text string) (*List, error) {
	tokens := &List{
		Filename: s.nextPosition.Filename,
	}
	for {
		tok, err := s.Pop()
		if err != nil {
			return nil, err
		}
		if tok.IsText(text) {
			return tokens, nil
		}
		tokens.AppendTokens(tok)
	}
}

func (s *Scanner) Errorf(format string, args ...interface{}) error {
	return s.Source().Errorf(format, args...)
}
func (s *Scanner) WrapError(err error) error {
	return s.Source().WrapError(err)
}

func (s *Scanner) PopBlock(start, end string) (*List, error) {
	level := 0
	block := &List{Filename: s.nextPosition.Filename}
	head, err := s.LookAhead(0)
	if err != nil {
		return nil, err
	}
	if !head.IsText(start) {
		return nil, head.Errorf("expected %s but got %s", start, head.String())
	}
	for {
		tok, err := s.Pop()
		if err != nil {
			return nil, err
		}
		block.AppendTokens(tok)
		if tok.IsText(start) {
			level++
		} else if tok.String() == end {
			level--
			if level == 0 {
				block.elements = block.elements[1 : len(block.elements)-1]
				return block, nil
			}
		}
	}
}

func (s *Scanner) Source() *Position {
	s.refillQueue(1)
	return s.queue.Source()
}

func (s *Scanner) IsEOF() bool {
	s.refillQueue(1)
	return s.queue.IsEOF()
}
