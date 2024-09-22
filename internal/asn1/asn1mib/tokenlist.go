package asn1mib

import (
	"io"
	"strings"
)

type TokenList struct {
	Filename string
	elements []*Token
}

var _ TokenQueue = (*TokenList)(nil)

func (tl TokenList) String() string {
	sb := &strings.Builder{}
	for n, t := range tl.elements {
		if n > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(t.String())
	}
	return sb.String()
}

func (tl *TokenList) Pop() (*Token, error) {
	tok, err := tl.LookAhead(0)
	if err != nil {
		return tok, err
	}
	err = tl.RemoveIndex(0)
	if err != nil {
		return nil, err
	}
	return tok, nil
}

func (tl *TokenList) LookAhead(n int) (*Token, error) {
	if n >= len(tl.elements) {
		return nil, EOFPosition(tl.Filename).WrapError(io.EOF)
	}
	return tl.elements[n], nil
}

func (tl *TokenList) PopBlock(start, end string) (*TokenList, error) {
	block := &TokenList{tl.Filename, nil}
	head, err := tl.LookAhead(0)
	if err != nil {
		return nil, err
	}
	if !head.IsText(start) {
		return nil, head.Errorf("expected %s", start)
	}
	level := 0

	for {
		t, err := tl.Pop()
		if err != nil {
			return nil, err
		}
		block.AppendTokens(t)
		if t.IsText(start) {
			level++
		}
		if t.IsText(end) {
			level--
			if level == 0 {
				block.elements = block.elements[1 : len(block.elements)-1]
				return block, nil
			}
		}
	}
}

func (tl *TokenList) PopUntil(end string) (*TokenList, error) {
	block := &TokenList{tl.Filename, nil}
	for {
		t, err := tl.Pop()
		if err != nil {
			return nil, err
		}
		if t.IsText(end) {
			return block, nil
		}
		block.AppendTokens(t)
	}
}

func (tl *TokenList) Length() int {
	return len(tl.elements)
}

func (tl *TokenList) IsEOF() bool {
	return tl.Length() == 0
}

func (tl *TokenList) Clone() *TokenList {
	clone := &TokenList{
		Filename: tl.Filename,
		elements: make([]*Token, len(tl.elements)),
	}
	copy(clone.elements, tl.elements)
	return clone
}

func (tl *TokenList) AppendTokens(tokens ...*Token) {
	tl.elements = append(tl.elements, tokens...)
}

func (tl *TokenList) AppendLists(tokens ...*TokenList) {
	for _, t := range tokens {
		tl.elements = append(tl.elements, t.elements...)
	}
}

func (tl *TokenList) PopExpected(elems ...string) error {
	for _, e := range elems {
		t, err := tl.Pop()
		if err != nil {
			return err
		}
		if !t.IsText(e) {
			return t.Errorf("expected %s", e)
		}
	}
	return nil
}

func (tl *TokenList) Source() *Position {
	if len(tl.elements) == 0 {
		return EOFPosition(tl.Filename)
	}
	return tl.elements[0].Source()
}

func (tl *TokenList) ForEach(f func(*Token) error) error {
	for _, t := range tl.elements {
		err := f(t)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tl *TokenList) RemoveBrackets(open, close string) error {
	if len(tl.elements) < 2 {
		return nil
	}
	if !tl.elements[0].IsText(open) {
		return tl.elements[0].Errorf("expected %s", open)
	}
	last := tl.elements[len(tl.elements)-1]
	if !last.IsText(close) {
		return last.Errorf("expected %s", close)
	}
	tl.elements = tl.elements[1 : len(tl.elements)-1]
	return nil
}

func (tl *TokenList) RemoveIndex(n int) error {
	if n >= len(tl.elements) {
		return io.EOF
	}
	if n+1 <= len(tl.elements) {
		copy(tl.elements[n:], tl.elements[n+1:])
	}
	tl.elements = tl.elements[:len(tl.elements)-1]
	return nil
}

func (tl *TokenList) Slice(start, end int) (*TokenList, error) {
	if start < 0 || start >= len(tl.elements) {
		return nil, tl.Source().Errorf("start index %d out of bounds [%d..%d]", start, 0, len(tl.elements))
	}
	if end < 0 || end > len(tl.elements) {
		return nil, tl.Source().Errorf("end index %d out of bounds [%d..%d]", end, 0, len(tl.elements))
	}
	if start > end {
		return nil, tl.Source().Errorf("start index %d is greater than end index %d", start, end)
	}
	tl2 := TokenList{
		Filename: tl.Filename,
		elements: make([]*Token, end-start),
	}
	copy(tl2.elements, tl.elements[start:end])
	return &tl2, nil
}
