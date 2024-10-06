package mibtoken

import (
	"errors"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
)

type Reader interface {
	Pop() (*Token, error)
	LookAhead(n int) (*Token, error)
	IsEOF() bool
	Source() *Source
}

func ReadUntil(r Reader, end string) (*List, error) {
	tokens := &List{
		source: *r.Source(),
	}
	for {
		tok, err := r.Pop()
		if err != nil {
			var general *asn1error.General
			if errors.As(err, &general) {
				return nil, general
			}
			return nil, err
		}
		if tok.IsText(end) {
			return tokens, nil
		}
		tokens.AppendTokens(tok)
	}
}

func ReadExpected(s Reader, expectedTexts ...string) error {
	for _, expectedText := range expectedTexts {
		actual, err := s.Pop()
		if err != nil {
			return err
		}
		if actual.String() != expectedText {
			return asn1error.NewUnexpectedError(expectedText, actual.String(), "token")
		}
	}
	return nil
}

func ReadBlock(s Reader, start, end string) (*List, error) {
	level := 0
	block := &List{source: *s.Source()}
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

func WrapError(r Reader, err error) error {
	return r.Source().WrapError(err)
}
