package mibdb

import (
	"context"
	"strconv"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type Value interface {
	Definition
	compile(ctx context.Context) error
}

// ------------------------------------

type OidValue struct {
	elements   []string
	metaTokens *mibtoken.List
	source     mibtoken.Source
	compiled   asn1go.OID
}

var _ Definition = (*OidValue)(nil)

func (value *OidValue) readOid(_ context.Context, s mibtoken.Reader) error {
	value.source = *s.Source()
	peek, err := s.LookAhead(0)
	if err != nil {
		return err
	}
	if peek.String() == "{" {
		elements, err := mibtoken.ReadBlock(s, "{", "}")
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

func (value *OidValue) Source() mibtoken.Source {
	return value.source
}

func (value *OidValue) compile(ctx context.Context) error {
	if len(value.compiled) > 0 {
		return nil
	}
	for _, element := range value.elements {

		n, err := strconv.Atoi(element)
		if err == nil {
			value.compiled = append(value.compiled, n)
			continue
		}
		other, err := Lookup[Definition](ctx, element)
		if err != nil {
			value.compiled = nil
			return value.source.WrapError(err)
		}
		otherOID, ok := other.(*OidValue)
		if !ok {
			value.compiled = nil
			return value.source.Errorf("expected OID but got %T", other)
		}
		err = otherOID.compile(ctx)
		if err != nil {
			value.compiled = nil
			return value.source.WrapError(err)
		}
		value.compiled = nil
		value.compiled = append(value.compiled, otherOID.compiled...)
	}
	return nil
}

// ------------------------------------

type ConstantValue struct {
	elements   []string
	metaTokens *mibtoken.List
	source     mibtoken.Source
}

var _ Definition = (*ConstantValue)(nil)

func (value *ConstantValue) read(_ context.Context, s mibtoken.Reader) error {
	value.source = *s.Source()
	peek, err := s.LookAhead(0)
	if err != nil {
		return err
	}
	if peek.String() == "{" {
		elements, err := mibtoken.ReadBlock(s, "{", "}")
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

func (value *ConstantValue) Source() mibtoken.Source {
	return value.source
}

func (value *ConstantValue) compile(_ context.Context) error {
	return nil
}

// ------------------------------------

type structureValue struct {
	source     mibtoken.Source
	vType      Type
	metaTokens *mibtoken.List
	fields     map[string]Value
}

func (value *structureValue) Source() mibtoken.Source {
	return value.source
}

func (value *structureValue) compile(ctx context.Context) error {
	return nil
}

// ------------------------------------

type goValue[T any] struct {
	value  T
	source mibtoken.Source
}

var _ Value = (*goValue[string])(nil)

func (value *goValue[T]) Source() mibtoken.Source {
	return value.source
}

func (value *goValue[T]) compile(ctx context.Context) error {
	return nil
}
