package mibdb

import (
	"context"

	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type Oid struct {
	elements   []string
	metaTokens *mibtoken.List
	source     mibtoken.Position
}

var _ Definition = (*Oid)(nil)

func (value *Oid) read(ctx context.Context, s mibtoken.Queue) error {
	value.source = *s.Source()
	peek, err := s.LookAhead(0)
	if err != nil {
		return err
	}
	if peek.String() == "{" {
		elements, err := s.PopBlock("{", "}")
		if err != nil {
			return err
		}
		for !elements.IsEOF() {
			element, err := elements.Pop()
			if err != nil {
				return err
			}
			value.elements = append(value.elements, element.String())
		}
		return nil
	}
	value.elements = append(value.elements, peek.String())
	return nil
}

func (value *Oid) Source() mibtoken.Position {
	return value.source
}

type Value struct {
	elements   []string
	metaTokens *mibtoken.List
	source     mibtoken.Position
}

var _ Definition = (*Value)(nil)

func (value *Value) read(ctx context.Context, s mibtoken.Queue) error {
	value.source = *s.Source()
	peek, err := s.LookAhead(0)
	if err != nil {
		return err
	}
	if peek.String() == "{" {
		elements, err := s.PopBlock("{", "}")
		if err != nil {
			return err
		}
		for !elements.IsEOF() {
			element, err := elements.Pop()
			if err != nil {
				return err
			}
			value.elements = append(value.elements, element.String())
		}
		return nil
	}
	value.elements = append(value.elements, peek.String())
	return nil
}

func (value *Value) Source() mibtoken.Position {
	return value.source
}
