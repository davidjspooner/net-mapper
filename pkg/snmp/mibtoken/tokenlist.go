package mibtoken

import (
	"io"
	"strings"
)

type List struct {
	source   Source
	elements []*Token
}

var _ Reader = (*List)(nil)

func (tl List) String() string {
	sb := &strings.Builder{}
	for n, t := range tl.elements {
		if n > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(t.String())
	}
	return sb.String()
}

func (tl *List) Pop() (*Token, error) {
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

func (tl *List) LookAhead(n int) (*Token, error) {
	if n >= len(tl.elements) {
		return nil, EOFPosition(tl.source.Filename).WrapError(io.EOF)
	}
	return tl.elements[n], nil
}

func (tl *List) ReadBlock(start, end string) (*List, error) {
	block := &List{*tl.Source(), nil}
	head, err := tl.LookAhead(0)
	if err != nil {
		return nil, err
	}
	if !head.IsText(start) {
		return nil, head.Errorf("expected %s but got %s", start, head.String())
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

func (tl *List) ReadUntil(end string) (*List, error) {
	block := &List{*tl.Source(), nil}
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

func (tl *List) Length() int {
	return len(tl.elements)
}

func (tl *List) IsEOF() bool {
	return tl.Length() == 0
}

func (tl *List) Clone() *List {
	clone := &List{
		source:   *tl.Source(),
		elements: make([]*Token, len(tl.elements)),
	}
	copy(clone.elements, tl.elements)
	return clone
}

func (tl *List) AppendTokens(tokens ...*Token) {
	tl.elements = append(tl.elements, tokens...)
}

func (tl *List) AppendLists(tokens ...*List) {
	for _, t := range tokens {
		tl.elements = append(tl.elements, t.elements...)
	}
}

func (tl *List) ReadExpected(elems ...string) error {
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

func (tl *List) Source() *Source {
	if len(tl.elements) == 0 {
		return EOFPosition(tl.Source().Filename)
	}
	return tl.elements[0].Source()
}

func (tl *List) ForEach(f func(*Token) error) error {
	for _, t := range tl.elements {
		err := f(t)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tl *List) RemoveBrackets(open, close string) error {
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

func (tl *List) RemoveIndex(n int) error {
	if n >= len(tl.elements) {
		return io.EOF
	}
	if n+1 <= len(tl.elements) {
		copy(tl.elements[n:], tl.elements[n+1:])
	}
	tl.elements = tl.elements[:len(tl.elements)-1]
	return nil
}

func (tl *List) Slice(start, end int) (*List, error) {
	if start < 0 || start >= len(tl.elements) {
		return nil, tl.Source().Errorf("start index %d out of bounds [%d..%d]", start, 0, len(tl.elements))
	}
	if end < 0 || end > len(tl.elements) {
		return nil, tl.Source().Errorf("end index %d out of bounds [%d..%d]", end, 0, len(tl.elements))
	}
	if start > end {
		return nil, tl.Source().Errorf("start index %d is greater than end index %d", start, end)
	}
	tl2 := List{
		source:   *tl.Source(),
		elements: make([]*Token, end-start),
	}
	copy(tl2.elements, tl.elements[start:end])
	return &tl2, nil
}

func (tl *List) WrapError(err error) error {
	if err == nil {
		return nil
	}
	return tl.Source().WrapError(err)
}
